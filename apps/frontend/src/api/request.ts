import axios from 'axios'
import type { AxiosError, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PageParams {
  page?: number
  pageSize?: number
  keyword?: string
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export const authTokenStorageKey = 'fabula_token'

export class ApiBusinessError<T = unknown> extends Error {
  code: number
  data: T
  response: ApiResponse<T>

  constructor(response: ApiResponse<T>) {
    super(response.message)
    this.name = 'ApiBusinessError'
    this.code = response.code
    this.data = response.data
    this.response = response
  }
}

export const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '/api',
  timeout: 30_000,
})

export const getAuthToken = () => localStorage.getItem(authTokenStorageKey)

export const setAuthToken = (token: string) => {
  localStorage.setItem(authTokenStorageKey, token)
}

export const clearAuthToken = () => {
  localStorage.removeItem(authTokenStorageKey)
}

request.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAuthToken()

  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  return config
})

request.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiResponse<unknown>>) => {
    const data = error.response?.data

    if (data && typeof data.code === 'number' && typeof data.message === 'string') {
      return Promise.reject(new ApiBusinessError(data))
    }

    return Promise.reject(error)
  },
)

const unwrapResponse = <T>(response: ApiResponse<T>) => {
  if (response.code !== 0) {
    throw new ApiBusinessError(response)
  }

  return response.data
}

export const apiClient = {
  async get<T>(url: string, config?: AxiosRequestConfig) {
    const response = await request.get<ApiResponse<T>>(url, config)
    return unwrapResponse(response.data)
  },

  async post<T, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig) {
    const response = await request.post<ApiResponse<T>>(url, data, config)
    return unwrapResponse(response.data)
  },

  async patch<T, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig) {
    const response = await request.patch<ApiResponse<T>>(url, data, config)
    return unwrapResponse(response.data)
  },

  async delete<T>(url: string, config?: AxiosRequestConfig) {
    const response = await request.delete<ApiResponse<T>>(url, config)
    return unwrapResponse(response.data)
  },
}
