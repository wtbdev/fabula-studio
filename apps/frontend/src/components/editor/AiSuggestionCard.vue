<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { ClipboardCopy, EyeOff } from 'lucide-vue-next'
import type { SceneSuggestion } from '../../api'

const props = defineProps<{
  suggestion: SceneSuggestion
  dismissing?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  dismiss: [suggestionId: string]
}>()

const message = useMessage()
const manualCopyVisible = ref(false)
const manualCopyRef = ref<HTMLTextAreaElement | null>(null)

const typeMeta: Record<SceneSuggestion['type'], { label: string; className: string }> = {
  dialogue: { label: '对白', className: 'dialogue' },
  conflict: { label: '冲突', className: 'conflict' },
  rhythm: { label: '节奏', className: 'rhythm' },
  character: { label: '动机', className: 'character' },
  structure: { label: '结构', className: 'structure' },
  visual: { label: '画面', className: 'visual' },
}

const currentTypeMeta = computed(() => typeMeta[props.suggestion.type])

const copyText = computed(() => {
  const lines = [
    `【${currentTypeMeta.value.label}】${props.suggestion.title}`,
    `问题：${props.suggestion.problem}`,
    `原因：${props.suggestion.reason}`,
    `建议：${props.suggestion.suggestion}`,
  ]

  if (props.suggestion.applyText) {
    lines.push(`可参考文本：${props.suggestion.applyText}`)
  }

  return lines.join('\n')
})

const fallbackCopy = (text: string) => {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.top = '0'
  textarea.style.left = '0'
  textarea.style.width = '1px'
  textarea.style.height = '1px'
  textarea.style.opacity = '0'
  textarea.style.pointerEvents = 'none'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  const copied = document.execCommand('copy')
  document.body.removeChild(textarea)
  return copied
}

const handleCopy = async () => {
  try {
    let copied = false

    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(copyText.value)
        copied = true
      } catch {
        copied = false
      }
    }

    if (!copied) {
      copied = fallbackCopy(copyText.value)
    }

    if (!copied) {
      manualCopyVisible.value = true
      await nextTick()
      manualCopyRef.value?.focus()
      manualCopyRef.value?.select()
      message.warning('浏览器限制自动复制，已选中建议内容')
      return
    }

    message.success('已复制建议内容')
  } catch {
    message.error('复制失败，请手动复制')
  }
}
</script>

<template>
  <article class="ai-suggestion-card" :class="`type-${currentTypeMeta.className}`">
    <header class="suggestion-card-head">
      <n-tag size="small" :bordered="false" class="suggestion-type">
        {{ currentTypeMeta.label }}
      </n-tag>
      <h3>{{ suggestion.title }}</h3>
    </header>

    <dl class="suggestion-body">
      <div>
        <dt>问题</dt>
        <dd>{{ suggestion.problem }}</dd>
      </div>
      <div>
        <dt>原因</dt>
        <dd>{{ suggestion.reason }}</dd>
      </div>
      <div>
        <dt>建议</dt>
        <dd>{{ suggestion.suggestion }}</dd>
      </div>
    </dl>

    <blockquote v-if="suggestion.applyText" class="suggestion-apply">
      {{ suggestion.applyText }}
    </blockquote>

    <footer class="suggestion-actions">
      <n-button size="tiny" secondary :disabled="disabled" @click="handleCopy">
        <template #icon>
          <n-icon><ClipboardCopy /></n-icon>
        </template>
        复制
      </n-button>
      <n-button
        size="tiny"
        quaternary
        type="warning"
        :loading="dismissing"
        :disabled="disabled"
        @click="emit('dismiss', suggestion.id)"
      >
        <template #icon>
          <n-icon><EyeOff /></n-icon>
        </template>
        忽略
      </n-button>
    </footer>

    <textarea
      v-if="manualCopyVisible"
      ref="manualCopyRef"
      class="manual-copy-box"
      readonly
      :value="copyText"
      aria-label="手动复制建议内容"
      @focus="($event.target as HTMLTextAreaElement).select()"
    />
  </article>
</template>

<style scoped>
.ai-suggestion-card {
  position: relative;
  overflow: hidden;
  padding: 14px;
  border: 1px solid rgba(220, 227, 223, 0.94);
  border-radius: 8px;
  background: #fffdf8;
  box-shadow: 0 12px 24px rgba(23, 33, 31, 0.05);
}

.ai-suggestion-card::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  background: var(--suggestion-accent, var(--color-sage));
  content: "";
}

.type-dialogue {
  --suggestion-accent: #2f7664;
}

.type-conflict {
  --suggestion-accent: #8d4938;
}

.type-rhythm {
  --suggestion-accent: #c49438;
}

.type-character {
  --suggestion-accent: #527b86;
}

.type-structure {
  --suggestion-accent: #6f6b9f;
}

.type-visual {
  --suggestion-accent: #98724d;
}

.suggestion-card-head {
  display: grid;
  gap: 9px;
}

.suggestion-type {
  justify-self: start;
  color: var(--suggestion-accent);
  background: color-mix(in srgb, var(--suggestion-accent) 12%, transparent);
  font-weight: 800;
}

.suggestion-card-head h3 {
  margin: 0;
  color: var(--color-ink);
  font-size: 15px;
  font-weight: 900;
  line-height: 1.45;
}

.suggestion-body {
  display: grid;
  gap: 10px;
  margin: 12px 0 0;
}

.suggestion-body div {
  display: grid;
  gap: 4px;
}

.suggestion-body dt {
  color: var(--color-muted);
  font-size: 12px;
  font-weight: 800;
}

.suggestion-body dd {
  margin: 0;
  color: var(--color-ink);
  font-size: 13px;
  line-height: 1.65;
}

.suggestion-apply {
  margin: 12px 0 0;
  padding: 10px 11px;
  border-left: 3px solid var(--suggestion-accent);
  border-radius: 0 8px 8px 0;
  color: var(--color-ink);
  background: rgba(244, 246, 244, 0.72);
  font-size: 13px;
  font-weight: 700;
  line-height: 1.65;
}

.suggestion-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.manual-copy-box {
  width: 100%;
  min-height: 92px;
  margin-top: 10px;
  padding: 9px 10px;
  resize: vertical;
  border: 1px solid rgba(141, 73, 56, 0.24);
  border-radius: 8px;
  color: var(--color-ink);
  background: rgba(255, 253, 248, 0.88);
  font-family: var(--font-ui);
  font-size: 12px;
  line-height: 1.55;
}
</style>
