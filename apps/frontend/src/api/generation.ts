import { apiClient } from './request'
import type { GenerateProjectRequest, GenerateProjectResponse, GenerateStatusDTO } from './types'

export const generationApi = {
  generate(projectId: string, payload: GenerateProjectRequest = {}) {
    return apiClient.post<GenerateProjectResponse, GenerateProjectRequest>(
      `/projects/${projectId}/generate`,
      payload,
    )
  },

  status(projectId: string) {
    return apiClient.get<GenerateStatusDTO>(`/projects/${projectId}/generate/status`)
  },
}
