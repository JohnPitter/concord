// Settings store using Svelte 5 runes
// Persists to localStorage under 'concord-settings'

export type NetworkMode = 'p2p' | 'server'

export interface P2PProfile {
  displayName: string
  avatarDataUrl?: string  // base64 data URL da imagem local
}

export interface SettingsData {
  audioInputDevice: string
  audioOutputDevice: string
  theme: 'dark'
  notificationsEnabled: boolean
  notificationSounds: boolean
  translationSourceLang: string
  translationTargetLang: string
  networkMode?: NetworkMode | null
  p2pProfile?: P2PProfile | null
}

const STORAGE_KEY = 'concord-settings'

const defaults: SettingsData = {
  audioInputDevice: '',
  audioOutputDevice: '',
  theme: 'dark',
  notificationsEnabled: true,
  notificationSounds: true,
  translationSourceLang: 'en',
  translationTargetLang: 'pt',
}

let audioInputDevice = $state<string>(defaults.audioInputDevice)
let audioOutputDevice = $state<string>(defaults.audioOutputDevice)
let theme = $state<'dark'>(defaults.theme)
let notificationsEnabled = $state(defaults.notificationsEnabled)
let notificationSounds = $state(defaults.notificationSounds)
let translationSourceLang = $state(defaults.translationSourceLang)
let translationTargetLang = $state(defaults.translationTargetLang)
let settingsOpen = $state(false)
let networkMode = $state<NetworkMode | null>(null)
let p2pProfile = $state<P2PProfile | null>(null)

export function getSettings() {
  return {
    get audioInputDevice() { return audioInputDevice },
    get audioOutputDevice() { return audioOutputDevice },
    get theme() { return theme },
    get notificationsEnabled() { return notificationsEnabled },
    get notificationSounds() { return notificationSounds },
    get translationSourceLang() { return translationSourceLang },
    get translationTargetLang() { return translationTargetLang },
    get open() { return settingsOpen },
    get networkMode() { return networkMode },
    get p2pProfile() { return p2pProfile },
  }
}

function persist(): void {
  const data: SettingsData = {
    audioInputDevice,
    audioOutputDevice,
    theme,
    notificationsEnabled,
    notificationSounds,
    translationSourceLang,
    translationTargetLang,
    networkMode,
    p2pProfile,
  }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  } catch (e) {
    console.error('Failed to persist settings:', e)
  }
}

export function loadSettings(): void {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return

    const data: Partial<SettingsData> = JSON.parse(raw)
    if (data.audioInputDevice !== undefined) audioInputDevice = data.audioInputDevice
    if (data.audioOutputDevice !== undefined) audioOutputDevice = data.audioOutputDevice
    if (data.theme !== undefined) theme = data.theme
    if (data.notificationsEnabled !== undefined) notificationsEnabled = data.notificationsEnabled
    if (data.notificationSounds !== undefined) notificationSounds = data.notificationSounds
    if (data.translationSourceLang !== undefined) translationSourceLang = data.translationSourceLang
    if (data.translationTargetLang !== undefined) translationTargetLang = data.translationTargetLang
    if (data.networkMode !== undefined) networkMode = data.networkMode ?? null
    if (data.p2pProfile !== undefined) p2pProfile = data.p2pProfile ?? null
  } catch (e) {
    console.error('Failed to load settings:', e)
  }
}

export function saveSettings(data: Partial<SettingsData>): void {
  if (data.audioInputDevice !== undefined) audioInputDevice = data.audioInputDevice
  if (data.audioOutputDevice !== undefined) audioOutputDevice = data.audioOutputDevice
  if (data.theme !== undefined) theme = data.theme
  if (data.notificationsEnabled !== undefined) notificationsEnabled = data.notificationsEnabled
  if (data.notificationSounds !== undefined) notificationSounds = data.notificationSounds
  if (data.translationSourceLang !== undefined) translationSourceLang = data.translationSourceLang
  if (data.translationTargetLang !== undefined) translationTargetLang = data.translationTargetLang
  if (data.networkMode !== undefined) networkMode = data.networkMode ?? null
  if (data.p2pProfile !== undefined) p2pProfile = data.p2pProfile ?? null
  persist()
}

export function setAudioInput(deviceId: string): void {
  audioInputDevice = deviceId
  persist()
}

export function setAudioOutput(deviceId: string): void {
  audioOutputDevice = deviceId
  persist()
}

export function setNotifications(enabled: boolean, sounds?: boolean): void {
  notificationsEnabled = enabled
  if (sounds !== undefined) notificationSounds = sounds
  persist()
}

export function setNotificationSounds(enabled: boolean): void {
  notificationSounds = enabled
  persist()
}

export function setTranslationLangs(source: string, target: string): void {
  translationSourceLang = source
  translationTargetLang = target
  persist()
}

export function toggleSettings(): void {
  settingsOpen = !settingsOpen
}

export function openSettings(): void {
  settingsOpen = true
}

export function closeSettings(): void {
  settingsOpen = false
}

export function setNetworkMode(mode: NetworkMode): void {
  networkMode = mode
  persist()
}

export function setP2PProfile(profile: P2PProfile): void {
  p2pProfile = profile
  persist()
}
