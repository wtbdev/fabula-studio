<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  ArrowLeft,
  BookOpenText,
  Clock,
  Download,
  FileCode2,
  FileSpreadsheet,
  FileText,
  FileType,
  Lightbulb,
  MapPin,
  RefreshCw,
  Save,
  Settings,
  UserRound,
  WandSparkles,
} from 'lucide-vue-next'
import { type ExportFormat, exportFormats, downloadExport } from '../utils/export'
import { generationApi } from '../api/generation'
import { projectsApi } from '../api/projects'
import { scenesApi } from '../api/scenes'
import { useAuth } from '../composables/useAuth'
import type { GenerateStatusDTO, ProjectDTO, SceneDTO } from '../api/types'
import SceneList from '../components/SceneList.vue'
import ScriptEditor from '../components/ScriptEditor.vue'
import AiSuggestionPanel from '../components/editor/AiSuggestionPanel.vue'
import SceneRegenerateModal from '../components/editor/SceneRegenerateModal.vue'

type SaveStatus = 'saved' | 'dirty' | 'saving' | 'failed'
type WorkbenchMode = 'script' | 'settings'
type ExtensionTab = 'info' | 'source' | 'suggestions'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { authState } = useAuth()

const projectId = computed(() => route.params.id as string)

const project = ref<ProjectDTO | null>(null)
const scenes = ref<SceneDTO[]>([])
const generateStatus = ref<GenerateStatusDTO | null>(null)
const activeSceneId = ref<string | null>(null)
const editorContent = ref('')
const saveStatus = ref<SaveStatus>('saved')
const loading = ref(true)
const sceneLoading = ref(false)
const metaSaving = ref(false)
const projectSaving = ref(false)
const generating = ref(false)
const activeMode = ref<WorkbenchMode>('script')
const extensionTab = ref<ExtensionTab>('info')
const exportVisible = ref(false)
const exportFormat = ref<ExportFormat>('txt')
const exportLoading = ref(false)
const sceneRegenerateVisible = ref(false)
const applyingRegeneratedScene = ref(false)

const workbenchModes: Array<{ key: WorkbenchMode; label: string }> = [
  { key: 'script', label: '剧本编辑器' },
  { key: 'settings', label: '项目设置' },
]

const sceneMetaForm = reactive({
  title: '',
  location: '',
  timeText: '',
  summary: '',
})

const projectSettingsForm = reactive({
  title: '',
  novelTitle: '',
})

const activeScene = computed(() =>
  scenes.value.find((scene) => scene.id === activeSceneId.value) ?? null,
)

const saveStatusMeta = computed(() => {
  const map: Record<SaveStatus, { text: string; type: 'error' | 'info' | 'success' | 'warning' }> =
    {
      saved: { text: '已保存', type: 'success' },
      dirty: { text: '未保存', type: 'warning' },
      saving: { text: '保存中', type: 'info' },
      failed: { text: '保存失败', type: 'error' },
    }
  return map[saveStatus.value]
})

const sceneCharacters = computed(() => {
  const characters = activeScene.value?.rawJson?.characters ?? []
  const scriptCharacters =
    activeScene.value?.rawJson?.script
      ?.map((block) => block.character)
      .filter((character): character is string => Boolean(character)) ?? []

  return Array.from(new Set([...characters, ...scriptCharacters]))
})

const sourceChapters = computed(() => activeScene.value?.rawJson?.source?.chapters ?? [])
const sourceSummary = computed(() => activeScene.value?.rawJson?.source?.summary ?? activeScene.value?.summary)

const projectStatusMeta = computed(() => {
  const map: Record<
    ProjectDTO['status'],
    { text: string; type: 'default' | 'error' | 'info' | 'success' | 'warning' }
  > = {
    draft: { text: '草稿', type: 'default' },
    generating: { text: '生成中', type: 'info' },
    completed: { text: '已完成', type: 'success' },
    failed: { text: '生成失败', type: 'error' },
  }

  return project.value ? map[project.value.status] : { text: '未加载', type: 'default' }
})

const isWorkbenchLocked = computed(
  () => generating.value || applyingRegeneratedScene.value || project.value?.status === 'generating',
)

const hasSceneMetaChanged = computed(() => {
  if (!activeScene.value) return false

  return (
    sceneMetaForm.title.trim() !== activeScene.value.title ||
    sceneMetaForm.location.trim() !== (activeScene.value.location ?? '') ||
    sceneMetaForm.timeText.trim() !== (activeScene.value.timeText ?? '') ||
    sceneMetaForm.summary.trim() !== (activeScene.value.summary ?? '')
  )
})

const hasProjectSettingsChanged = computed(() => {
  if (!project.value) return false

  return (
    projectSettingsForm.title.trim() !== project.value.title ||
    projectSettingsForm.novelTitle.trim() !== (project.value.novelTitle ?? '')
  )
})

const projectConfigItems = computed(() => {
  const config = project.value?.config
  if (!config) return []

  return [
    ['改编风格', config.style],
    ['对白详细度', config.dialogueLevel],
    ['改编方式', config.adaptationMode],
    ['场景粒度', config.sceneGranularity],
    ['旁白保留', config.narrationLevel],
  ].filter((item): item is [string, string] => Boolean(item[1]))
})

const formatDate = (value?: string) => {
  if (!value) return '未记录'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

const loadProject = async () => {
  try {
    project.value = await projectsApi.detail(projectId.value)
    generateStatus.value = await generationApi.status(projectId.value)
  } catch {
    message.error('项目加载失败')
  }
}

const replaceScene = (nextScene: SceneDTO) => {
  const sceneIndex = scenes.value.findIndex((scene) => scene.id === nextScene.id)
  if (sceneIndex >= 0) {
    scenes.value.splice(sceneIndex, 1, nextScene)
  } else {
    scenes.value.push(nextScene)
  }
}

const selectScene = async (sceneId: string) => {
  if (sceneId === activeSceneId.value) return
  activeSceneId.value = sceneId
  const scene = scenes.value.find((item) => item.id === sceneId)
  editorContent.value = scene?.content ?? ''
  saveStatus.value = 'saved'

  sceneLoading.value = true
  try {
    const detail = await scenesApi.detail(sceneId)
    replaceScene(detail)
    if (activeSceneId.value === sceneId && saveStatus.value === 'saved') {
      editorContent.value = detail.content
    }
  } catch {
    message.error('场次详情加载失败')
  } finally {
    sceneLoading.value = false
  }
}

const loadScenes = async () => {
  try {
    scenes.value = await scenesApi.list(projectId.value)
    if (scenes.value.length > 0 && !activeSceneId.value) {
      await selectScene(scenes.value[0].id)
    }
  } catch {
    message.error('场次列表加载失败')
  }
}

const saveCurrentScene = async () => {
  if (!activeScene.value) return true
  if (saveStatus.value === 'saving') return false
  if (saveStatus.value === 'saved') return true
  if (editorContent.value === activeScene.value.content && saveStatus.value !== 'failed') {
    saveStatus.value = 'saved'
    return true
  }

  saveStatus.value = 'saving'

  try {
    const result = await scenesApi.update(activeScene.value.id, {
      content: editorContent.value,
    })
    const scene = scenes.value.find((item) => item.id === activeScene.value!.id)
    if (scene) {
      scene.content = editorContent.value
      scene.updatedAt = result.updatedAt
    }
    saveStatus.value = 'saved'
    message.success('场次保存成功')
    return true
  } catch {
    saveStatus.value = 'failed'
    message.error('场次保存失败，请重试')
    return false
  }
}

const handleSceneSelect = async (sceneId: string) => {
  if (isWorkbenchLocked.value) return
  if (sceneId === activeSceneId.value) return

  if (saveStatus.value === 'dirty' || saveStatus.value === 'failed') {
    const saved = await saveCurrentScene()
    if (!saved) return
  }

  await selectScene(sceneId)

  if (window.matchMedia('(max-width: 960px)').matches) {
    requestAnimationFrame(() => {
      document
        .querySelector('.workbench-editor-column')
        ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }
}

const handleContentChange = (value: string) => {
  if (isWorkbenchLocked.value) return

  editorContent.value = value
  if (activeScene.value && saveStatus.value !== 'saving') {
    saveStatus.value = value === activeScene.value.content ? 'saved' : 'dirty'
  }
}

const handleSave = () => {
  if (isWorkbenchLocked.value) return
  void saveCurrentScene()
}

const handleOpenSceneRegenerate = async () => {
  if (isWorkbenchLocked.value) return

  if (!activeScene.value) {
    message.warning('请先选择一个场次')
    return
  }

  if (!editorContent.value.trim()) {
    message.warning('当前场次内容为空，无法重新生成')
    return
  }

  if ((authState.user?.aiPoints ?? 0) < 80) {
    message.warning('AI 点数不足，无法重新生成')
    return
  }

  if (saveStatus.value === 'dirty' || saveStatus.value === 'failed') {
    const saved = await saveCurrentScene()
    if (!saved) return
  }

  sceneRegenerateVisible.value = true
}

const handleSceneRegenerateApply = async (content: string) => {
  if (!activeScene.value || applyingRegeneratedScene.value) return

  const sceneId = activeScene.value.id
  applyingRegeneratedScene.value = true
  editorContent.value = content
  saveStatus.value = 'saving'

  try {
    const result = await scenesApi.update(sceneId, {
      content,
      versionSource: 'ai_regenerate',
    })
    const scene = scenes.value.find((item) => item.id === sceneId)
    if (scene) {
      scene.content = content
      scene.updatedAt = result.updatedAt
    }
    saveStatus.value = 'saved'
    sceneRegenerateVisible.value = false
    message.success('已应用并保存新版本')
  } catch {
    saveStatus.value = 'failed'
    message.error('新版本已生成，但保存失败，请手动保存')
  } finally {
    applyingRegeneratedScene.value = false
  }
}

const handleModeChange = async (mode: WorkbenchMode) => {
  if (isWorkbenchLocked.value) return
  if (mode === activeMode.value) return

  if (activeMode.value === 'script' && (saveStatus.value === 'dirty' || saveStatus.value === 'failed')) {
    const saved = await saveCurrentScene()
    if (!saved) return
  }

  activeMode.value = mode
}

const handleSceneMetaSave = async () => {
  if (!activeScene.value || metaSaving.value || isWorkbenchLocked.value) return

  const title = sceneMetaForm.title.trim()
  if (!title) {
    message.warning('场次标题不能为空')
    return
  }

  metaSaving.value = true
  saveStatus.value = 'saving'

  try {
    const result = await scenesApi.update(activeScene.value.id, {
      title,
      location: sceneMetaForm.location.trim(),
      timeText: sceneMetaForm.timeText.trim(),
      summary: sceneMetaForm.summary.trim(),
      content: editorContent.value,
    })

    const scene = scenes.value.find((item) => item.id === activeScene.value!.id)
    if (scene) {
      scene.title = title
      scene.location = sceneMetaForm.location.trim()
      scene.timeText = sceneMetaForm.timeText.trim()
      scene.summary = sceneMetaForm.summary.trim()
      scene.content = editorContent.value
      scene.updatedAt = result.updatedAt
    }

    saveStatus.value = 'saved'
    message.success('场次信息已保存')
  } catch {
    saveStatus.value = 'failed'
    message.error('场次信息保存失败，请重试')
  } finally {
    metaSaving.value = false
  }
}

const handleProjectSettingsSave = async () => {
  if (!project.value || projectSaving.value || isWorkbenchLocked.value) return

  const title = projectSettingsForm.title.trim()
  if (!title) {
    message.warning('项目名称不能为空')
    return
  }

  projectSaving.value = true

  try {
    const updatedProject = await projectsApi.update(project.value.id, {
      title,
      novelTitle: projectSettingsForm.novelTitle.trim(),
    })

    project.value = {
      ...project.value,
      ...updatedProject,
      title,
      novelTitle: projectSettingsForm.novelTitle.trim(),
    }
    message.success('项目设置已保存')
  } catch {
    message.error('项目设置保存失败，请重试')
  } finally {
    projectSaving.value = false
  }
}

const handleGenerateProject = async () => {
  if (!project.value || isWorkbenchLocked.value) return

  if (saveStatus.value === 'dirty' || saveStatus.value === 'failed') {
    const saved = await saveCurrentScene()
    if (!saved) return
  }

  const previousStatus = project.value.status
  const previousGenerateStatus = generateStatus.value
  const previousSceneId = activeSceneId.value
  generating.value = true
  project.value = {
    ...project.value,
    status: 'generating',
    updatedAt: new Date().toISOString(),
  }
  generateStatus.value = {
    projectId: project.value.id,
    status: 'generating',
    progress: 64,
    currentStep: '正在基于当前剧本增量生成，并同步人物关系',
  }

  try {
    const result = await generationApi.generate(project.value.id)
    const generatedScenes = [...result.scenes].sort(
      (previous, next) => previous.sceneNo - next.sceneNo,
    )
    scenes.value = generatedScenes
    project.value = {
      ...project.value,
      status: result.status,
      sceneCount: generatedScenes.length,
      updatedAt: new Date().toISOString(),
    }
    generateStatus.value = await generationApi.status(project.value.id)
    if (authState.user) {
      authState.user.aiPoints = result.remainingPoints
    }
    const nextSceneId =
      (previousSceneId && generatedScenes.some((scene) => scene.id === previousSceneId)
        ? previousSceneId
        : generatedScenes[0]?.id) ?? null

    activeSceneId.value = null
    editorContent.value = ''
    if (nextSceneId) {
      await selectScene(nextSceneId)
    }
    saveStatus.value = 'saved'
    message.success(`增量生成完成，消耗 ${result.costPoints} AI 点数`)
  } catch (error) {
    if (project.value) {
      project.value = {
        ...project.value,
        status: previousStatus,
        updatedAt: new Date().toISOString(),
      }
    }
    generateStatus.value = previousGenerateStatus
    message.error(error instanceof Error ? error.message : '剧本生成失败，请稍后重试')
  } finally {
    generating.value = false
  }
}

const handleExit = async () => {
  if (isWorkbenchLocked.value) return

  if (saveStatus.value === 'dirty' || saveStatus.value === 'failed') {
    const saved = await saveCurrentScene()
    if (!saved) return
  }

  await router.push('/projects')
}

const handleExport = () => {
  if (isWorkbenchLocked.value) return

  if (!project.value || scenes.value.length === 0) {
    message.warning('暂无可导出的剧本内容')
    return
  }

  exportFormat.value = 'txt'
  exportVisible.value = true
}

const handleExportConfirm = async () => {
  if (!project.value) return

  exportLoading.value = true

  try {
    if (saveStatus.value === 'dirty' || saveStatus.value === 'failed') {
      const saved = await saveCurrentScene()
      if (!saved) {
        exportLoading.value = false
        return
      }
    }

    await downloadExport(exportFormat.value, project.value, scenes.value, activeSceneId.value, editorContent.value)
    const label = exportFormats.find((f) => f.key === exportFormat.value)?.label ?? exportFormat.value
    message.success(`已导出 ${label} 格式剧本`)
    exportVisible.value = false
  } catch {
    message.error('导出失败，请重试')
  } finally {
    exportLoading.value = false
  }
}

const handleAccount = async () => {
  if (isWorkbenchLocked.value) return

  if (saveStatus.value === 'dirty' || saveStatus.value === 'failed') {
    const saved = await saveCurrentScene()
    if (!saved) return
  }

  await router.push('/account')
}

const handleSuggestionAiPointsChange = (remainingPoints: number) => {
  if (authState.user) {
    authState.user.aiPoints = remainingPoints
  }
}

onMounted(async () => {
  loading.value = true
  await Promise.all([loadProject(), loadScenes()])
  loading.value = false
})

watch(
  () => activeScene.value?.content,
  (newContent) => {
    if (newContent !== undefined && saveStatus.value === 'saved') {
      editorContent.value = newContent
    }
  },
)

watch(
  activeScene,
  (scene) => {
    sceneMetaForm.title = scene?.title ?? ''
    sceneMetaForm.location = scene?.location ?? ''
    sceneMetaForm.timeText = scene?.timeText ?? ''
    sceneMetaForm.summary = scene?.summary ?? ''
  },
  { immediate: true },
)

watch(
  project,
  (nextProject) => {
    projectSettingsForm.title = nextProject?.title ?? ''
    projectSettingsForm.novelTitle = nextProject?.novelTitle ?? ''
  },
  { immediate: true },
)
</script>

<template>
  <n-spin :show="loading" class="workbench-spin">
    <main class="workbench-page">
      <header class="workbench-topbar">
        <div class="workbench-topbar-left">
          <n-button secondary size="small" :disabled="isWorkbenchLocked" @click="handleExit">
            <template #icon>
              <n-icon><ArrowLeft /></n-icon>
            </template>
            退出编辑
          </n-button>

          <nav class="workbench-tabs" aria-label="编辑模块">
            <button
              v-for="mode in workbenchModes"
              :key="mode.key"
              type="button"
              class="workbench-tab"
              :class="{ active: activeMode === mode.key, disabled: isWorkbenchLocked }"
              :disabled="isWorkbenchLocked"
              @click="handleModeChange(mode.key)"
            >
              {{ mode.label }}
            </button>
          </nav>
        </div>

        <div class="workbench-topbar-right">
          <n-tag v-if="activeMode === 'script'" :bordered="false" :type="saveStatusMeta.type">
            {{ saveStatusMeta.text }}
          </n-tag>
          <n-tag v-else :bordered="false" :type="projectStatusMeta.type">
            {{ projectStatusMeta.text }}
          </n-tag>
          <n-button
            v-if="activeMode === 'script'"
            size="small"
            type="primary"
            :loading="saveStatus === 'saving'"
            :disabled="isWorkbenchLocked || (saveStatus !== 'dirty' && saveStatus !== 'failed')"
            @click="handleSave"
          >
            <template #icon>
              <n-icon><Save /></n-icon>
            </template>
            保存
          </n-button>
          <n-button
            v-if="activeMode === 'script'"
            size="small"
            secondary
            type="primary"
            :loading="applyingRegeneratedScene"
            :disabled="isWorkbenchLocked || !activeScene"
            @click="handleOpenSceneRegenerate"
          >
            <template #icon>
              <n-icon><RefreshCw /></n-icon>
            </template>
            重新生成本场
          </n-button>
          <n-popconfirm
            v-if="activeMode === 'script'"
            positive-text="开始生成"
            negative-text="取消"
            :show-icon="false"
            @positive-click="handleGenerateProject"
          >
            <template #trigger>
              <n-button
                size="small"
                secondary
                type="primary"
                :loading="generating"
                :disabled="isWorkbenchLocked || !project?.sourceText"
              >
                <template #icon>
                  <n-icon><WandSparkles /></n-icon>
                </template>
                增量生成
              </n-button>
            </template>
            将基于当前剧本继续生成场次并同步人物关系，期间编辑器会暂时锁定，消耗 300 AI 点数。
          </n-popconfirm>
          <n-button size="small" type="primary" :disabled="isWorkbenchLocked" @click="handleExport">
            <template #icon>
              <n-icon><Download /></n-icon>
            </template>
            导出
          </n-button>
          <n-tag :bordered="false" type="info">AI 点数：{{ authState.user?.aiPoints ?? 0 }}</n-tag>
          <n-button
            circle
            secondary
            size="small"
            aria-label="用户入口"
            :disabled="isWorkbenchLocked"
            @click="handleAccount"
          >
            <template #icon>
              <n-icon><UserRound /></n-icon>
            </template>
          </n-button>
        </div>
      </header>

      <div class="workbench-body" :class="{ 'settings-mode': activeMode === 'settings' }">
        <aside v-if="activeMode === 'script'" class="workbench-scene-column" aria-label="剧本场次区域">
          <SceneList
            :scenes="scenes"
            :active-scene-id="activeSceneId"
            :disabled="isWorkbenchLocked"
            @select="handleSceneSelect"
          />
        </aside>

        <section class="workbench-editor-column" aria-label="编辑器区域">
          <ScriptEditor
            v-if="activeMode === 'script'"
            :scene="activeScene"
            :project="project"
            :model-value="editorContent"
            :save-status="saveStatus"
            :loading="sceneLoading"
            :readonly="isWorkbenchLocked"
            :character-suggestions="sceneCharacters"
            @update:model-value="handleContentChange"
          />

          <section v-else class="project-settings-panel">
            <header class="settings-header">
              <span class="placeholder-icon">
                <n-icon><Settings /></n-icon>
              </span>
              <div>
                <h2>项目设置</h2>
                <p>当前只开放已有 API 支持的项目基础信息与增量生成。</p>
              </div>
            </header>

            <n-form label-placement="top" class="settings-form">
              <n-form-item label="项目名称">
                <n-input
                  v-model:value="projectSettingsForm.title"
                  placeholder="输入项目名称"
                  :disabled="isWorkbenchLocked"
                />
              </n-form-item>
              <n-form-item label="原小说名称">
                <n-input
                  v-model:value="projectSettingsForm.novelTitle"
                  placeholder="输入原小说名称"
                  :disabled="isWorkbenchLocked"
                />
              </n-form-item>
              <n-button
                type="primary"
                :loading="projectSaving"
                :disabled="isWorkbenchLocked || !hasProjectSettingsChanged"
                @click="handleProjectSettingsSave"
              >
                <template #icon>
                  <n-icon><Save /></n-icon>
                </template>
                保存项目设置
              </n-button>
            </n-form>

            <section class="settings-section">
              <div class="settings-section-title">
                <h3>生成状态</h3>
                <n-tag :bordered="false" :type="projectStatusMeta.type">
                  {{ projectStatusMeta.text }}
                </n-tag>
              </div>
              <n-progress
                v-if="generateStatus"
                type="line"
                :percentage="generateStatus.progress"
                :indicator-placement="'inside'"
              />
              <p>{{ generateStatus?.currentStep ?? '暂无生成状态' }}</p>
            </section>

            <section v-if="projectConfigItems.length" class="settings-section">
              <div class="settings-section-title">
                <h3>改编参数</h3>
              </div>
              <dl class="config-grid">
                <div v-for="[label, value] in projectConfigItems" :key="label">
                  <dt>{{ label }}</dt>
                  <dd>{{ value }}</dd>
                </div>
              </dl>
            </section>

            <section class="settings-section danger">
              <div class="settings-section-title">
                <h3>增量生成剧本</h3>
                <n-tag :bordered="false" type="warning">消耗 300 点</n-tag>
              </div>
              <p>该操作会调用现有生成接口，基于当前剧本继续补充场次，并同步人物关系等生成结果。</p>
              <n-popconfirm
                positive-text="开始生成"
                negative-text="取消"
                @positive-click="handleGenerateProject"
              >
                <template #trigger>
                  <n-button
                    type="warning"
                    :loading="generating"
                    :disabled="isWorkbenchLocked || !project?.sourceText"
                  >
                    <template #icon>
                      <n-icon><WandSparkles /></n-icon>
                    </template>
                    增量生成
                  </n-button>
                </template>
                将基于当前剧本继续生成，并消耗 300 AI 点数。生成完成前编辑器会暂时锁定。
              </n-popconfirm>
            </section>
          </section>
        </section>

        <aside v-if="activeMode === 'script'" class="workbench-extension-column" aria-label="拓展功能区">
          <div class="extension-tabs">
            <button
              type="button"
              :class="{ active: extensionTab === 'info' }"
              @click="extensionTab = 'info'"
            >
              <n-icon><BookOpenText /></n-icon>
              场次信息
            </button>
            <button
              type="button"
              :class="{ active: extensionTab === 'source' }"
              @click="extensionTab = 'source'"
            >
              <n-icon><FileText /></n-icon>
              原文依据
            </button>
            <button
              type="button"
              :class="{ active: extensionTab === 'suggestions' }"
              @click="extensionTab = 'suggestions'"
            >
              <n-icon><Lightbulb /></n-icon>
              AI 建议
            </button>
          </div>

          <div v-if="activeScene" class="extension-content">
            <section v-if="extensionTab === 'info'" class="extension-card">
              <span class="extension-icon"><n-icon><BookOpenText /></n-icon></span>
              <h2>{{ activeScene.title }}</h2>
              <p>{{ activeScene.summary || '当前场次暂无概要。' }}</p>

              <dl class="scene-facts">
                <div>
                  <dt><n-icon><MapPin /></n-icon>地点</dt>
                  <dd>{{ activeScene.location || '未记录' }}</dd>
                </div>
                <div>
                  <dt><n-icon><Clock /></n-icon>时间</dt>
                  <dd>{{ activeScene.timeText || '未记录' }}</dd>
                </div>
                <div>
                  <dt>更新时间</dt>
                  <dd>{{ formatDate(activeScene.updatedAt) }}</dd>
                </div>
              </dl>

              <div class="character-strip">
                <span>出场人物</span>
                <n-space v-if="sceneCharacters.length" size="small">
                  <n-tag v-for="character in sceneCharacters" :key="character" size="small" :bordered="false">
                    {{ character }}
                  </n-tag>
                </n-space>
                <p v-else>暂无人物记录。</p>
              </div>

              <n-collapse class="scene-meta-collapse" arrow-placement="right">
                <n-collapse-item title="编辑场次信息" name="scene-meta">
                  <n-form label-placement="top" class="scene-meta-form">
                    <n-form-item label="场次标题">
                      <n-input
                        v-model:value="sceneMetaForm.title"
                        placeholder="输入场次标题"
                        :disabled="isWorkbenchLocked"
                      />
                    </n-form-item>
                    <n-form-item label="地点">
                      <n-input
                        v-model:value="sceneMetaForm.location"
                        placeholder="输入地点"
                        :disabled="isWorkbenchLocked"
                      />
                    </n-form-item>
                    <n-form-item label="时间">
                      <n-input
                        v-model:value="sceneMetaForm.timeText"
                        placeholder="输入时间"
                        :disabled="isWorkbenchLocked"
                      />
                    </n-form-item>
                    <n-form-item label="概要">
                      <n-input
                        v-model:value="sceneMetaForm.summary"
                        type="textarea"
                        :autosize="{ minRows: 3, maxRows: 5 }"
                        placeholder="输入当前场次概要"
                        :disabled="isWorkbenchLocked"
                      />
                    </n-form-item>
                    <n-button
                      block
                      type="primary"
                      :loading="metaSaving"
                      :disabled="isWorkbenchLocked || !hasSceneMetaChanged"
                      @click="handleSceneMetaSave"
                    >
                      <template #icon>
                        <n-icon><Save /></n-icon>
                      </template>
                      保存场次信息
                    </n-button>
                  </n-form>
                </n-collapse-item>
              </n-collapse>
            </section>

            <section v-else-if="extensionTab === 'source'" class="extension-card">
              <span class="extension-icon brick"><n-icon><FileText /></n-icon></span>
              <h2>原文依据</h2>
              <p>{{ sourceSummary || '暂无原文依据摘要。' }}</p>
              <div class="source-chapters">
                <span>章节索引</span>
                <n-space v-if="sourceChapters.length" size="small">
                  <n-tag v-for="chapter in sourceChapters" :key="chapter" size="small" type="info" :bordered="false">
                    {{ chapter }}
                  </n-tag>
                </n-space>
                <p v-else>尚未关联章节。</p>
              </div>
            </section>

            <AiSuggestionPanel
              v-else
              :project-id="project?.id ?? null"
              :scene-id="activeScene.id"
              :scene-title="activeScene.title"
              :scene-content="editorContent"
              :disabled="isWorkbenchLocked"
              @refresh-user="handleSuggestionAiPointsChange"
            />
          </div>

          <n-empty v-else description="暂无选中场次" class="extension-empty" />
        </aside>
      </div>
    </main>

    <n-modal
      v-model:show="exportVisible"
      preset="card"
      title="导出剧本"
      class="export-modal"
      :bordered="false"
    >
      <div class="export-modal-body">
        <p class="export-modal-hint">选择导出格式，将下载当前项目的全部场次内容。</p>

        <div class="export-format-list">
          <label
            v-for="format in exportFormats"
            :key="format.key"
            class="export-format-option"
            :class="{ active: exportFormat === format.key }"
          >
            <input
              v-model="exportFormat"
              type="radio"
              name="export-format"
              :value="format.key"
              class="export-format-radio"
            />
            <span class="export-format-icon">
              <n-icon>
                <FileText v-if="format.key === 'txt'" />
                <FileSpreadsheet v-else-if="format.key === 'word'" />
                <FileType v-else-if="format.key === 'markdown'" />
                <FileCode2 v-else />
              </n-icon>
            </span>
            <span class="export-format-copy">
              <strong>{{ format.label }}<small>.{{ format.extension }}</small></strong>
              <span>{{ format.description }}</span>
            </span>
          </label>
        </div>
      </div>

      <template #footer>
        <n-space justify="end">
          <n-button @click="exportVisible = false">取消</n-button>
          <n-button type="primary" :loading="exportLoading" @click="handleExportConfirm">
            <template #icon>
              <n-icon><Download /></n-icon>
            </template>
            导出
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <SceneRegenerateModal
      v-model:show="sceneRegenerateVisible"
      :scene-id="activeScene?.id ?? null"
      :scene-title="activeScene?.title ?? ''"
      :scene-content="editorContent"
      :ai-points="authState.user?.aiPoints ?? 0"
      :disabled="isWorkbenchLocked && !applyingRegeneratedScene"
      :applying="applyingRegeneratedScene"
      @apply="handleSceneRegenerateApply"
      @refresh-user="handleSuggestionAiPointsChange"
    />
  </n-spin>
</template>

<style scoped>
.workbench-spin {
  min-height: 100vh;
}

.workbench-spin :deep(.n-spin-content) {
  min-height: 100vh;
}

.workbench-page {
  display: grid;
  grid-template-rows: 58px minmax(0, 1fr);
  min-width: 320px;
  height: 100vh;
  box-sizing: border-box;
  padding: 18px 24px 22px;
  color: var(--color-ink);
  background:
    linear-gradient(rgba(23, 33, 31, 0.032) 1px, transparent 1px) 0 0 / 100% 32px,
    linear-gradient(90deg, rgba(23, 33, 31, 0.024) 1px, transparent 1px) 0 0 / 32px 100%,
    var(--color-canvas);
}

.workbench-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  min-height: 52px;
  padding: 0 16px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.94);
  backdrop-filter: blur(10px);
  box-shadow: var(--shadow-panel);
}

.workbench-topbar-left,
.workbench-topbar-right,
.workbench-tabs {
  display: flex;
  align-items: center;
}

.workbench-topbar-left {
  flex: 1 1 auto;
  min-width: 0;
  gap: 18px;
}

.workbench-topbar-right {
  flex: 0 1 auto;
  min-width: 0;
  gap: 10px;
}

.workbench-tabs {
  gap: 4px;
  min-height: 52px;
}

.workbench-tab {
  display: inline-flex;
  position: relative;
  min-height: 52px;
  align-items: center;
  gap: 4px;
  padding: 0 16px;
  border: 0;
  color: var(--color-muted);
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
}

.workbench-tab::after {
  position: absolute;
  right: 14px;
  bottom: 0;
  left: 14px;
  height: 2px;
  border-radius: 999px;
  background: transparent;
  content: "";
}

.workbench-tab.active {
  color: var(--color-sage);
}

.workbench-tab.disabled {
  cursor: not-allowed;
  opacity: 0.56;
}

.workbench-tab.active::after {
  background: var(--color-sage);
}

.workbench-tab .n-icon {
  font-size: 14px;
}

.workbench-body {
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(560px, 1fr) minmax(260px, 320px);
  gap: 12px;
  overflow: hidden;
  min-height: 0;
  padding-top: 16px;
}

.workbench-body.settings-mode {
  grid-template-columns: minmax(0, 960px);
  justify-content: center;
  overflow: auto;
}

.workbench-scene-column,
.workbench-editor-column,
.workbench-extension-column {
  min-width: 0;
  min-height: 0;
}

.workbench-editor-column {
  display: flex;
}

.settings-mode .workbench-editor-column {
  min-height: 0;
}

.project-settings-panel {
  display: flex;
  flex-direction: column;
  gap: 18px;
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 22px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.96);
  box-shadow: var(--shadow-panel);
}

.settings-header,
.settings-section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.settings-header {
  justify-content: flex-start;
}

.placeholder-icon {
  display: grid;
  flex-shrink: 0;
  width: 54px;
  height: 54px;
  place-items: center;
  border-radius: 8px;
  color: var(--color-sage);
  background: var(--color-sage-soft);
  font-size: 28px;
}

.settings-header h2 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 24px;
}

.settings-header p,
.settings-section p {
  margin: 6px 0 0;
  color: var(--color-muted);
  line-height: 1.7;
}

.settings-form,
.settings-section {
  padding: 16px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: #fff;
}

.settings-section-title h3 {
  margin: 0;
  color: var(--color-ink);
  font-size: 15px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 14px 0 0;
}

.config-grid div {
  padding: 10px;
  border: 1px solid rgba(220, 227, 223, 0.9);
  border-radius: 8px;
  background: #fafbf9;
}

.config-grid dt {
  color: var(--color-muted);
  font-size: 12px;
}

.config-grid dd {
  margin: 5px 0 0;
  color: var(--color-ink);
  font-weight: 800;
}

.settings-section.danger {
  border-color: rgba(141, 73, 56, 0.26);
}

.workbench-extension-column {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.94);
  box-shadow: var(--shadow-panel);
}

.extension-tabs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-bottom: 1px solid var(--color-line);
}

.extension-tabs button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 42px;
  min-width: 0;
  border: 0;
  border-right: 1px solid var(--color-line);
  color: var(--color-muted);
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
}

.extension-tabs button:last-child {
  border-right: 0;
}

.extension-tabs button.active {
  color: var(--color-sage);
  background: var(--color-sage-soft);
}

.extension-tabs .n-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.extension-content {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.extension-card {
  padding: 18px;
}

.extension-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  color: var(--color-sage);
  background: var(--color-sage-soft);
  font-size: 22px;
}

.extension-icon.brick {
  color: var(--color-paper);
  background: var(--color-brick);
}

.extension-card h2 {
  margin: 16px 0 0;
  color: var(--color-ink);
  font-family: var(--font-display);
  font-size: 19px;
  line-height: 1.35;
}

.extension-card p {
  margin: 10px 0 0;
  color: var(--color-muted);
  font-size: 14px;
  line-height: 1.75;
}

.scene-facts {
  display: grid;
  gap: 10px;
  margin: 18px 0 0;
}

.scene-facts div,
.character-strip,
.source-chapters {
  padding: 12px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.72);
}

.scene-facts dt {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-muted);
  font-size: 12px;
}

.scene-facts dd {
  margin: 4px 0 0;
  color: var(--color-ink);
  font-weight: 700;
}

.character-strip,
.source-chapters,
.scene-meta-collapse {
  margin-top: 12px;
}

.scene-meta-collapse :deep(.n-collapse-item) {
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: #fff;
}

.scene-meta-collapse :deep(.n-collapse-item__header) {
  padding: 12px;
  font-weight: 800;
}

.scene-meta-collapse :deep(.n-collapse-item__content-inner) {
  padding: 0 12px 12px;
}

.character-strip > span,
.source-chapters > span {
  display: block;
  margin-bottom: 8px;
  color: var(--color-ink);
  font-size: 13px;
  font-weight: 800;
}

.extension-empty {
  margin: auto;
}

@media (max-width: 1180px) {
  .workbench-page {
    grid-template-rows: auto minmax(0, 1fr);
    height: auto;
    min-height: 100vh;
    padding: 16px;
  }

  .workbench-topbar {
    gap: 12px;
    padding-inline: 12px;
  }

  .workbench-body {
    grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
    overflow: visible;
  }

  .workbench-body.settings-mode {
    grid-template-columns: minmax(0, 920px);
  }

  .workbench-scene-column,
  .workbench-editor-column {
    min-height: 520px;
  }

  .settings-mode .workbench-editor-column {
    min-height: 0;
  }

  .workbench-extension-column {
    grid-column: 1 / -1;
    min-height: 360px;
  }
}

@media (max-width: 960px) {
  .workbench-page {
    padding: 14px;
  }

  .workbench-topbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
    padding-block: 10px;
  }

  .workbench-topbar-left,
  .workbench-topbar-right {
    width: 100%;
    justify-content: flex-start;
  }

  .workbench-topbar-left {
    flex-wrap: wrap;
  }

  .workbench-topbar-right {
    flex-wrap: wrap;
  }

  .workbench-tabs {
    margin-left: auto;
  }

  .workbench-tab {
    min-height: 40px;
  }

  .workbench-body {
    grid-template-columns: minmax(0, 1fr);
  }

  .workbench-body.settings-mode {
    grid-template-columns: minmax(0, 1fr);
  }

  .workbench-scene-column {
    height: min(46vh, 380px);
    min-height: 260px;
    max-height: 380px;
  }

  .workbench-editor-column {
    min-height: 540px;
  }

  .workbench-extension-column {
    min-height: 440px;
  }
}

@media (max-width: 820px) {
  .workbench-page {
    padding: 12px;
  }

  .workbench-topbar-left,
  .workbench-topbar-right {
    align-items: flex-start;
    flex-wrap: wrap;
    justify-content: flex-start;
  }

  .workbench-tabs {
    width: 100%;
    margin-left: 0;
    min-height: auto;
    overflow-x: auto;
  }

  .workbench-tab {
    min-height: 40px;
    flex: 0 0 auto;
  }

  .workbench-scene-column {
    height: min(42vh, 340px);
    min-height: 240px;
    max-height: 340px;
  }

  .workbench-editor-column {
    min-height: 520px;
  }

  .workbench-extension-column {
    min-height: 420px;
  }
}

.export-modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.export-modal-hint {
  margin: 0;
  color: var(--color-muted);
  font-size: 14px;
  line-height: 1.7;
}

.export-format-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.export-format-option {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.72);
  cursor: pointer;
  transition:
    border-color 180ms ease,
    background-color 180ms ease,
    box-shadow 180ms ease;
}

.export-format-option:hover {
  border-color: rgba(47, 118, 100, 0.26);
  background: rgba(238, 247, 243, 0.4);
}

.export-format-option.active {
  border-color: var(--color-sage);
  background: var(--color-sage-soft);
  box-shadow: 0 0 0 1px var(--color-sage);
}

.export-format-radio {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
}

.export-format-icon {
  display: grid;
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  place-items: center;
  border-radius: 8px;
  color: var(--color-sage);
  background: rgba(238, 247, 243, 0.86);
  font-size: 20px;
}

.export-format-option.active .export-format-icon {
  color: #fffdf8;
  background: var(--color-sage);
}

.export-format-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.export-format-copy strong {
  color: var(--color-ink);
  font-size: 15px;
  font-weight: 700;
  line-height: 22px;
}

.export-format-copy strong small {
  margin-left: 4px;
  color: var(--color-muted);
  font-weight: 500;
  font-size: 13px;
}

.export-format-copy > span {
  color: var(--color-muted);
  font-size: 13px;
  line-height: 1.65;
}

@media (max-width: 960px) {
  .export-format-list {
    gap: 8px;
  }
}
</style>

<style>
.export-modal {
  width: min(480px, calc(100vw - 32px)) !important;
}
</style>
