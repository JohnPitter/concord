import * as App from '../../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime'

const LATEST_RELEASE_API = 'https://api.github.com/repos/JohnPitter/concord/releases/latest'
const LATEST_RELEASE_PAGE = 'https://github.com/JohnPitter/concord/releases/latest'

export interface UpdateInfo {
  platform: 'windows' | 'darwin' | 'linux' | 'unknown'
  currentVersion: string
  latestVersion: string
  available: boolean
  releaseURL: string
  assetName: string | null
  assetURL: string | null
  assetSHA256: string | null
  autoInstallSupported: boolean
}

interface GitHubRelease {
  tag_name: string
  html_url: string
  assets?: GitHubReleaseAsset[]
}

interface GitHubReleaseAsset {
  name: string
  browser_download_url: string
  digest?: string
}

function getFallbackVersion(): string {
  const envVersion = (import.meta.env.VITE_APP_VERSION as string | undefined)?.trim()
  if (envVersion) return normalizeVersion(envVersion)
  return '0.0.0'
}

function detectPlatformFromUserAgent(): 'windows' | 'darwin' | 'linux' | 'unknown' {
  const userAgent = navigator.userAgent.toLowerCase()
  if (userAgent.includes('win')) return 'windows'
  if (userAgent.includes('mac')) return 'darwin'
  if (userAgent.includes('linux')) return 'linux'
  return 'unknown'
}

function normalizePlatform(platform: string): 'windows' | 'darwin' | 'linux' | 'unknown' {
  const normalized = platform.trim().toLowerCase()
  if (normalized.startsWith('windows')) return 'windows'
  if (normalized.startsWith('darwin') || normalized.startsWith('mac')) return 'darwin'
  if (normalized.startsWith('linux')) return 'linux'
  return 'unknown'
}

function normalizeDigest(digest?: string): string | null {
  const value = (digest || '').trim().toLowerCase()
  if (!value) return null
  if (value.startsWith('sha256:')) {
    return value.slice('sha256:'.length)
  }
  return value
}

function selectReleaseAsset(
  assets: GitHubReleaseAsset[] | undefined,
  platform: 'windows' | 'darwin' | 'linux' | 'unknown',
): GitHubReleaseAsset | null {
  if (!assets || assets.length === 0) return null

  const matcher: Record<'windows' | 'darwin' | 'linux', RegExp> = {
    windows: /^concord-windows-.*\.zip$/i,
    darwin: /^concord-macos-.*\.zip$/i,
    linux: /^concord-linux-.*\.tar\.gz$/i,
  }

  if (platform === 'unknown') return null
  return assets.find((asset) => matcher[platform].test(asset.name || '')) ?? null
}

async function getCurrentVersionInfo(): Promise<{ version: string; platform: 'windows' | 'darwin' | 'linux' | 'unknown' }> {
  try {
    const info = await App.GetVersion()
    const current = normalizeVersion(info.version || '')
    const platform = normalizePlatform(info.platform || '')
    const commit = (info.git_commit || '').trim().toLowerCase()
    const buildDate = (info.build_date || '').trim().toLowerCase()

    // Guard against non-injected default values from local/dev builds.
    const looksLikePlaceholder =
      current === '1.0.0' &&
      (commit === '' || commit === 'unknown') &&
      (buildDate === '' || buildDate === 'unknown')

    if (!looksLikePlaceholder && current) {
      return {
        version: current,
        platform: platform === 'unknown' ? detectPlatformFromUserAgent() : platform,
      }
    }
    return {
      version: getFallbackVersion(),
      platform: platform === 'unknown' ? detectPlatformFromUserAgent() : platform,
    }
  } catch {
    return {
      version: getFallbackVersion(),
      platform: detectPlatformFromUserAgent(),
    }
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
  const currentInfo = await getCurrentVersionInfo()
  const currentVersion = currentInfo.version
  const release = await fetchLatestRelease()
  const latestVersion = normalizeVersion(release.tag_name || currentVersion)
  const available = compareSemver(latestVersion, currentVersion) > 0
  const asset = selectReleaseAsset(release.assets, currentInfo.platform)
  const autoInstallSupported = currentInfo.platform === 'windows' && !!asset

  return {
    platform: currentInfo.platform,
    currentVersion,
    latestVersion,
    available,
    releaseURL: release.html_url || LATEST_RELEASE_PAGE,
    assetName: asset?.name || null,
    assetURL: asset?.browser_download_url || null,
    assetSHA256: normalizeDigest(asset?.digest),
    autoInstallSupported,
  }
}

export async function installUpdate(updateInfo: UpdateInfo): Promise<void> {
  if (!updateInfo.available) {
    throw new Error('No update available.')
  }

  if (updateInfo.autoInstallSupported && updateInfo.assetURL) {
    await App.ApplyAutoUpdate(
      updateInfo.assetURL,
      updateInfo.latestVersion,
      updateInfo.assetSHA256 || '',
    )
    return
  }

  openUpdatePage(updateInfo.releaseURL)
}

export function openUpdatePage(url?: string): void {
  const target = url || LATEST_RELEASE_PAGE
  try {
    BrowserOpenURL(target)
  } catch {
    window.open(target, '_blank', 'noopener,noreferrer')
  }
}
