<script setup lang="ts">
import { computed, onBeforeUnmount, shallowRef, watch } from 'vue'
import { Compartment, EditorState } from '@codemirror/state'
import { autocompletion, type Completion, type CompletionContext } from '@codemirror/autocomplete'
import { history, historyKeymap, indentWithTab, standardKeymap } from '@codemirror/commands'
import { highlightSelectionMatches, searchKeymap } from '@codemirror/search'
import {
  drawSelection,
  dropCursor,
  EditorView,
  highlightActiveLine,
  highlightActiveLineGutter,
  keymap,
  lineNumbers,
} from '@codemirror/view'
import { BookOpenText, Clock, FileText, MapPin, PencilLine } from 'lucide-vue-next'
import type { ProjectDTO, SceneDTO } from '../api/types'

type SaveStatus = 'saved' | 'dirty' | 'saving' | 'failed'

const props = defineProps<{
  scene: SceneDTO | null
  project: ProjectDTO | null
  modelValue: string
  saveStatus: SaveStatus
  loading?: boolean
  readonly?: boolean
  characterSuggestions?: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorHostRef = shallowRef<HTMLDivElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)

const editableCompartment = new Compartment()
const readOnlyCompartment = new Compartment()
const characterCompletionCompartment = new Compartment()

const normalizeCharacterSuggestions = (characters: string[] = []) =>
  Array.from(new Set(characters.map((character) => character.trim()).filter(Boolean)))

const buildCharacterCompletionExtension = (characters: string[]) =>
  autocompletion({
    activateOnTyping: true,
    maxRenderedOptions: 8,
    override: [
      (context: CompletionContext) => {
        if (characters.length === 0) return null

        const line = context.state.doc.lineAt(context.pos)
        const linePrefix = line.text.slice(0, context.pos - line.from)
        const match = linePrefix.match(/^[\s　]*([\u4e00-\u9fa5A-Za-z0-9_·]{0,16})$/)
        if (!match) return null

        const token = match[1] ?? ''
        if (!context.explicit && token.length === 0) return null

        const from = context.pos - token.length
        const options = characters
          .filter((character) => !token || character.includes(token))
          .map(
            (character): Completion => ({
              label: character,
              type: 'variable',
              detail: '当前场景人物',
              apply: `${character}：`,
            }),
          )

        if (options.length === 0) return null

        return {
          from,
          options,
          validFor: /^[\u4e00-\u9fa5A-Za-z0-9_·]*$/,
        }
      },
    ],
  })

const normalizedCharacterSuggestions = computed(() =>
  normalizeCharacterSuggestions(props.characterSuggestions),
)

const isEditorEditable = computed(() => Boolean(props.scene) && !props.readonly)

const scriptEditorTheme = EditorView.theme({
  '&': {
    height: '100%',
    minHeight: '0',
    color: 'var(--color-ink)',
    backgroundColor: '#fff',
    fontFamily:
      '"Noto Sans Mono CJK SC", "Sarasa Mono SC", "Menlo", "Consolas", "Courier New", var(--font-ui)',
    fontSize: '15px',
  },
  '.cm-scroller': {
    height: '100%',
    overflow: 'auto',
    lineHeight: '28px',
  },
  '.cm-content': {
    minHeight: '100%',
    padding: '18px 22px 28px',
    caretColor: 'var(--color-sage)',
  },
  '.cm-line': {
    padding: '0',
  },
  '.cm-gutters': {
    borderRight: '1px solid #e7ece9',
    backgroundColor: '#f7f9f7',
    color: '#95a19e',
    fontFamily: '"Menlo", "Consolas", "Courier New", monospace',
    fontSize: '12px',
  },
  '.cm-gutterElement': {
    minWidth: '42px',
    padding: '0 12px 0 0',
    lineHeight: '28px',
  },
  '.cm-activeLineGutter': {
    color: 'var(--color-sage)',
    backgroundColor: 'rgba(47, 118, 100, 0.08)',
  },
  '.cm-activeLine': {
    backgroundColor: 'rgba(47, 118, 100, 0.045)',
  },
  '.cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'rgba(47, 118, 100, 0.18) !important',
  },
  '.cm-searchMatch': {
    backgroundColor: 'rgba(141, 73, 56, 0.16)',
    outline: '1px solid rgba(141, 73, 56, 0.2)',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '&.cm-focused .cm-cursor': {
    borderLeftColor: 'var(--color-sage)',
  },
  '&.cm-editor-disabled': {
    backgroundColor: '#fbfcfb',
  },
  '&.cm-editor-disabled .cm-content': {
    cursor: 'not-allowed',
    color: 'rgba(89, 105, 102, 0.72)',
  },
  '.cm-tooltip.cm-tooltip-autocomplete': {
    overflow: 'hidden',
    border: '1px solid rgba(220, 227, 223, 0.95)',
    borderRadius: '8px',
    backgroundColor: 'var(--color-paper)',
    boxShadow: '0 14px 30px rgba(23, 33, 31, 0.12)',
  },
  '.cm-tooltip-autocomplete ul': {
    fontFamily: 'var(--font-ui)',
    fontSize: '13px',
  },
  '.cm-tooltip-autocomplete ul li': {
    display: 'flex',
    alignItems: 'center',
    minHeight: '30px',
    padding: '4px 10px',
    color: 'var(--color-ink)',
  },
  '.cm-tooltip-autocomplete ul li[aria-selected]': {
    backgroundColor: 'var(--color-sage-soft)',
    color: 'var(--color-sage)',
  },
  '.cm-completionDetail': {
    color: 'var(--color-muted)',
    fontSize: '12px',
  },
})

const scriptEditorExtensions = [
  lineNumbers(),
  highlightActiveLineGutter(),
  history(),
  drawSelection(),
  dropCursor(),
  EditorState.allowMultipleSelections.of(true),
  EditorView.lineWrapping,
  highlightActiveLine(),
  highlightSelectionMatches(),
  scriptEditorTheme,
  keymap.of([indentWithTab, ...standardKeymap, ...historyKeymap, ...searchKeymap]),
  EditorView.updateListener.of((update) => {
    if (!update.docChanged) return
    emit('update:modelValue', update.state.doc.toString())
  }),
]

const sceneLabel = computed(() => {
  if (!props.scene) return ''
  const parts = [`第 ${props.scene.sceneNo} 场`]
  if (props.scene.location) parts.push(props.scene.location)
  if (props.scene.timeText) parts.push(props.scene.timeText)
  return parts.join('，')
})

const saveStatusMeta = computed(() => {
  const map: Record<SaveStatus, { text: string; className: string }> = {
    saved: { text: '文本已同步', className: 'saved' },
    dirty: { text: '有未保存修改', className: 'dirty' },
    saving: { text: '正在保存', className: 'saving' },
    failed: { text: '保存失败', className: 'failed' },
  }

  return map[props.saveStatus]
})

const editorStatusText = computed(() => {
  if (props.loading) return '正在加载场次'
  if (props.readonly && props.scene) return '生成中，暂时锁定'
  return saveStatusMeta.value.text
})

const wordCount = computed(() => props.modelValue.replace(/\s/g, '').length)

const lineCount = computed(() => {
  if (!props.modelValue) return 0
  return props.modelValue.split('\n').length
})

const projectName = computed(() => props.project?.title ?? '剧本工程')

const novelName = computed(() => props.project?.novelTitle ?? '原小说未命名')

const updatedAtText = computed(() => {
  if (!props.scene?.updatedAt) return ''

  const date = new Date(props.scene.updatedAt)
  if (Number.isNaN(date.getTime())) return props.scene.updatedAt

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
})

const createEditor = (host: HTMLDivElement) => {
  editorView.value?.destroy()

  editorView.value = new EditorView({
    parent: host,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        editableCompartment.of(EditorView.editable.of(isEditorEditable.value)),
        readOnlyCompartment.of(EditorState.readOnly.of(!isEditorEditable.value)),
        characterCompletionCompartment.of(
          buildCharacterCompletionExtension(normalizedCharacterSuggestions.value),
        ),
        ...scriptEditorExtensions,
      ],
    }),
  })
}

watch(editorHostRef, (host) => {
  if (host) createEditor(host)
})

watch(
  () => props.modelValue,
  (value) => {
    const view = editorView.value
    if (!view || value === view.state.doc.toString()) return

    view.dispatch({
      changes: {
        from: 0,
        to: view.state.doc.length,
        insert: value,
      },
    })
  },
)

watch(
  () => isEditorEditable.value,
  (isEditable) => {
    editorView.value?.dispatch({
      effects: [
        editableCompartment.reconfigure(EditorView.editable.of(isEditable)),
        readOnlyCompartment.reconfigure(EditorState.readOnly.of(!isEditable)),
      ],
    })
  },
)

watch(
  () => normalizedCharacterSuggestions.value.join('\u0000'),
  () => {
    editorView.value?.dispatch({
      effects: characterCompletionCompartment.reconfigure(
        buildCharacterCompletionExtension(normalizedCharacterSuggestions.value),
      ),
    })
  },
)

onBeforeUnmount(() => {
  editorView.value?.destroy()
  editorView.value = null
})
</script>

<template>
  <div class="script-editor-panel">
    <header class="editor-titlebar">
      <div class="script-editor-title">
        <span class="scene-index">SCENE {{ scene?.sceneNo ?? '--' }}</span>
        <h1>{{ scene?.title ?? '选择一个场次开始编辑' }}</h1>
      </div>

      <div
        class="script-editor-status"
        :class="[saveStatusMeta.className, { locked: readonly && scene }]"
      >
        <span class="status-dot" aria-hidden="true" />
        {{ editorStatusText }}
      </div>
    </header>

    <div v-if="scene" class="scene-context-bar">
      <span>
        <n-icon><PencilLine /></n-icon>
        {{ sceneLabel }}
      </span>
      <span v-if="scene.location">
        <n-icon><MapPin /></n-icon>
        {{ scene.location }}
      </span>
      <span v-if="scene.timeText">
        <n-icon><Clock /></n-icon>
        {{ scene.timeText }}
      </span>
    </div>

    <div class="editor-toolbar" aria-label="编辑器状态">
      <span>
        <n-icon><BookOpenText /></n-icon>
        {{ novelName }}
      </span>
      <span>
        <n-icon><FileText /></n-icon>
        {{ projectName }}
      </span>
      <span class="toolbar-spacer" aria-hidden="true" />
      <strong>{{ wordCount }} 字</strong>
      <strong>{{ lineCount }} 行</strong>
    </div>

    <section class="editor-text-surface" aria-label="剧本文本编辑区">
      <div
        ref="editorHostRef"
        class="script-codemirror"
        :class="{ 'is-empty': !modelValue, 'is-disabled': !scene || readonly }"
        :data-placeholder="
          scene
            ? readonly
              ? '增量生成中，编辑器暂时锁定。'
              : '在这里继续打磨剧本文本。'
            : '选择左侧场次后，在这里继续打磨剧本文本。'
        "
      />
    </section>

    <footer class="script-editor-footer">
      <span>{{ scene ? (readonly ? '增量生成中，暂时禁止编辑' : '正在编辑当前场次') : '等待选择场次' }}</span>
      <span v-if="updatedAtText">最近保存：{{ updatedAtText }}</span>
    </footer>
  </div>
</template>

<style scoped>
.script-editor-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 18px 38px rgba(23, 33, 31, 0.07);
}

.editor-titlebar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  min-height: 64px;
  padding: 0 18px;
  border-bottom: 1px solid rgba(220, 227, 223, 0.86);
  background: #fff;
}

.script-editor-title {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.scene-index {
  display: grid;
  flex-shrink: 0;
  min-width: 72px;
  height: 28px;
  place-items: center;
  border: 1px solid rgba(47, 118, 100, 0.2);
  border-radius: 8px;
  color: var(--color-sage);
  background: var(--color-sage-soft);
  font-size: 12px;
  font-weight: 900;
}

.script-editor-title h1 {
  min-width: 0;
  margin: 0;
  color: var(--color-ink);
  font-family: var(--font-ui);
  font-size: 17px;
  font-weight: 800;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.script-editor-status {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 7px;
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--color-muted);
  background: rgba(244, 246, 244, 0.86);
  font-size: 12px;
  font-weight: 800;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: currentColor;
}

.script-editor-status.saved {
  color: var(--color-sage);
  background: var(--color-sage-soft);
}

.script-editor-status.dirty,
.script-editor-status.saving {
  color: var(--color-brick);
  background: rgba(141, 73, 56, 0.08);
}

.script-editor-status.failed {
  color: #b42318;
  background: rgba(180, 35, 24, 0.08);
}

.script-editor-status.locked {
  color: #7a5b23;
  background: rgba(196, 148, 56, 0.12);
}

.scene-context-bar {
  display: flex;
  flex-shrink: 0;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px 14px;
  border-bottom: 1px solid rgba(220, 227, 223, 0.72);
  background: #fafbf9;
}

.scene-context-bar span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 26px;
  padding: 0 8px;
  border: 1px solid rgba(47, 118, 100, 0.16);
  border-radius: 8px;
  color: var(--color-muted);
  background: #fff;
  font-size: 12px;
  font-weight: 700;
}

.editor-toolbar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 14px;
  border-bottom: 1px solid rgba(220, 227, 223, 0.86);
  color: var(--color-muted);
  background: #f7f9f7;
  font-size: 12px;
  font-weight: 700;
}

.editor-toolbar span,
.editor-toolbar strong {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  white-space: nowrap;
}

.editor-toolbar span:not(.toolbar-spacer) {
  overflow: hidden;
  text-overflow: ellipsis;
}

.toolbar-spacer {
  flex: 1;
}

.editor-toolbar strong {
  color: var(--color-ink);
  font-weight: 800;
}

.editor-text-surface {
  flex: 1;
  min-height: 0;
  background: #fff;
}

.script-codemirror {
  position: relative;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
}

.script-codemirror.is-empty::after {
  position: absolute;
  top: 18px;
  left: 76px;
  z-index: 1;
  color: rgba(89, 105, 102, 0.58);
  content: attr(data-placeholder);
  font-size: 15px;
  line-height: 28px;
  pointer-events: none;
}

.script-codemirror.is-disabled::after {
  color: rgba(89, 105, 102, 0.48);
}

.script-editor-footer {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 14px;
  min-height: 40px;
  padding: 0 22px;
  border-top: 1px solid rgba(220, 227, 223, 0.86);
  color: var(--color-muted);
  background: #f7f9f7;
  font-size: 12px;
  font-weight: 700;
}

@media (max-width: 720px) {
  .editor-titlebar {
    min-height: auto;
    padding: 14px;
  }

  .script-editor-title {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }

  .script-editor-status {
    align-self: flex-start;
  }

  .editor-toolbar {
    flex-wrap: wrap;
    padding-block: 8px;
  }

  .toolbar-spacer {
    display: none;
  }

  .script-codemirror :deep(.cm-content) {
    padding-inline: 14px;
    font-size: 14px;
  }

  .script-codemirror :deep(.cm-gutterElement) {
    min-width: 34px;
    padding-right: 9px;
  }

  .script-codemirror.is-empty::after {
    left: 54px;
    font-size: 14px;
  }
}
</style>
