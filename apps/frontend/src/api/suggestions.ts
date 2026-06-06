import { apiClient } from './request'
import type {
  GenerateSceneSuggestionsRequest,
  GenerateSceneSuggestionsResponse,
  SceneSuggestion,
  SceneSuggestionListParams,
  UpdateSceneSuggestionRequest,
  UpdateSceneSuggestionResponse,
} from './types'

export const suggestionsApi = {
  list(sceneId: string, params: SceneSuggestionListParams = { status: 'pending' }) {
    return apiClient.get<SceneSuggestion[]>(`/scenes/${sceneId}/suggestions`, {
      params,
    })
  },

  generate(sceneId: string, payload: GenerateSceneSuggestionsRequest) {
    return apiClient.post<GenerateSceneSuggestionsResponse, GenerateSceneSuggestionsRequest>(
      `/scenes/${sceneId}/suggestions`,
      payload,
    )
  },

  update(suggestionId: string, payload: UpdateSceneSuggestionRequest) {
    return apiClient.patch<UpdateSceneSuggestionResponse, UpdateSceneSuggestionRequest>(
      `/suggestions/${suggestionId}`,
      payload,
    )
  },
}
