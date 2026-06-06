<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { Check, RefreshCw, WandSparkles, X } from 'lucide-vue-next'
import { scenesApi, type GenerateSceneRegenerationResponse, type SceneRegenerationMode } from '../../api'

const REGENERATE_COST = 80

const props = withDefaults(
  defineProps<{
    show: boolean
    sceneId: string | null
    sceneTitle?: string
    sceneContent?: string
    aiPoints?: number
    disabled?: boolean
    applying?: boolean
  }>(),
  {
    sceneTitle: '',
    sceneContent: '',
    aiPoints: 0,
    disabled: false,
    applying: false,
  },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  apply: [content: string]
  refreshUser: [remainingPoints: number]
}>()

interface RegenerateShortcut {
  label: string
  instruction: string
  mode: SceneRegenerationMode
}

const message = useMessage()
const instruction = ref('')
const mode = ref<SceneRegenerationMode>('replace')
const loading = ref(false)
const result = ref<GenerateSceneRegenerationResponse | null>(null)
const errorMessage = ref('')

const modeOptions: Array<{ label: string; value: SceneRegenerationMode }> = [
  { label: '重写整场', value: 'replace' },
  { label: '润色', value: 'polish' },
  { label: '缩短', value: 'shorten' },
  { label: '扩写', value: 'expand' },
]

const shortcuts: RegenerateShortcut[] = [
  { label: '减少旁白', instruction: '减少旁白，让动作和对白承担更多信息。', mode: 'polish' },
  { label: '增强冲突', instruction: '增强人物之间的冲突和试探，让本场更有张力。', mode: 'replace' },
  { label: '对白自然', instruction: '让对白更自然，减少直白说明，保留潜台词。', mode: 'polish' },
  { label: '增加动作', instruction: '增加可拍摄的动作描写和视觉细节。', mode: 'expand' },
  { label: '缩短内容', instruction: '缩短本场内容，保留关键剧情和必要对白。', mode: 'shorten' },
]

const originalContent = computed(() => result.value?.originalContent ?? props.sceneContent)
const regeneratedContent = computed(() => result.value?.regeneratedContent ?? '')
const hasEnoughPoints = computed(() => props.aiPoints >= REGENERATE_COST)
const canGenerate = computed(
  () =>
    Boolean(props.sceneId) &&
    Boolean(props.sceneContent.trim()) &&
    hasEnoughPoints.value &&
    !props.disabled &&
    !loading.value &&
    !props.applying,
)

const resetState = () => {
  instruction.value = ''
  mode.value = 'replace'
  loading.value = false
  result.value = null
  errorMessage.value = ''
}

const handleClose = () => {
  if (loading.value || props.applying) return
  emit('update:show', false)
}

const handleModalShowChange = (value: boolean) => {
  if (!value) {
    handleClose()
    return
  }
  emit('update:show', true)
}

const applyShortcut = (shortcut: RegenerateShortcut) => {
  instruction.value = shortcut.instruction
  mode.value = shortcut.mode
}

const handleGenerate = async () => {
  if (!props.sceneId || loading.value || props.disabled) return

  if (!props.sceneContent.trim()) {
    message.warning('当前场次内容为空，无法重新生成')
    return
  }

  if (!hasEnoughPoints.value) {
    message.warning('AI 点数不足，无法重新生成')
    return
  }

  loading.value = true
  errorMessage.value = ''

  try {
    const nextResult = await scenesApi.regenerate(props.sceneId, {
      instruction: instruction.value.trim(),
      mode: mode.value,
    })
    result.value = nextResult
    emit('refreshUser', nextResult.remainingPoints)
    message.success('本场新版本已生成')
  } catch (error) {
    const text = error instanceof Error ? error.message : '本场重新生成失败，请稍后重试'
    errorMessage.value = text
    message.error(text)
  } finally {
    loading.value = false
  }
}

const handleApply = () => {
  if (!result.value || loading.value || props.applying) return
  emit('apply', result.value.regeneratedContent)
}

watch(
  () => props.show,
  (show) => {
    if (show) resetState()
  },
)

watch(
  () => props.sceneId,
  () => {
    if (props.show) resetState()
  },
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    class="scene-regenerate-modal"
    :bordered="false"
    :closable="!loading && !applying"
    :mask-closable="!loading && !applying"
    @update:show="handleModalShowChange"
  >
    <template #header>
      <div class="regenerate-modal-title">
        <span class="regenerate-title-icon">
          <n-icon><WandSparkles /></n-icon>
        </span>
        <div>
          <h2>重新生成本场</h2>
          <p>{{ sceneTitle || '当前场次' }}</p>
        </div>
      </div>
    </template>

    <div class="regenerate-modal-body">
      <section class="regenerate-control-panel">
        <div class="regenerate-control-head">
          <div>
            <strong>改写要求</strong>
            <span>本次重新生成将消耗 80 AI 点数。</span>
          </div>
          <n-tag :bordered="false" :type="hasEnoughPoints ? 'success' : 'warning'">
            当前 {{ aiPoints }} 点
          </n-tag>
        </div>

        <n-radio-group v-model:value="mode" name="scene-regenerate-mode" :disabled="loading || applying">
          <n-radio-button
            v-for="option in modeOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </n-radio-button>
        </n-radio-group>

        <n-input
          v-model:value="instruction"
          type="textarea"
          placeholder="例如：减少旁白，增强人物对白冲突。"
          :autosize="{ minRows: 3, maxRows: 5 }"
          :disabled="loading || applying"
        />

        <div class="regenerate-shortcuts" aria-label="快捷改写要求">
          <n-button
            v-for="shortcut in shortcuts"
            :key="shortcut.label"
            size="small"
            tertiary
            :disabled="loading || applying"
            @click="applyShortcut(shortcut)"
          >
            {{ shortcut.label }}
          </n-button>
        </div>

        <n-alert
          v-if="!hasEnoughPoints"
          type="warning"
          :bordered="false"
          class="regenerate-inline-alert"
        >
          AI 点数不足，无法发起本场重新生成。
        </n-alert>
        <n-alert
          v-else-if="errorMessage"
          type="error"
          :bordered="false"
          class="regenerate-inline-alert"
        >
          {{ errorMessage }}
        </n-alert>
      </section>

      <section class="regenerate-preview-grid" :class="{ 'has-result': result }">
        <article class="regenerate-preview-pane">
          <header>
            <strong>当前版本</strong>
            <span>{{ originalContent.replace(/\s/g, '').length }} 字</span>
          </header>
          <pre>{{ originalContent || '当前场次暂无内容。' }}</pre>
        </article>

        <article class="regenerate-preview-pane new-version">
          <header>
            <strong>AI 新版本</strong>
            <span v-if="regeneratedContent">{{ regeneratedContent.replace(/\s/g, '').length }} 字</span>
            <span v-else>等待生成</span>
          </header>
          <pre v-if="regeneratedContent">{{ regeneratedContent }}</pre>
          <n-empty v-else description="输入要求后开始生成" class="regenerate-empty">
            <template #extra>
              <span>生成后可在这里预览新版本，再决定是否应用。</span>
            </template>
          </n-empty>
        </article>
      </section>
    </div>

    <template #footer>
      <n-space justify="end">
        <n-button :disabled="loading || applying" @click="handleClose">
          <template #icon>
            <n-icon><X /></n-icon>
          </template>
          取消
        </n-button>
        <n-button
          secondary
          type="primary"
          :loading="loading"
          :disabled="!canGenerate"
          @click="handleGenerate"
        >
          <template #icon>
            <n-icon><RefreshCw /></n-icon>
          </template>
          {{ result ? '重新生成' : '开始生成' }}
        </n-button>
        <n-button
          type="primary"
          :loading="applying"
          :disabled="!result || loading || disabled"
          @click="handleApply"
        >
          <template #icon>
            <n-icon><Check /></n-icon>
          </template>
          应用新版本
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.regenerate-modal-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.regenerate-title-icon {
  display: grid;
  width: 42px;
  height: 42px;
  flex-shrink: 0;
  place-items: center;
  border-radius: 8px;
  color: var(--color-paper);
  background: var(--color-sage);
  font-size: 22px;
}

.regenerate-modal-title h2 {
  margin: 0;
  color: var(--color-ink);
  font-family: var(--font-display);
  font-size: 21px;
  line-height: 1.3;
}

.regenerate-modal-title p {
  margin: 3px 0 0;
  color: var(--color-muted);
  font-size: 13px;
  line-height: 1.45;
}

.regenerate-modal-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.regenerate-control-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.78);
}

.regenerate-control-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.regenerate-control-head strong,
.regenerate-control-head span {
  display: block;
}

.regenerate-control-head strong {
  color: var(--color-ink);
  font-size: 14px;
  line-height: 20px;
}

.regenerate-control-head span {
  margin-top: 2px;
  color: var(--color-muted);
  font-size: 12px;
  line-height: 18px;
}

.regenerate-shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.regenerate-inline-alert {
  margin-top: 0;
}

.regenerate-preview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  min-height: 360px;
}

.regenerate-preview-pane {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: #fff;
}

.regenerate-preview-pane.new-version {
  border-color: rgba(47, 118, 100, 0.32);
}

.regenerate-preview-pane header {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  min-height: 40px;
  padding: 0 12px;
  border-bottom: 1px solid var(--color-line);
  color: var(--color-muted);
  background: #f7f9f7;
  font-size: 12px;
  line-height: 18px;
}

.regenerate-preview-pane header strong {
  color: var(--color-ink);
  font-size: 13px;
}

.regenerate-preview-pane pre {
  flex: 1;
  min-height: 0;
  margin: 0;
  overflow: auto;
  padding: 14px;
  color: var(--color-ink);
  font-family:
    "Noto Sans Mono CJK SC", "Sarasa Mono SC", "Menlo", "Consolas", "Courier New",
    var(--font-ui);
  font-size: 13px;
  line-height: 1.8;
  white-space: pre-wrap;
}

.regenerate-empty {
  flex: 1;
  min-height: 0;
  justify-content: center;
}

.regenerate-empty span {
  color: var(--color-muted);
  font-size: 12px;
}

@media (max-width: 760px) {
  .regenerate-preview-grid {
    grid-template-columns: minmax(0, 1fr);
    min-height: 0;
  }

  .regenerate-preview-pane {
    min-height: 260px;
  }

  .regenerate-control-head {
    flex-direction: column;
  }
}
</style>

<style>
.scene-regenerate-modal {
  width: min(920px, calc(100vw - 32px)) !important;
}
</style>
