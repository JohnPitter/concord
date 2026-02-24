import * as App from '../../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime'

const LATEST_RELEASE_API = 'https://api.github.com/repos/JohnPitter/concord/releases/latest'
const LATEST_RELEASE_PAGE = 'https://github.com/JohnPitter/concord/releases/latest'

export interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  available: boolean
  releaseURL: string
}

interface GitHubRelease {
  tag_name: string
  html_url: string
}

function getFallbackVersion(): string {
  const envVersion = (import.meta.env.VITE_APP_VERSION as string | undefined)?.trim()
  if (envVersion) return normalizeVersion(envVersion)
  return '0.0.0'
}

async function getCurrentVersion(): Promise<string> {
  try {
    const info = await App.GetVersion()
    const current = normalizeVersion(info.version || '')
    const commit = (info.git_commit || '').trim().toLowerCase()
    const buildDate = (info.build_date || '').trim().toLowerCase()

    // Guard against non-injected default values from local/dev builds.
    const looksLikePlaceholder =
      current === '1.0.0' &&
      (commit === '' || commit === 'unknown') &&
      (buildDate === '' || buildDate === 'unknown')

    if (!looksLikePlaceholder && current) {
      return current
    }
    return getFallbackVersion()
  } catch {
    return getFallbackVersion()
  }
}

async function fetchLatestRelease(): Promise<GitHubRelease> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 8000)
  try {
    const res = await fetch(LATEST_RELEASE_API, {
      cache: 'no-store',
      signal: controller.signal,
      headers: {
        Accept: 'application/vnd.github+json',
      },
    })
    if (!res.ok) {
      if (res.status === 403) {
        throw new Error('GitHub rate limit reached. Try again in a few minutes.')
      }
      throw new Error(`failed to fetch latest release: HTTP ${res.status}`)
    }
    return res.json() as Promise<GitHubRelease>
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') {
      throw new Error('Update check timed out. Please try again.')
    }
    throw e
  } finally {
    clearTimeout(timer)
  }
}

function normalizeVersion(v: string): string {
  return v.trim().replace(/^v/i, '')
}

function compareSemver(a: string, b: string): number {
  const pa = normalizeVersion(a).split('.').map((p) => Number.parseInt(p, 10) || 0)
  const pb = normalizeVersion(b).split('.').map((p) => Number.parseInt(p, 10) || 0)
  const size = Math.max(pa.length, pb.length)
  for (let i = 0; i < size; i++) {
    const av = pa[i] ?? 0
    const bv = pb[i] ?? 0
    if (av > bv) return 1
    if (av < bv) return -1
  }
  return 0
}

export async function checkForUpdates(): Promise<UpdateInfo> {
  const currentVersion = await getCurrentVersion()
  const release = await fetchLatestRelease()
  const latestVersion = normalizeVersion(release.tag_name || currentVersion)
  const available = compareSemver(latestVersion, currentVersion) > 0

  return {
    currentVersion,
    latestVersion,
    available,
    releaseURL: release.html_url || LATEST_RELEASE_PAGE,
  }
}

export function openUpdatePage(url?: string): void {
  const target = url || LATEST_RELEASE_PAGE
  try {
    const result = BrowserOpenURL(target)
    if (result && typeof (result as Promise<unknown>).catch === 'function') {
      ;(result as Promise<unknown>).catch(() => {
        window.open(target, '_blank', 'noopener,noreferrer')
      })
    }
  } catch {
    window.open(target, '_blank', 'noopener,noreferrer')
  }
}
