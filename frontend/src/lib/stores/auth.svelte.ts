// Auth store using Svelte 5 runes
// Manages authentication state — Wails bindings (P2P) or HTTP API (Server)

import * as App from '../../../wailsjs/go/main/App'
import { isServerMode } from '../api/mode'
import { apiStartLogin, apiCompleteLogin, apiRefreshSession } from '../api/auth'
import { apiClient, discoverServerURL } from '../api/client'

interface User {
  id: string
  github_id: number
  username: string
  display_name: string
  avatar_url: string
}

interface AuthState {
  authenticated: boolean
  user?: User
  access_token?: string
  expires_at?: number
}

interface DeviceCodeResponse {
  device_code: string
  user_code: string
  verification_uri: string
  expires_in: number
  interval: number
}

// Persisted user ID key for session restore
const USER_ID_KEY = 'concord_user_id'

// Refresh token 2 minutes before expiry
const REFRESH_BUFFER_MS = 2 * 60 * 1000
const SERVER_DISCOVERY_MAX_WAIT_MS = 3500

let authenticated = $state(false)
let user = $state<User | null>(null)
let accessToken = $state<string | null>(null)
let expiresAt = $state<number | null>(null)
let loading = $state(true)
let error = $state<string | null>(null)

// Device flow state
let deviceCode = $state<DeviceCodeResponse | null>(null)
let polling = $state(false)

// Refresh timer
let refreshTimer: ReturnType<typeof setTimeout> | null = null

export function getAuth() {
  return {
    get authenticated() { return authenticated },
    get user() { return user },
    get accessToken() { return accessToken },
    get loading() { return loading },
    get error() { return error },
    get deviceCode() { return deviceCode },
    get polling() { return polling },
  }
}

function scheduleRefresh(): void {
  if (refreshTimer) {
    clearTimeout(refreshTimer)
    refreshTimer = null
  }
  if (!expiresAt || !user) return

  const now = Date.now()
  const expiresMs = expiresAt * 1000
  const delay = Math.max(expiresMs - now - REFRESH_BUFFER_MS, 0)

  refreshTimer = setTimeout(async () => {
    await refreshAccessToken()
  }, delay)
}

async function refreshAccessToken(): Promise<boolean> {
  if (!user) return false

  try {
    if (isServerMode()) {
      const state = await apiRefreshSession(user.id)
      if (state.authenticated && state.user) {
        user = state.user
        accessToken = state.access_token ?? null
        expiresAt = state.expires_at ?? null
        apiClient.setTokens({
          accessToken: state.access_token!,
          expiresAt: state.expires_at,
          userId: state.user.id,
        })
        scheduleRefresh()
        return true
      }
    } else {
      const state: AuthState = await App.RestoreSession(user.id)
      if (state.authenticated && state.user) {
        user = state.user
        accessToken = state.access_token ?? null
        expiresAt = state.expires_at ?? null
        scheduleRefresh()
        return true
      }
    }
    // Session expired — force logout
    await logout()
    return false
  } catch (e) {
    console.error('Failed to refresh token:', e)
    return false
  }
}

/**
 * Ensures the access token is valid before making a backend call.
 * If the token expires within REFRESH_BUFFER_MS, it refreshes first.
 * Returns true if the token is valid, false if refresh failed.
 */
export async function ensureValidToken(): Promise<boolean> {
  if (!authenticated || !user) return false

  // In server mode, apiClient handles its own token refresh
  if (isServerMode()) return true

  if (!expiresAt) return true // No expiry info, assume valid

  const now = Date.now()
  const expiresMs = expiresAt * 1000

  if (expiresMs - now < REFRESH_BUFFER_MS) {
    return await refreshAccessToken()
  }
  return true
}

export async function initAuth(): Promise<void> {
  loading = true
  error = null

  try {
    // Discover latest server URL before any API call
    if (isServerMode()) {
      await Promise.race([
        discoverServerURL().catch(() => undefined),
        new Promise<void>((resolve) => setTimeout(resolve, SERVER_DISCOVERY_MAX_WAIT_MS)),
      ])
    }

    const savedUserID = localStorage.getItem(USER_ID_KEY)
    if (!savedUserID) {
      loading = false
      return
    }

    let state: AuthState
    if (isServerMode()) {
      state = await apiRefreshSession(savedUserID)
      if (state.authenticated && state.user && state.access_token) {
        apiClient.setTokens({
          accessToken: state.access_token,
          expiresAt: state.expires_at,
          userId: state.user.id,
        })
      }
    } else {
      state = await App.RestoreSession(savedUserID)
    }

    if (state.authenticated && state.user) {
      authenticated = true
      user = state.user
      accessToken = state.access_token ?? null
      expiresAt = state.expires_at ?? null
      scheduleRefresh()
    } else {
      localStorage.removeItem(USER_ID_KEY)
    }
  } catch (e) {
    console.error('Failed to restore session:', e)
    localStorage.removeItem(USER_ID_KEY)
  } finally {
    loading = false
  }
}

export async function startLogin(): Promise<void> {
  error = null
  polling = false

  try {
    // Ensure we have the latest server URL
    if (isServerMode()) {
      await discoverServerURL()
    }

    let response: DeviceCodeResponse
    if (isServerMode()) {
      response = await apiStartLogin()
    } else {
      response = await App.StartLogin()
    }
    deviceCode = response
  } catch (e) {
    error = e instanceof Error ? e.message : typeof e === 'string' ? e : 'Failed to start login'
  }
}

export async function pollForCompletion(): Promise<void> {
  if (!deviceCode) return
  polling = true
  error = null

  try {
    let state: AuthState
    if (isServerMode()) {
      state = await apiCompleteLogin(deviceCode.device_code, deviceCode.interval)
      if (state.authenticated && state.user && state.access_token) {
        apiClient.setTokens({
          accessToken: state.access_token,
          expiresAt: state.expires_at,
          userId: state.user.id,
        })
      }
    } else {
      state = await App.CompleteLogin(
        deviceCode.device_code,
        deviceCode.interval
      )
    }

    if (state.authenticated && state.user) {
      authenticated = true
      user = state.user
      accessToken = state.access_token ?? null
      expiresAt = state.expires_at ?? null
      deviceCode = null
      localStorage.setItem(USER_ID_KEY, state.user.id)
      scheduleRefresh()
    }
  } catch (e) {
    error = e instanceof Error ? e.message : 'Login failed'
  } finally {
    polling = false
  }
}

export async function logout(): Promise<void> {
  try {
    if (user) {
      if (isServerMode()) {
        apiClient.clearTokens()
      } else {
        await App.Logout(user.id)
      }
    }
  } catch (e) {
    console.error('Logout error:', e)
  } finally {
    if (refreshTimer) {
      clearTimeout(refreshTimer)
      refreshTimer = null
    }
    authenticated = false
    user = null
    accessToken = null
    expiresAt = null
    deviceCode = null
    polling = false
    error = null
    localStorage.removeItem(USER_ID_KEY)
  }
}

export function clearError(): void {
  error = null
}

export function cancelLogin(): void {
  deviceCode = null
  polling = false
  error = null
}
