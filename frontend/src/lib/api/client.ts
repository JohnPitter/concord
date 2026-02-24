// HTTP API client for central server communication
// Used when networkMode === 'server' instead of Wails bindings

const API_STORAGE_KEY = 'concord-api-tokens'
const DEFAULT_REQUEST_TIMEOUT_MS = 10000
const DISCOVERY_TIMEOUT_MS = 2500
const API_UNAVAILABLE_MESSAGE = 'Servidor indisponível. Tente novamente.'

interface ApiTokens {
  accessToken: string
  refreshToken?: string
  expiresAt?: number // unix seconds
  userId: string
}

class ApiClient {
  private baseURL: string
  private tokens: ApiTokens | null = null

  constructor(baseURL: string) {
    this.baseURL = baseURL.replace(/\/$/, '')
    this.loadTokens()
  }

  setBaseURL(url: string) {
    this.baseURL = url.replace(/\/$/, '')
  }

  getBaseURL(): string {
    return this.baseURL
  }

  setTokens(tokens: ApiTokens) {
    this.tokens = tokens
    try {
      localStorage.setItem(API_STORAGE_KEY, JSON.stringify(tokens))
    } catch { /* ignore storage errors */ }
  }

  clearTokens() {
    this.tokens = null
    localStorage.removeItem(API_STORAGE_KEY)
  }

  getTokens(): ApiTokens | null {
    return this.tokens
  }

  private loadTokens() {
    try {
      const raw = localStorage.getItem(API_STORAGE_KEY)
      if (raw) this.tokens = JSON.parse(raw)
    } catch { /* ignore parse errors */ }
  }

  private async ensureToken(): Promise<string> {
    if (!this.tokens) throw new Error('Not authenticated')

    // Auto-refresh if expiring within 2 minutes
    if (this.tokens.expiresAt) {
      const now = Date.now() / 1000
      if (this.tokens.expiresAt - now < 120) {
        await this.refreshToken()
      }
    }

    return this.tokens.accessToken
  }

  private async refreshToken(): Promise<void> {
    if (!this.tokens?.userId) throw new Error('No user ID for refresh')

    const res = await fetchWithTimeout(`${this.baseURL}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: this.tokens.userId }),
    })

    if (!res.ok) {
      this.clearTokens()
      throw new Error('Session expired')
    }

    const data = await res.json()
    if (data.authenticated && data.access_token) {
      this.setTokens({
        ...this.tokens!,
        accessToken: data.access_token,
        expiresAt: data.expires_at,
      })
    }
  }

  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const token = await this.ensureToken()
    const opts: RequestInit = {
      method,
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    }
    if (body !== undefined) opts.body = JSON.stringify(body)

    const res = await fetchWithTimeout(`${this.baseURL}${path}`, opts)

    if (res.status === 204) return undefined as T

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: { message: res.statusText } }))
      throw new Error(err.error?.message ?? `HTTP ${res.status}`)
    }

    return res.json()
  }

  async publicRequest<T>(method: string, path: string, body?: unknown): Promise<T> {
    const opts: RequestInit = {
      method,
      headers: { 'Content-Type': 'application/json' },
    }
    if (body !== undefined) opts.body = JSON.stringify(body)

    const res = await fetchWithTimeout(`${this.baseURL}${path}`, opts)

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: { message: res.statusText } }))
      throw new Error(err.error?.message ?? `HTTP ${res.status}`)
    }

    return res.json()
  }

  get<T>(path: string) { return this.request<T>('GET', path) }
  post<T>(path: string, body?: unknown) { return this.request<T>('POST', path, body) }
  put<T>(path: string, body?: unknown) { return this.request<T>('PUT', path, body) }
  del<T>(path: string) { return this.request<T>('DELETE', path) }
}

// Server URL from build-time env var (VITE_SERVER_URL).
// In desktop mode, relative URLs may resolve to app:// and fail. Keep an HTTP fallback.
function defaultServerURL(): string {
  const configured = (import.meta.env.VITE_SERVER_URL as string | undefined)?.trim()
  if (configured) return configured
  return 'http://localhost'
}

async function fetchWithTimeout(input: RequestInfo | URL, init: RequestInit = {}, timeoutMs = DEFAULT_REQUEST_TIMEOUT_MS): Promise<Response> {
  const controller = new AbortController()
  let timeoutHandle: ReturnType<typeof setTimeout> | null = null

  if (init.signal) {
    if (init.signal.aborted) {
      controller.abort()
    } else {
      init.signal.addEventListener('abort', () => controller.abort(), { once: true })
    }
  }

  const fetchPromise = fetch(input, { ...init, signal: controller.signal })
  const timeoutPromise = new Promise<Response>((_, reject) => {
    timeoutHandle = setTimeout(() => {
      controller.abort()
      reject(new Error(`Request timeout after ${timeoutMs}ms`))
    }, timeoutMs)
  })

  try {
    return await Promise.race([fetchPromise, timeoutPromise])
  } catch (err) {
    throw normalizeNetworkError(err)
  } finally {
    if (timeoutHandle) clearTimeout(timeoutHandle)
  }
}

function normalizeNetworkError(err: unknown): Error {
  if (err instanceof Error) {
    const msg = err.message.toLowerCase()
    const name = err.name.toLowerCase()
    if (
      name === 'aborterror' ||
      msg.includes('failed to fetch') ||
      msg.includes('networkerror') ||
      msg.includes('load failed') ||
      msg.includes('request timeout') ||
      msg.includes('timed out')
    ) {
      return new Error(API_UNAVAILABLE_MESSAGE)
    }
    return err
  }
  return new Error(API_UNAVAILABLE_MESSAGE)
}

const SERVER_URL = defaultServerURL()

// Remote discovery gist — returns dynamic server URL when tunnel changes
const DISCOVERY_URL = 'https://gist.githubusercontent.com/JohnPitter/ee556dbee0baf301f58e908a5d1ba9b7/raw/server.json'

async function isReachable(url: string): Promise<boolean> {
  const base = url.replace(/\/$/, '')

  try {
    const res = await fetchWithTimeout(
      `${base}/health`,
      { cache: 'no-store' },
      1500,
    )
    return res.ok
  } catch {
    return false
  }
}

async function chooseFallbackURL(preferred: string): Promise<string> {
  const local = 'http://localhost'
  const candidate = preferred?.trim() ? preferred : local
  if (await isReachable(candidate)) return candidate
  if (candidate !== local && await isReachable(local)) return local
  return candidate
}

// Singleton — initialized with the build-time server URL
export const apiClient = new ApiClient(SERVER_URL)

/**
 * Fetches the current server URL from the remote discovery gist.
 * Falls back to the build-time VITE_SERVER_URL if fetch fails.
 * Updates apiClient.baseURL in-place so all subsequent calls use the new URL.
 */
export async function discoverServerURL(): Promise<string> {
  // Keep current URL as fallback; if unhealthy, swap to localhost.
  const fallback = await chooseFallbackURL(apiClient.getBaseURL())

  try {
    const res = await fetchWithTimeout(
      DISCOVERY_URL,
      { cache: 'no-store' },
      DISCOVERY_TIMEOUT_MS,
    )
    if (!res.ok) {
      apiClient.setBaseURL(fallback)
      return fallback
    }
    const data: { server_url?: string } = await res.json()
    if (data.server_url) {
      const chosen = await chooseFallbackURL(data.server_url)
      apiClient.setBaseURL(chosen)
      return chosen
    }
  } catch { /* network error — keep build-time URL */ }

  apiClient.setBaseURL(fallback)
  return fallback
}
