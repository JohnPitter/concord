// HTTP API client for central server communication
// Used when networkMode === 'server' instead of Wails bindings

const API_STORAGE_KEY = 'concord-api-tokens'

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

    const res = await fetch(`${this.baseURL}/api/v1/auth/refresh`, {
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

    const res = await fetch(`${this.baseURL}${path}`, opts)

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

    const res = await fetch(`${this.baseURL}${path}`, opts)

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

// Singleton
export const apiClient = new ApiClient('')
