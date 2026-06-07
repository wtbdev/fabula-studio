<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
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
  GitBranch,
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
import type { GenerationArtifacts, GenerateStatusDTO, PipelineEventDTO, ProjectDTO, SceneDTO } from '../api/types'
import SceneList from '../components/SceneList.vue'
import ScriptEditor from '../components/ScriptEditor.vue'
import AiSuggestionPanel from '../components/editor/AiSuggestionPanel.vue'
import SceneRegenerateModal from '../components/editor/SceneRegenerateModal.vue'

type SaveStatus = 'saved' | 'dirty' | 'saving' | 'failed'
type WorkbenchMode = 'script' | 'settings'
type ExtensionTab = 'info' | 'source' | 'suggestions' | 'artifacts'

type EventDetails = Record<string, unknown>
type RealtimeTreeUnit = {
  id: string
  title: string
  summary: string
  meta: string
  characters: string[]
}
type RealtimePlan = {
  id: string
  title: string
  purpose: string
  location?: string
  sourceNodeIds: string[]
}
type RealtimeSceneHeading = {
  id: string
  heading: string
}

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { authState, fetchMe } = useAuth()

const projectId = computed(() => route.params.id as string)

const project = ref<ProjectDTO | null>(null)
const scenes = ref<SceneDTO[]>([])
const generateStatus = ref<GenerateStatusDTO | null>(null)
const generationArtifacts = ref<GenerationArtifacts | null>(null)
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
const realtimeEvents = ref<PipelineEventDTO[]>([])
const realtimeTreeUnits = ref<RealtimeTreeUnit[]>([])
const realtimePlans = ref<RealtimePlan[]>([])
const realtimeSceneHeadings = ref<RealtimeSceneHeading[]>([])
const realtimeTraceId = ref<string | null>(null)
const realtimeRunId = ref<string | null>(null)
const realtimeProgress = ref<number | null>(null)
const realtimeCurrentStep = ref<string | null>(null)
const pendingGenerationJobId = ref<string | null>(null)
let generationPollingTimer: number | null = null
let pollingGenerationStatus = false
let generationEventSource: EventSource | null = null


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

const stageLabelMap: Record<string, string> = {
  queued: '生成排队中',
  running: '生成运行中',
  source_indexing: '原文索引构建中',
  beat_extracting: '故事节拍提取中',
  beat_reconciling: '节拍边界校准中',
  scene_planning: '场景规划中',
  scene_writing: '场景写作中',
  graph_updating: '关系图谱更新中',
  editor_reviewing: '主编审校中',
  final_validating: '最终校验中',
  extract_story_beats: '故事节拍提取中',
  aggregate_units: '剧情单元聚合中',
  update_graph: '关系图谱更新中',
  graph_update: '关系图谱更新中',
  plan_scenes: '场景规划中',
  generate_scenes: '场景写作中',
  write_scene: '场景写作中',
  graph_hook: '图谱钩子处理中',
  editor_review: '主编审校中',
  validation: '最终校验中',
  commit_result: '生成结果提交中',
  completed: '生成完成',
}

const generationStepLabel = computed(() => {
  const currentStep =
    realtimeCurrentStep.value ?? generateStatus.value?.currentStep ?? generateStatus.value?.job?.currentStep
  if (!currentStep) return '暂无生成状态'

  return stageLabelMap[currentStep] ?? currentStep
})

const generationProgress = computed(
  () => realtimeProgress.value ?? generateStatus.value?.progress ?? generateStatus.value?.job?.progress ?? 0,
)

const applyGenerationArtifacts = (artifacts?: GenerationArtifacts) => {
  if (artifacts) generationArtifacts.value = artifacts
}

const currentGenerationJobId = computed(() => generateStatus.value?.jobId ?? generateStatus.value?.job?.id ?? null)

const eventMatchesCurrentGeneration = (event: PipelineEventDTO) => {
  const currentJobId = currentGenerationJobId.value
  const pendingJobId = pendingGenerationJobId.value

  if (event.projectId) return event.projectId === projectId.value
  if (event.jobId && currentJobId) return event.jobId === currentJobId
  if (event.jobId && pendingJobId) return event.jobId === pendingJobId

  return false
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null

const toStringArray = (value: unknown) =>
  Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []

const getString = (record: Record<string, unknown>, key: string) => {
  const value = record[key]
  return typeof value === 'string' ? value : ''
}

const resetRealtimeGeneration = () => {
  realtimeEvents.value = []
  realtimeTreeUnits.value = []
  realtimePlans.value = []
  realtimeSceneHeadings.value = []
  realtimeTraceId.value = null
  realtimeRunId.value = null
  realtimeProgress.value = null
  realtimeCurrentStep.value = null
  pendingGenerationJobId.value = null
}

const collectRealtimeTreeUnits = (details: EventDetails) => {
  const tree = details.tree
  if (!isRecord(tree) || !isRecord(tree.nodes)) return []
  const nodes = tree.nodes

  const leafIds = toStringArray(tree.leaf_node_ids)
  const rootNodeId = typeof tree.root_node_id === 'string' ? tree.root_node_id : ''
  const ids = leafIds.length ? leafIds : Object.keys(nodes).filter((id) => id !== rootNodeId)

  return ids.flatMap((id): RealtimeTreeUnit[] => {
    const node = nodes[id]
    if (!isRecord(node)) return []

    const start = getString(node, 'start_sentence_id') || '?'
    const end = getString(node, 'end_sentence_id') || '?'
    const location = getString(node, 'location') || '-'
    const timeFrame = getString(node, 'time_frame') || '-'
    const summary = getString(node, 'summary') || getString(node, 'main_conflict') || getString(node, 'text_content')

    return [
      {
        id,
        title: getString(node, 'unit_type') || getString(node, 'decision') || id,
        summary: summary || '暂无摘要。',
        meta: `${start} → ${end} · ${location} · ${timeFrame}`,
        characters: toStringArray(node.characters).slice(0, 5),
      },
    ]
  })
}

const collectRealtimePlans = (details: EventDetails) => {
  const plans = details.plans
  if (!Array.isArray(plans)) return []

  return plans.flatMap((plan, index): RealtimePlan[] => {
    if (!isRecord(plan)) return []
    const id = getString(plan, 'id') || `plan-${index + 1}`
    const sceneCount = typeof plan.scene_count === 'number' ? plan.scene_count : undefined

    return [
      {
        id,
        title: sceneCount ? `${id} · ${sceneCount} 场` : id,
        purpose: getString(plan, 'purpose') || '该场景计划未提供目的说明。',
        location: getString(plan, 'location') || undefined,
        sourceNodeIds: toStringArray(plan.source_node_ids),
      },
    ]
  })
}

const collectRealtimeSceneHeadings = (details: EventDetails) => {
  const scenes = details.scenes
  if (!Array.isArray(scenes)) return []

  return scenes.flatMap((scene, index): RealtimeSceneHeading[] => {
    if (typeof scene === 'string') return [{ id: `scene-${index + 1}`, heading: scene }]
    if (!isRecord(scene)) return []

    const heading = getString(scene, 'heading') || getString(scene, 'title') || `场景 ${index + 1}`
    return [{ id: getString(scene, 'id') || `scene-${index + 1}`, heading }]
  })
}

const applyPipelineEvent = (event: PipelineEventDTO) => {
  if (!eventMatchesCurrentGeneration(event)) return
  pendingGenerationJobId.value = event.jobId ?? pendingGenerationJobId.value

  realtimeEvents.value = [event, ...realtimeEvents.value].slice(0, 30)
  realtimeTraceId.value = event.traceId ?? realtimeTraceId.value
  realtimeRunId.value = event.runId ?? realtimeRunId.value
  realtimeProgress.value = typeof event.progress === 'number' ? event.progress : realtimeProgress.value
  realtimeCurrentStep.value = event.step ?? realtimeCurrentStep.value

  const details = event.details ?? {}
  if (event.type === 'tree_snapshot') {
    realtimeTreeUnits.value = collectRealtimeTreeUnits(details)
  }
  if (event.type === 'scene_planned') {
    realtimePlans.value = collectRealtimePlans(details)
  }
  if (event.type === 'scene_written' || event.step === 'generate_scenes') {
    const headings = collectRealtimeSceneHeadings(details)
    if (headings.length) realtimeSceneHeadings.value = headings
  }
}

const getEventStreamURL = () => {
  const baseURL = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')
  return `${baseURL}/events/stream`
}

const stopGenerationEvents = () => {
  if (!generationEventSource) return
  generationEventSource.close()
  generationEventSource = null
}

const startGenerationEvents = () => {
  if (generationEventSource) return

  generationEventSource = new EventSource(getEventStreamURL())
  generationEventSource.onmessage = (messageEvent) => {
    try {
      applyPipelineEvent(JSON.parse(messageEvent.data) as PipelineEventDTO)
    } catch {
      // Ignore malformed SSE payloads; polling remains the source of truth for completion.
    }
  }
}

const isGenerationActive = (status?: string) =>
  status === 'generating' || status === 'queued' || status === 'running'

const stopGenerationPolling = () => {
  if (generationPollingTimer) {
    window.clearInterval(generationPollingTimer)
    generationPollingTimer = null
  }
  stopGenerationEvents()
}

const applyGenerateStatus = (status: GenerateStatusDTO) => {
  pendingGenerationJobId.value = status.jobId ?? status.job?.id ?? pendingGenerationJobId.value
  generateStatus.value = status
  applyGenerationArtifacts(status.artifacts ?? status.job?.artifacts)

  if (!project.value) return

  if (isGenerationActive(status.status)) {
    project.value = {
      ...project.value,
      status: 'generating',
      errorMessage: null,
      updatedAt: status.job?.updatedAt ?? project.value.updatedAt,
    }
    return
  }

  if (status.status === 'failed') {
    project.value = {
      ...project.value,
      status: 'failed',
      errorMessage: status.errorMessage ?? status.job?.errorMessage ?? project.value.errorMessage,
      updatedAt: status.job?.updatedAt ?? project.value.updatedAt,
    }
  }
}

const refreshCompletedGeneration = async () => {
  const previousSceneId = activeSceneId.value
  const [nextProject, nextScenes] = await Promise.all([
    projectsApi.detail(projectId.value),
    scenesApi.list(projectId.value),
  ])
  project.value = nextProject
  applyGenerationArtifacts(nextProject.artifacts)
  scenes.value = [...nextScenes].sort((previous, next) => previous.sceneNo - next.sceneNo)

  const nextSceneId =
    (previousSceneId && scenes.value.some((scene) => scene.id === previousSceneId)
      ? previousSceneId
      : scenes.value[0]?.id) ?? null

  activeSceneId.value = null
  editorContent.value = ''
  if (nextSceneId) {
    await selectScene(nextSceneId)
  }
  saveStatus.value = 'saved'
}

const pollGenerationStatus = async () => {
  if (pollingGenerationStatus) return
  pollingGenerationStatus = true

  try {
    const status = await generationApi.status(projectId.value)
    applyGenerateStatus(status)

    if (status.status === 'completed') {
      stopGenerationPolling()
      realtimeProgress.value = null
      realtimeCurrentStep.value = null
      await refreshCompletedGeneration()
      void fetchMe()
      generating.value = false
      message.success('剧本生成完成')
      return
    }

    if (status.status === 'failed') {
      stopGenerationPolling()
      realtimeProgress.value = null
      realtimeCurrentStep.value = null
      generating.value = false
      message.error(status.errorMessage ?? status.job?.errorMessage ?? '剧本生成失败，请稍后重试')
      return
    }

    generating.value = isGenerationActive(status.status)
  } catch {
    // Keep polling: transient network errors should not unlock the editor while backend is generating.
  } finally {
    pollingGenerationStatus = false
  }
}

const startGenerationPolling = () => {
  generating.value = true
  startGenerationEvents()
  void pollGenerationStatus()
  if (!generationPollingTimer) {
    generationPollingTimer = window.setInterval(() => {
      void pollGenerationStatus()
    }, 2500)
  }
}

const sourceIndexChapterCount = computed(
  () => generationArtifacts.value?.sourceIndex?.chapters?.length ?? 0,
)

const storyBeatCount = computed(() => generationArtifacts.value?.storyBeats?.length ?? 0)
const scenePlanCount = computed(() => generationArtifacts.value?.scenePlan?.length ?? 0)

const realtimeTreeUnitCount = computed(() => realtimeTreeUnits.value.length)
const realtimePlanCount = computed(() => realtimePlans.value.length)
const realtimeSceneHeadingCount = computed(() => realtimeSceneHeadings.value.length)

const hasRealtimeGenerationArtifacts = computed(
  () =>
    realtimeEvents.value.length > 0 ||
    realtimeTreeUnitCount.value > 0 ||
    realtimePlanCount.value > 0 ||
    realtimeSceneHeadingCount.value > 0 ||
    Boolean(realtimeTraceId.value || realtimeRunId.value),
)

const hasGenerationArtifacts = computed(
  () =>
    hasRealtimeGenerationArtifacts.value ||
    Boolean(generationArtifacts.value?.sourceIndex) ||
    storyBeatCount.value > 0 ||
    scenePlanCount.value > 0 ||
    Boolean(generationArtifacts.value?.graphSnapshot) ||
    Boolean(generationArtifacts.value?.warnings?.length),
)

const graphSnapshotItems = computed(() => {
  const snapshot = generationArtifacts.value?.graphSnapshot
  if (!snapshot) return []

  return [
    ['节点', snapshot.nodeCount],
    ['边', snapshot.edgeCount],
    ['角色', snapshot.characterCount],
    ['关系', snapshot.relationshipCount],
    ['更新时间', snapshot.updatedAt ? formatDate(snapshot.updatedAt) : undefined],
  ].filter((item): item is [string, string | number] => item[1] !== undefined && item[1] !== null)
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

const formatEventTimestamp = (value?: string) => {
  if (!value) return '刚刚'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date)
}

const getEventLabel = (event: PipelineEventDTO) => {
  const step = event.step ? stageLabelMap[event.step] ?? event.step : event.type
  return event.message || event.error || step
}

const loadProject = async () => {
  try {
    project.value = await projectsApi.detail(projectId.value)
    applyGenerationArtifacts(project.value.artifacts)
    const status = await generationApi.status(projectId.value)
    applyGenerateStatus(status)
    if (isGenerationActive(status.status) || project.value.status === 'generating') {
      startGenerationPolling()
    }
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
  generating.value = true
  project.value = {
    ...project.value,
    status: 'generating',
    errorMessage: null,
    updatedAt: new Date().toISOString(),
  }
  resetRealtimeGeneration()
  startGenerationEvents()
  generateStatus.value = {
    projectId: project.value.id,
    status: 'queued',
    progress: 0,
    currentStep: 'queued',
  }

  try {
    const result = await generationApi.generate(project.value.id, {
      config: project.value.config,
      adaptationProfile: project.value.adaptationProfile,
    })
    pendingGenerationJobId.value = result.jobId ?? result.job?.id ?? pendingGenerationJobId.value
    generateStatus.value = {
      projectId: project.value.id,
      jobId: result.jobId,
      job: result.job,
      status: result.status,
      progress: result.job?.progress ?? (result.status === 'completed' ? 100 : 0),
      currentStep: result.job?.currentStep ?? 'source_indexing',
      artifacts: result.artifacts,
    }
    applyGenerationArtifacts(result.artifacts ?? result.job?.artifacts)
    message.success('生成任务已启动')
    startGenerationPolling()
  } catch (error) {
    if (project.value) {
      project.value = {
        ...project.value,
        status: previousStatus,
        updatedAt: new Date().toISOString(),
      }
    }
    generateStatus.value = previousGenerateStatus
    generating.value = false
    stopGenerationEvents()
    message.error(error instanceof Error ? error.message : '剧本生成失败，请稍后重试')
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

onUnmounted(() => {
  stopGenerationPolling()
  stopGenerationEvents()
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
                :percentage="generationProgress"
                :indicator-placement="'inside'"
              />
              <p>{{ generationStepLabel }}</p>
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
            <button
              type="button"
              :class="{ active: extensionTab === 'artifacts' }"
              @click="extensionTab = 'artifacts'"
            >
              <n-icon><GitBranch /></n-icon>
              生成产物
            </button>
          </div>

          <div v-if="activeScene || extensionTab === 'artifacts'" class="extension-content">
            <section v-if="extensionTab === 'info' && activeScene" class="extension-card">
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

            <section v-else-if="extensionTab === 'source' && activeScene" class="extension-card">
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

            <section v-else-if="extensionTab === 'artifacts'" class="extension-card artifact-card">
              <span class="extension-icon"><n-icon><GitBranch /></n-icon></span>
              <h2>生成产物</h2>

              <div v-if="hasGenerationArtifacts" class="artifact-stack">
                <section v-if="hasRealtimeGenerationArtifacts" class="artifact-section realtime">
                  <div class="artifact-section-heading">
                    <h3>实时生成</h3>
                    <n-tag size="small" :bordered="false" type="info">
                      {{ realtimeEvents.length }} 事件
                    </n-tag>
                  </div>
                  <p>{{ generationStepLabel }}</p>
                  <dl v-if="realtimeTraceId || realtimeRunId" class="config-grid artifact-graph-grid artifact-trace-grid">
                    <div v-if="realtimeTraceId">
                      <dt>Trace ID</dt>
                      <dd>{{ realtimeTraceId }}</dd>
                    </div>
                    <div v-if="realtimeRunId">
                      <dt>Run ID</dt>
                      <dd>{{ realtimeRunId }}</dd>
                    </div>
                  </dl>
                  <ul v-if="realtimeEvents.length" class="artifact-event-list">
                    <li v-for="event in realtimeEvents.slice(0, 8)" :key="`${event.timestamp}-${event.type}-${event.step ?? ''}`">
                      <span>{{ formatEventTimestamp(event.timestamp) }}</span>
                      <strong>{{ getEventLabel(event) }}</strong>
                    </li>
                  </ul>
                </section>

                <section v-if="realtimeTreeUnits.length" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>实时剧情单元</h3>
                    <n-tag size="small" :bordered="false" type="success">{{ realtimeTreeUnitCount }} 个</n-tag>
                  </div>
                  <ol class="artifact-list">
                    <li v-for="unit in realtimeTreeUnits" :key="unit.id">
                      <strong>{{ unit.title }}</strong>
                      <p>{{ unit.meta }}</p>
                      <p>{{ unit.summary }}</p>
                      <n-space v-if="unit.characters.length" size="small">
                        <n-tag v-for="character in unit.characters" :key="character" size="small" :bordered="false">
                          {{ character }}
                        </n-tag>
                      </n-space>
                    </li>
                  </ol>
                </section>

                <section v-if="realtimePlans.length" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>实时场景规划</h3>
                    <n-tag size="small" :bordered="false" type="warning">{{ realtimePlanCount }} 场</n-tag>
                  </div>
                  <ol class="artifact-list">
                    <li v-for="plan in realtimePlans" :key="plan.id">
                      <strong>{{ plan.title }}</strong>
                      <p>{{ plan.purpose }}</p>
                      <p v-if="plan.location">地点：{{ plan.location }}</p>
                      <p v-if="plan.sourceNodeIds.length">来源：{{ plan.sourceNodeIds.join('、') }}</p>
                    </li>
                  </ol>
                </section>

                <section v-if="realtimeSceneHeadings.length" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>实时场景标题</h3>
                    <n-tag size="small" :bordered="false" type="success">{{ realtimeSceneHeadingCount }} 场</n-tag>
                  </div>
                  <ol class="artifact-list">
                    <li v-for="scene in realtimeSceneHeadings" :key="scene.id">
                      <strong>{{ scene.heading }}</strong>
                    </li>
                  </ol>
                </section>

                <section v-if="generationArtifacts?.sourceIndex" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>原文索引</h3>
                    <n-tag size="small" :bordered="false" type="info">
                      {{ sourceIndexChapterCount }} 章
                    </n-tag>
                  </div>
                  <p>{{ generationArtifacts?.sourceIndex?.summary || '已接收原文索引，暂无摘要。' }}</p>
                  <n-space v-if="generationArtifacts?.sourceIndex?.characterNames?.length" size="small">
                    <n-tag
                      v-for="character in generationArtifacts.sourceIndex.characterNames"
                      :key="character"
                      size="small"
                      :bordered="false"
                    >
                      {{ character }}
                    </n-tag>
                  </n-space>
                </section>

                <section v-if="generationArtifacts?.storyBeats?.length" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>故事节拍</h3>
                    <n-tag size="small" :bordered="false" type="success">{{ storyBeatCount }} 条</n-tag>
                  </div>
                  <ol class="artifact-list">
                    <li v-for="beat in generationArtifacts.storyBeats" :key="beat.id">
                      <strong>{{ beat.title || `节拍 ${beat.order ?? beat.id}` }}</strong>
                      <p>{{ beat.summary }}</p>
                    </li>
                  </ol>
                </section>

                <section v-if="generationArtifacts?.scenePlan?.length" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>场景规划</h3>
                    <n-tag size="small" :bordered="false" type="warning">{{ scenePlanCount }} 场</n-tag>
                  </div>
                  <ol class="artifact-list">
                    <li v-for="plan in generationArtifacts.scenePlan" :key="plan.id">
                      <strong>{{ plan.title || `场景 ${plan.sceneNo ?? plan.id}` }}</strong>
                      <p>{{ plan.purpose || '该场景计划未提供目的说明。' }}</p>
                    </li>
                  </ol>
                </section>

                <section v-if="generationArtifacts?.warnings?.length" class="artifact-section warning">
                  <div class="artifact-section-heading">
                    <h3>校验警告</h3>
                    <n-tag size="small" :bordered="false" type="error">
                      {{ generationArtifacts.warnings.length }} 条
                    </n-tag>
                  </div>
                  <ul class="artifact-warning-list">
                    <li v-for="warning in generationArtifacts.warnings" :key="warning">{{ warning }}</li>
                  </ul>
                </section>

                <section v-if="generationArtifacts?.graphSnapshot" class="artifact-section">
                  <div class="artifact-section-heading">
                    <h3>图谱快照</h3>
                    <n-tag size="small" :bordered="false" type="info">GraphAfter</n-tag>
                  </div>
                  <p>{{ generationArtifacts.graphSnapshot.summary || '图谱快照已更新。' }}</p>
                  <dl v-if="graphSnapshotItems.length" class="config-grid artifact-graph-grid">
                    <div v-for="[label, value] in graphSnapshotItems" :key="label">
                      <dt>{{ label }}</dt>
                      <dd>{{ value }}</dd>
                    </div>
                  </dl>
                </section>
              </div>

              <n-empty v-else description="当前生成接口尚未返回结构化产物。" />
            </section>

            <AiSuggestionPanel
              v-else-if="activeScene"
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
  grid-template-columns: repeat(4, minmax(0, 1fr));
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

.artifact-stack {
  display: grid;
  gap: 12px;
  margin-top: 16px;
}

.artifact-section {
  padding: 12px;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.72);
}

.artifact-section.warning {
  border-color: rgba(141, 73, 56, 0.28);
  background: rgba(141, 73, 56, 0.06);
}

.artifact-section.realtime {
  border-color: rgba(98, 130, 111, 0.32);
  background: rgba(98, 130, 111, 0.08);
}

.artifact-section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.artifact-section-heading h3 {
  margin: 0;
  color: var(--color-ink);
  font-size: 14px;
}

.artifact-list,
.artifact-warning-list {
  display: grid;
  gap: 10px;
  margin: 0;
  padding-left: 18px;
}

.artifact-list li::marker,
.artifact-warning-list li::marker {
  color: var(--color-sage);
  font-weight: 800;
}

.artifact-list strong {
  color: var(--color-ink);
  font-size: 13px;
}

.artifact-list p,
.artifact-warning-list li {
  color: var(--color-muted);
  font-size: 13px;
  line-height: 1.65;
}

.artifact-list p {
  margin: 4px 0 0;
}

.artifact-graph-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.artifact-trace-grid dd {
  overflow-wrap: anywhere;
}

.artifact-event-list {
  display: grid;
  gap: 8px;
  margin: 12px 0 0;
  padding: 0;
  list-style: none;
}

.artifact-event-list li {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 8px;
  align-items: baseline;
  color: var(--color-muted);
  font-size: 12px;
  line-height: 1.5;
}

.artifact-event-list span {
  color: var(--color-sage);
  font-weight: 700;
}

.artifact-event-list strong {
  color: var(--color-ink);
  font-weight: 600;
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
