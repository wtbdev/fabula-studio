import { apiClient } from './request'
import type { PageResult } from './request'
import type {
  CreateProjectRequest,
  ProjectDTO,
  ProjectListParams,
} from './types'

export const projectsApi = {
  create(payload: CreateProjectRequest) {
    return apiClient.post<ProjectDTO, CreateProjectRequest>('/projects', payload)
  },

  list(params?: ProjectListParams) {
    return apiClient.get<PageResult<ProjectDTO>>('/projects', { params })
  },

  detail(projectId: string) {
    return apiClient.get<ProjectDTO>(`/projects/${projectId}`)
  },

  remove(projectId: string) {
    return apiClient.delete<boolean>(`/projects/${projectId}`)
  },
}
