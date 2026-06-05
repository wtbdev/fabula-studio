import { apiClient } from './request'
import type { GenerateProjectResponse, GenerateStatusDTO } from './types'

export const generationApi = {
  generate(projectId: string) {
    return apiClient.post<GenerateProjectResponse, Record<string, never>>(
      `/projects/${projectId}/generate`,
      {},
    )
  },

  status(projectId: string) {
    return apiClient.get<GenerateStatusDTO>(`/projects/${projectId}/generate/status`)
  },
}
