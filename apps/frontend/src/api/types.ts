import type { PageParams } from './request'

export type ProjectStatus = 'draft' | 'generating' | 'completed' | 'failed'
export type GenerationJobStatus = 'queued' | 'running' | 'completed' | 'failed'
export type GenerateStatus = ProjectStatus | GenerationJobStatus

export type AdaptStyle = '影视剧' | '短剧' | '舞台剧' | '广播剧' | (string & {})
export type DialogueLevel = '简略' | '适中' | '详细'
export type AdaptationMode = '忠实原文' | '适度改编' | '大胆改编'
export type SceneGranularity = '少量大场' | '适中' | '较多小场'
export type NarrationLevel = '少量保留' | '适中保留' | '大量保留'
export type ScriptBlockType = 'action' | 'dialogue' | 'narration' | 'voice_over' | 'transition'

export interface UserDTO {
  id: string
  email: string
  nickname: string
  aiPoints: number
  createdAt?: string
  updatedAt?: string
}

export interface AuthTokenDTO {
  token: string
  user: UserDTO
}

export interface RegisterRequest {
  email: string
  password: string
  nickname: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface AdaptConfig {
  style: AdaptStyle
  dialogueLevel: DialogueLevel
  adaptationMode: AdaptationMode
  sceneGranularity?: SceneGranularity
  narrationLevel?: NarrationLevel
  customPrompt?: string
}
export interface AdaptationProfile {
  style: AdaptStyle
  dialogueLevel: DialogueLevel
  adaptationMode: AdaptationMode
  sceneGranularity?: SceneGranularity
  narrationLevel?: NarrationLevel
  guidance?: string
}

export interface SourceIndexChapter {
  id?: string
  title: string
  order?: number
  summary?: string
  startOffset?: number
  endOffset?: number
}

export interface SourceIndex {
  title?: string
  chapters?: SourceIndexChapter[]
  characterNames?: string[]
  locationNames?: string[]
  summary?: string
}

export interface StoryBeat {
  id: string
  order?: number
  title?: string
  summary: string
  sourceChapterIds?: string[]
  characters?: string[]
  warnings?: string[]
}

export interface ScenePlanItem {
  id: string
  sceneNo?: number
  title?: string
  purpose?: string
  sourceBeatIds?: string[]
  sourceNodeIds?: string[]
  characters?: string[]
  location?: string
  warnings?: string[]
}

export interface GraphSnapshotSummary {
  nodeCount?: number
  edgeCount?: number
  characterCount?: number
  relationshipCount?: number
  summary?: string
  updatedAt?: string
}

export interface GenerationArtifacts {
  sourceIndex?: SourceIndex
  storyBeats?: StoryBeat[]
  scenePlan?: ScenePlanItem[]
  warnings?: string[]
  graphSnapshot?: GraphSnapshotSummary
}
export interface GenerationJobDTO {
  id: string
  projectId: string
  status: GenerationJobStatus
  progress?: number
  currentStep?: string
  errorMessage?: string | null
  artifacts?: GenerationArtifacts
  startedAt?: string | null
  completedAt?: string | null
  createdAt?: string
  updatedAt?: string
}

export interface PipelineEventDTO {
  type: string
  timestamp: string
  step?: string
  node_id?: string
  message: string
  details?: Record<string, unknown>
  duration?: number | string | null
  error?: string
  projectId?: string
  jobId?: string
  runId?: string
  traceId?: string
  progress?: number
}

export interface GenerateProjectRequest {
  config?: AdaptConfig
  adaptationProfile?: AdaptationProfile
}


export interface ProjectDTO {
  id: string
  userId?: string
  title: string
  novelTitle?: string
  sourceText?: string
  config?: AdaptConfig
  adaptationProfile?: AdaptationProfile
  artifacts?: GenerationArtifacts
  status: ProjectStatus
  errorMessage?: string | null
  sceneCount?: number
  createdAt: string
  updatedAt: string
}

export interface CreateProjectRequest {
  title: string
  novelTitle?: string
  sourceText: string
  config: AdaptConfig
  adaptationProfile?: AdaptationProfile
}

export interface UpdateProjectRequest {
  title?: string
  novelTitle?: string
}

export type ProjectListParams = PageParams

export interface ScriptBlock {
  type: ScriptBlockType
  character?: string
  content: string
}

export interface SceneRawJson {
  characters?: string[]
  script?: ScriptBlock[]
  source?: {
    chapters?: string[]
    summary?: string
  }
}

export interface SceneDTO {
  id: string
  projectId: string
  sceneNo: number
  title: string
  location?: string
  timeText?: string
  summary?: string
  content: string
  rawJson?: SceneRawJson
  createdAt?: string
  updatedAt: string
}

export interface GenerateProjectResponse {
  projectId: string
  jobId?: string
  job?: GenerationJobDTO
  status: GenerateStatus
  costPoints?: number
  remainingPoints?: number
  scenes?: SceneDTO[]
  artifacts?: GenerationArtifacts
}

export interface GenerateStatusDTO {
  projectId: string
  jobId?: string
  job?: GenerationJobDTO
  status: GenerateStatus
  progress?: number
  currentStep?: string
  errorMessage?: string | null
  artifacts?: GenerationArtifacts
  scenes?: SceneDTO[]
}

export interface UpdateSceneRequest {
  title?: string
  location?: string
  timeText?: string
  summary?: string
  content: string
  versionSource?: 'manual_save' | 'ai_regenerate' | 'restore'
}

export interface UpdateSceneResponse {
  id: string
  updatedAt: string
}

export type SceneRegenerationMode = 'replace' | 'polish' | 'shorten' | 'expand'

export interface GenerateSceneRegenerationRequest {
  instruction?: string
  mode?: SceneRegenerationMode
}

export interface GenerateSceneRegenerationResponse {
  sceneId: string
  originalContent: string
  regeneratedContent: string
  instruction: string
  costPoints: number
  remainingPoints: number
}

