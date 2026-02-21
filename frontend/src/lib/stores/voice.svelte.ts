// Voice store using Svelte 5 runes
// Manages voice channel state via Wails bindings

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'

export interface SpeakerData {
  peer_id: string
  user_id: string
  username: string
  volume: number
  speaking: boolean
}

export interface VoiceStatusData {
  state: 'disconnected' | 'connecting' | 'connected'
  channel_id: string
  muted: boolean
  deafened: boolean
  peer_count: number
  speakers: SpeakerData[]
}

let state = $state<'disconnected' | 'connecting' | 'connected'>('disconnected')
let channelId = $state<string | null>(null)
let muted = $state(false)
let deafened = $state(false)
let speakers = $state<SpeakerData[]>([])
let error = $state<string | null>(null)

export function getVoice() {
  return {
    get state() { return state },
    get channelId() { return channelId },
    get muted() { return muted },
    get deafened() { return deafened },
    get speakers() { return speakers },
    get error() { return error },
    get connected() { return state === 'connected' },
  }
}

export async function joinVoice(voiceChannelId: string): Promise<void> {
  if (state !== 'disconnected') return

  state = 'connecting'
  error = null

  try {
    await ensureValidToken()
    await App.JoinVoice(voiceChannelId)
    state = 'connected'
    channelId = voiceChannelId
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to join voice channel'
    state = 'disconnected'
  }
}

export async function leaveVoice(): Promise<void> {
  if (state === 'disconnected') return

  try {
    await App.LeaveVoice()
  } catch (e) {
    console.error('Failed to leave voice channel:', e)
  } finally {
    state = 'disconnected'
    channelId = null
    speakers = []
    muted = false
    deafened = false
  }
}

export async function toggleMute(): Promise<void> {
  try {
    const isMuted: boolean = await App.ToggleMute()
    muted = isMuted
  } catch (e) {
    console.error('Failed to toggle mute:', e)
  }
}

export async function toggleDeafen(): Promise<void> {
  try {
    const isDeafened: boolean = await App.ToggleDeafen()
    deafened = isDeafened
    if (isDeafened) muted = true
  } catch (e) {
    console.error('Failed to toggle deafen:', e)
  }
}

export async function refreshVoiceStatus(): Promise<void> {
  try {
    const status = await App.GetVoiceStatus()
    state = status.state as VoiceStatusData['state']
    channelId = status.channel_id || null
    muted = status.muted
    deafened = status.deafened
    speakers = (status.speakers ?? []) as unknown as SpeakerData[]
  } catch (e) {
    console.error('Failed to refresh voice status:', e)
  }
}

export function resetVoice(): void {
  state = 'disconnected'
  channelId = null
  muted = false
  deafened = false
  speakers = []
  error = null
}
