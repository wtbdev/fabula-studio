import type { PageParams } from './request'

export type ProjectStatus = 'draft' | 'generating' | 'completed' | 'failed'
export type SceneSuggestionType =
  | 'dialogue'
  | 'conflict'
  | 'rhythm'
  | 'character'
  | 'structure'
  | 'visual'
export type SceneSuggestionStatus = 'pending' | 'accepted' | 'dismissed'

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

export interface ProjectDTO {
  id: string
  userId?: string
  title: string
  novelTitle?: string
  sourceText?: string
  config?: AdaptConfig
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
  status: ProjectStatus
  costPoints: number
  remainingPoints: number
  scenes: SceneDTO[]
}

export interface GenerateStatusDTO {
  projectId: string
  status: ProjectStatus
  progress: number
  currentStep: string
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

export interface SceneSuggestion {
  id: string
  projectId: string
  sceneId: string
  type: SceneSuggestionType
  title: string
  problem: string
  reason: string
  suggestion: string
  applyText?: string
  status: SceneSuggestionStatus
  createdAt: string
  updatedAt: string
}

export interface SceneSuggestionListParams {
  status?: SceneSuggestionStatus | 'all'
}

export interface GenerateSceneSuggestionsRequest {
  content: string
  count?: number
}

export interface GenerateSceneSuggestionsResponse {
  costPoints: number
  remainingPoints: number
  suggestions: SceneSuggestion[]
}

export interface UpdateSceneSuggestionRequest {
  status: Exclude<SceneSuggestionStatus, 'pending'>
}

export interface UpdateSceneSuggestionResponse {
  id: string
  status: Exclude<SceneSuggestionStatus, 'pending'>
  updatedAt: string
}
