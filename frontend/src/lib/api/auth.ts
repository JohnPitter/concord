// Auth API functions for central server communication

import { apiClient } from './client'

export async function apiStartLogin() {
  return apiClient.publicRequest<{
    device_code: string
    user_code: string
    verification_uri: string
    expires_in: number
    interval: number
  }>('POST', '/api/v1/auth/device-code')
}

export async function apiCompleteLogin(deviceCode: string, interval: number) {
  return apiClient.publicRequest<{
    authenticated: boolean
    user?: {
      id: string
      github_id: number
      username: string
      display_name: string
      avatar_url: string
    }
    access_token?: string
    expires_at?: number
  }>('POST', '/api/v1/auth/token', {
    device_code: deviceCode,
    interval,
  })
}

export async function apiRefreshSession(userId: string) {
  return apiClient.publicRequest<{
    authenticated: boolean
    user?: {
      id: string
      github_id: number
      username: string
      display_name: string
      avatar_url: string
    }
    access_token?: string
    expires_at?: number
  }>('POST', '/api/v1/auth/refresh', {
    user_id: userId,
  })
}
