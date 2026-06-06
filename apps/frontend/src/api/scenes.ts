import { apiClient } from './request'
import type { SceneDTO, UpdateSceneRequest, UpdateSceneResponse } from './types'

export const scenesApi = {
  list(projectId: string) {
    return apiClient.get<SceneDTO[]>(`/projects/${projectId}/scenes`)
  },

  detail(sceneId: string) {
    return apiClient.get<SceneDTO>(`/scenes/${sceneId}`)
  },

  update(sceneId: string, payload: UpdateSceneRequest) {
    return apiClient.patch<UpdateSceneResponse, UpdateSceneRequest>(`/scenes/${sceneId}`, payload)
  },
}
