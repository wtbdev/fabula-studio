import { apiClient, clearAuthToken, setAuthToken } from './request'
import type { AuthTokenDTO, LoginRequest, RegisterRequest, UserDTO } from './types'

export const authApi = {
  async register(payload: RegisterRequest) {
    const data = await apiClient.post<AuthTokenDTO, RegisterRequest>('/auth/register', payload)
    setAuthToken(data.token)
    return data
  },

  async login(payload: LoginRequest) {
    const data = await apiClient.post<AuthTokenDTO, LoginRequest>('/auth/login', payload)
    setAuthToken(data.token)
    return data
  },

  me() {
    return apiClient.get<UserDTO>('/auth/me')
  },

  async logout() {
    try {
      return await apiClient.post<boolean>('/auth/logout')
    } finally {
      clearAuthToken()
    }
  },
}
