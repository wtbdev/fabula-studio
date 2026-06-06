<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { AlertCircle, Lightbulb, RefreshCw, Sparkles } from 'lucide-vue-next'
import { suggestionsApi, type SceneSuggestion } from '../../api'
import AiSuggestionCard from './AiSuggestionCard.vue'

const props = withDefaults(
  defineProps<{
    projectId: string | null
    sceneId: string | null
    sceneTitle?: string
    sceneContent?: string
    disabled?: boolean
  }>(),
  {
    sceneTitle: '',
    sceneContent: '',
    disabled: false,
  },
)

const emit = defineEmits<{
  refreshUser: [remainingPoints: number]
}>()

const message = useMessage()
const suggestionsBySceneId = reactive<Record<string, SceneSuggestion[]>>({})
const loadingSceneIds = ref<string[]>([])
const generating = ref(false)
const dismissingId = ref<string | null>(null)
const errorMessage = ref('')

const currentSuggestions = computed(() =>
  props.sceneId ? suggestionsBySceneId[props.sceneId] ?? [] : [],
)

const isLoading = computed(() =>
  Boolean(props.sceneId && loadingSceneIds.value.includes(props.sceneId)),
)

const canGenerate = computed(() => Boolean(props.projectId && props.sceneId) && !props.disabled)

const setSceneLoading = (sceneId: string, loading: boolean) => {
  loadingSceneIds.value = loading
    ? Array.from(new Set([...loadingSceneIds.value, sceneId]))
    : loadingSceneIds.value.filter((id) => id !== sceneId)
}

const fetchSuggestions = async () => {
  if (!props.sceneId) return

  const sceneId = props.sceneId
  setSceneLoading(sceneId, true)
  errorMessage.value = ''

  try {
    const suggestions = await suggestionsApi.list(sceneId)
    if (props.sceneId === sceneId) {
      suggestionsBySceneId[sceneId] = suggestions
    }
  } catch (error) {
    if (props.sceneId === sceneId) {
      errorMessage.value = error instanceof Error ? error.message : '建议加载失败，请稍后重试'
    }
  } finally {
    setSceneLoading(sceneId, false)
  }
}

const handleGenerate = async () => {
  if (!props.sceneId || generating.value || props.disabled) return

  const content = props.sceneContent.trim()
  if (!content) {
    message.warning('当前场次没有可分析内容')
    return
  }

  const sceneId = props.sceneId
  generating.value = true
  errorMessage.value = ''

  try {
    const result = await suggestionsApi.generate(sceneId, {
      content,
      count: 3,
    })
    suggestionsBySceneId[sceneId] = result.suggestions
    emit('refreshUser', result.remainingPoints)
    message.success(`已生成 ${result.suggestions.length} 条 AI 建议`)
  } catch (error) {
    const text = error instanceof Error ? error.message : '建议生成失败，请重新尝试'
    errorMessage.value = text
    message.error(text)
  } finally {
    generating.value = false
  }
}

const handleDismiss = async (suggestionId: string) => {
  if (!props.sceneId || dismissingId.value || props.disabled) return

  const sceneId = props.sceneId
  dismissingId.value = suggestionId

  try {
    await suggestionsApi.update(suggestionId, { status: 'dismissed' })
    suggestionsBySceneId[sceneId] = currentSuggestions.value.filter(
      (suggestion) => suggestion.id !== suggestionId,
    )
    message.success('已忽略建议')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '建议状态更新失败')
  } finally {
    dismissingId.value = null
  }
}

watch(
  () => props.sceneId,
  () => {
    void fetchSuggestions()
  },
  { immediate: true },
)
</script>

<template>
  <section class="ai-suggestion-panel">
    <header class="suggestion-panel-head">
      <span class="suggestion-panel-icon">
        <n-icon><Lightbulb /></n-icon>
      </span>
      <div>
        <h2>AI 场景建议</h2>
        <p>{{ sceneTitle || '选择场次后查看建议' }}</p>
      </div>
      <n-button
        circle
        quaternary
        size="small"
        aria-label="刷新建议"
        :loading="isLoading"
        :disabled="!sceneId || disabled"
        @click="fetchSuggestions"
      >
        <template #icon>
          <n-icon><RefreshCw /></n-icon>
        </template>
      </n-button>
    </header>

    <div class="suggestion-generate-box">
      <div>
        <strong>让 AI 检查当前场次</strong>
        <span>每次消耗 30 点，返回对白、冲突、节奏等具体建议。</span>
      </div>
      <n-popconfirm
        positive-text="生成建议"
        negative-text="取消"
        :show-icon="false"
        @positive-click="handleGenerate"
      >
        <template #trigger>
          <n-button
            size="small"
            type="primary"
            secondary
            :loading="generating"
            :disabled="!canGenerate"
          >
            <template #icon>
              <n-icon><Sparkles /></n-icon>
            </template>
            生成 AI 建议
          </n-button>
        </template>
        本次生成建议将消耗 30 AI 点数，是否继续？
      </n-popconfirm>
    </div>

    <n-alert v-if="errorMessage" type="error" class="suggestion-alert" :bordered="false">
      <template #icon>
        <n-icon><AlertCircle /></n-icon>
      </template>
      {{ errorMessage }}
    </n-alert>

    <n-spin :show="isLoading" class="suggestion-spin">
      <div v-if="currentSuggestions.length" class="suggestion-list">
        <AiSuggestionCard
          v-for="suggestion in currentSuggestions"
          :key="suggestion.id"
          :suggestion="suggestion"
          :dismissing="dismissingId === suggestion.id"
          :disabled="disabled"
          @dismiss="handleDismiss"
        />
      </div>

      <n-empty
        v-else
        description="当前场次暂无 AI 建议"
        class="suggestion-empty"
      >
        <template #extra>
          <p>点击生成建议，让 AI 帮你检查对白、节奏和人物动机。</p>
        </template>
      </n-empty>
    </n-spin>
  </section>
</template>

<style scoped>
.ai-suggestion-panel {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  padding: 16px;
}

.suggestion-panel-head {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
}

.suggestion-panel-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  color: var(--color-paper);
  background: var(--color-brick);
  font-size: 21px;
}

.suggestion-panel-head h2 {
  margin: 0;
  color: var(--color-ink);
  font-family: var(--font-display);
  font-size: 18px;
  line-height: 1.35;
}

.suggestion-panel-head p {
  overflow: hidden;
  margin: 4px 0 0;
  color: var(--color-muted);
  font-size: 12px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.suggestion-generate-box {
  display: grid;
  gap: 12px;
  margin-top: 16px;
  padding: 13px;
  border: 1px solid rgba(141, 73, 56, 0.18);
  border-radius: 8px;
  background: rgba(141, 73, 56, 0.06);
}

.suggestion-generate-box div {
  display: grid;
  gap: 4px;
}

.suggestion-generate-box strong {
  color: var(--color-ink);
  font-size: 13px;
  font-weight: 900;
}

.suggestion-generate-box span {
  color: var(--color-muted);
  font-size: 12px;
  line-height: 1.55;
}

.suggestion-alert {
  margin-top: 12px;
}

.suggestion-spin {
  flex: 1;
  min-height: 0;
  margin-top: 14px;
}

.suggestion-spin :deep(.n-spin-content) {
  min-height: 100%;
}

.suggestion-list {
  display: grid;
  gap: 12px;
  padding-bottom: 2px;
}

.suggestion-empty {
  padding: 34px 6px 20px;
}

.suggestion-empty p {
  max-width: 210px;
  margin: 0 auto;
  color: var(--color-muted);
  font-size: 12px;
  line-height: 1.65;
  text-align: center;
}
</style>
