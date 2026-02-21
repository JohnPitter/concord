// Voice store using Svelte 5 runes
// Manages voice channel state via Wails bindings

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'

export interface SpeakerData {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  volume: number
  speaking: boolean
  screenSharing?: boolean
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
let noiseSuppression = $state(true)
let screenSharing = $state(false)
let speakers = $state<SpeakerData[]>([])
let error = $state<string | null>(null)
let joinedAt = $state<number | null>(null)
let elapsedSeconds = $state(0)
let timerInterval: ReturnType<typeof setInterval> | null = null

function startTimer() {
  joinedAt = Date.now()
  elapsedSeconds = 0
  if (timerInterval) clearInterval(timerInterval)
  timerInterval = setInterval(() => {
    if (joinedAt) elapsedSeconds = Math.floor((Date.now() - joinedAt) / 1000)
  }, 1000)
}

function stopTimer() {
  if (timerInterval) { clearInterval(timerInterval); timerInterval = null }
  joinedAt = null
  elapsedSeconds = 0
}

function formatElapsed(s: number): string {
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const pad = (n: number) => n.toString().padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${pad(m)}:${pad(sec)}`
}

export function getVoice() {
  return {
    get state() { return state },
    get channelId() { return channelId },
    get muted() { return muted },
    get deafened() { return deafened },
    get noiseSuppression() { return noiseSuppression },
    get screenSharing() { return screenSharing },
    get speakers() {
      // Enrich speakers reactively: local user gets client-side VAD + screen sharing
      return speakers.map(s => ({
        ...s,
        speaking: s.speaking || (localSpeaking && isLocalUser(s)),
        screenSharing: isLocalUser(s) ? screenSharing : false,
      }))
    },
    get localSpeaking() { return localSpeaking },
    get error() { return error },
    get connected() { return state === 'connected' },
    get elapsed() { return formatElapsed(elapsedSeconds) },
  }
}

// Client-side voice activity detection via Web Audio API
let audioContext: AudioContext | null = null
let analyserNode: AnalyserNode | null = null
let mediaStream: MediaStream | null = null
let vadInterval: ReturnType<typeof setInterval> | null = null
let localSpeaking = $state(false)

async function startVoiceActivityDetection() {
  try {
    mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true })
    audioContext = new AudioContext()
    const source = audioContext.createMediaStreamSource(mediaStream)
    analyserNode = audioContext.createAnalyser()
    analyserNode.fftSize = 256
    analyserNode.smoothingTimeConstant = 0.3
    analyserNode.minDecibels = -90
    analyserNode.maxDecibels = -10
    source.connect(analyserNode)

    const dataArray = new Uint8Array(analyserNode.frequencyBinCount)

    // Use requestAnimationFrame for smooth, responsive VAD
    function checkLevel() {
      if (!analyserNode) return
      if (muted || deafened) {
        if (localSpeaking) localSpeaking = false
      } else {
        analyserNode.getByteTimeDomainData(dataArray)
        // Compute RMS (root mean square) for better amplitude detection
        let sumSquares = 0
        for (let i = 0; i < dataArray.length; i++) {
          const normalized = (dataArray[i] - 128) / 128
          sumSquares += normalized * normalized
        }
        const rms = Math.sqrt(sumSquares / dataArray.length)
        const newSpeaking = rms > 0.02
        if (newSpeaking !== localSpeaking) localSpeaking = newSpeaking
      }
      vadInterval = requestAnimationFrame(checkLevel) as unknown as ReturnType<typeof setInterval>
    }
    vadInterval = requestAnimationFrame(checkLevel) as unknown as ReturnType<typeof setInterval>
  } catch (e) {
    console.error('Failed to start voice activity detection:', e)
  }
}

function stopVoiceActivityDetection() {
  if (vadInterval) { cancelAnimationFrame(vadInterval as unknown as number); vadInterval = null }
  if (mediaStream) { mediaStream.getTracks().forEach(t => t.stop()); mediaStream = null }
  if (audioContext) { audioContext.close().catch(() => {}); audioContext = null }
  analyserNode = null
  localSpeaking = false
}

// Screen sharing via getDisplayMedia
let screenStream: MediaStream | null = null

export async function startScreenShare(): Promise<boolean> {
  try {
    screenStream = await navigator.mediaDevices.getDisplayMedia({
      video: true,
      audio: true,
    })
    screenSharing = true
    // Auto-stop when user clicks "Stop sharing" in browser UI
    screenStream.getVideoTracks()[0]?.addEventListener('ended', () => {
      stopScreenShare()
    })
    return true
  } catch {
    // User cancelled the picker
    screenSharing = false
    return false
  }
}

export function stopScreenShare(): void {
  if (screenStream) {
    screenStream.getTracks().forEach(t => t.stop())
    screenStream = null
  }
  screenSharing = false
}

// Sound notification for voice join/leave events
let previousSpeakerCount = 0

function playJoinSound() {
  try {
    const ctx = new AudioContext()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.frequency.setValueAtTime(800, ctx.currentTime)
    osc.frequency.setValueAtTime(1000, ctx.currentTime + 0.08)
    gain.gain.setValueAtTime(0.15, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.2)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.2)
    osc.onended = () => ctx.close()
  } catch { /* audio not available */ }
}

function playLeaveSound() {
  try {
    const ctx = new AudioContext()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.frequency.setValueAtTime(600, ctx.currentTime)
    osc.frequency.setValueAtTime(400, ctx.currentTime + 0.08)
    gain.gain.setValueAtTime(0.12, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.2)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.2)
    osc.onended = () => ctx.close()
  } catch { /* audio not available */ }
}

let voicePolling: ReturnType<typeof setInterval> | null = null

function startVoicePolling() {
  if (voicePolling) return
  voicePolling = setInterval(() => refreshVoiceStatus(), 2000)
}

function stopVoicePolling() {
  if (voicePolling) {
    clearInterval(voicePolling)
    voicePolling = null
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
    startTimer()
    startVoicePolling()
    startVoiceActivityDetection()
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
    stopVoicePolling()
    stopVoiceActivityDetection()
    stopScreenShare()
    stopTimer()
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

export function toggleNoiseSuppression(): void {
  noiseSuppression = !noiseSuppression
}

export async function toggleScreenSharing(): Promise<void> {
  if (screenSharing) {
    stopScreenShare()
  } else {
    await startScreenShare()
  }
}

export async function refreshVoiceStatus(): Promise<void> {
  try {
    const status = await App.GetVoiceStatus()
    state = status.state as VoiceStatusData['state']
    channelId = status.channel_id || null
    muted = status.muted
    deafened = status.deafened
    const newSpeakers = (status.speakers ?? []) as unknown as SpeakerData[]
    // Play join/leave sound when speaker count changes (only while we're connected)
    if (state === 'connected' && !deafened) {
      const oldCount = previousSpeakerCount
      const newCount = newSpeakers.length
      if (newCount > oldCount && oldCount > 0) playJoinSound()
      else if (newCount < oldCount && newCount > 0) playLeaveSound()
    }
    previousSpeakerCount = newSpeakers.length
    speakers = newSpeakers
  } catch (e) {
    console.error('Failed to refresh voice status:', e)
  }
}

// Track the local user's username for VAD matching
let localUsername: string | null = null

export function setLocalUsername(username: string): void {
  localUsername = username
}

function isLocalUser(speaker: SpeakerData): boolean {
  if (!localUsername) return false
  return speaker.username === localUsername
}

export function resetVoice(): void {
  state = 'disconnected'
  channelId = null
  muted = false
  deafened = false
  speakers = []
  error = null
}
