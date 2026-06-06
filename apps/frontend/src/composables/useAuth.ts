import { computed, reactive } from 'vue'
import { authApi } from '../api'
import { clearAuthToken, getAuthToken } from '../api/request'
import type { AuthTokenDTO, LoginRequest, RegisterRequest, UserDTO } from '../api'

interface AuthState {
  token: string | null
  user: UserDTO | null
  isReady: boolean
}

const authState = reactive<AuthState>({
  token: getAuthToken(),
  user: null,
  isReady: false,
})

const syncToken = () => {
  authState.token = getAuthToken()
}

const clearAuthState = () => {
  clearAuthToken()
  authState.token = null
  authState.user = null
}

const applyAuthToken = (data: AuthTokenDTO) => {
  syncToken()
  authState.user = data.user
  authState.isReady = true
}

export const useAuth = () => {
  const isLoggedIn = computed(() => Boolean(authState.token && authState.user))

  const fetchMe = async () => {
    syncToken()

    if (!authState.token) {
      authState.user = null
      authState.isReady = true
      return null
    }

    try {
      const user = await authApi.me()
      authState.user = user
      return user
    } catch (error) {
      clearAuthState()
      throw error
    } finally {
      authState.isReady = true
    }
  }

  const login = async (payload: LoginRequest) => {
    const data = await authApi.login(payload)
    applyAuthToken(data)
    return (await fetchMe()) ?? data.user
  }

  const register = async (payload: RegisterRequest) => {
    const data = await authApi.register(payload)
    applyAuthToken(data)
    return data.user
  }

  const logout = async () => {
    try {
      await authApi.logout()
    } finally {
      clearAuthState()
      authState.isReady = true
    }
  }

  return {
    authState,
    isLoggedIn,
    login,
    register,
    fetchMe,
    logout,
  }
}
