// Auth store using Svelte 5 runes
// Manages authentication state and communicates with Go backend via Wails bindings

import * as App from '../../../wailsjs/go/main/App'

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

let authenticated = $state(false)
let user = $state<User | null>(null)
let accessToken = $state<string | null>(null)
let loading = $state(true)
let error = $state<string | null>(null)

// Device flow state
let deviceCode = $state<DeviceCodeResponse | null>(null)
let polling = $state(false)

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

export async function initAuth(): Promise<void> {
  loading = true
  error = null

  try {
    const savedUserID = localStorage.getItem(USER_ID_KEY)
    if (!savedUserID) {
      loading = false
      return
    }

    const state: AuthState = await App.RestoreSession(savedUserID)
    if (state.authenticated && state.user) {
      authenticated = true
      user = state.user
      accessToken = state.access_token ?? null
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
    const response: DeviceCodeResponse = await App.StartLogin()
    deviceCode = response
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to start login'
  }
}

export async function pollForCompletion(): Promise<void> {
  if (!deviceCode) return
  polling = true
  error = null

  try {
    const state: AuthState = await App.CompleteLogin(
      deviceCode.device_code,
      deviceCode.interval
    )

    if (state.authenticated && state.user) {
      authenticated = true
      user = state.user
      accessToken = state.access_token ?? null
      deviceCode = null
      localStorage.setItem(USER_ID_KEY, state.user.id)
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
      await App.Logout(user.id)
    }
  } catch (e) {
    console.error('Logout error:', e)
  } finally {
    authenticated = false
    user = null
    accessToken = null
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
