// Voice store using Svelte 5 runes
// Manages voice channel state via Wails bindings

import * as App from '../../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { ensureValidToken } from './auth.svelte'
import { apiClient } from '../api/client'
import { isServerMode } from '../api/mode'
import { getSettings } from './settings.svelte'
import { VoiceRTCClient, type VoiceRTCStatus } from '../services/voiceRTC'

export interface SpeakerData {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  volume: number
  speaking: boolean
  screenSharing?: boolean
  muted?: boolean
  deafened?: boolean
}

export interface VoiceStatusData {
  state: 'disconnected' | 'connecting' | 'connected'
  channel_id: string
  muted: boolean
  deafened: boolean
  peer_count: number
  speakers: SpeakerData[]
  channel_started_at?: number
}

let state = $state<'disconnected' | 'connecting' | 'connected'>('disconnected')
let channelId = $state<string | null>(null)
let muted = $state(false)
let deafened = $state(false)
let noiseSuppression = $state(true)
let screenSharing = $state(false)
let speakers = $state<SpeakerData[]>([])
let error = $state<string | null>(null)
let channelStartedAt = $state<number | null>(null)
let joinedAt = $state<number | null>(null)
let elapsedSeconds = $state(0)
let timerInterval: ReturnType<typeof setInterval> | null = null
let rtcClient: VoiceRTCClient | null = null

function startTimer(startAt?: number) {
  const now = Date.now()
  if (typeof startAt === 'number' && Number.isFinite(startAt) && startAt > 0) {
    joinedAt = startAt
  } else {
    joinedAt = now
  }
  if (joinedAt && joinedAt > now) joinedAt = now
  elapsedSeconds = joinedAt ? Math.max(0, Math.floor((now - joinedAt) / 1000)) : 0
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
      // channelParticipants polling provides a backup source for mute/deaf state
      // in all modes (server + P2P), ensuring indicators stay visible.
      const channelPeers = channelId
        ? (channelParticipants[channelId] ?? [])
        : []

      return speakers.map(s => {
        const local = isLocalUser(s)
        let merged: SpeakerData = {
          ...s,
          speaking: s.speaking || (localSpeaking && local),
          screenSharing: local ? screenSharing : false,
          muted: local ? muted : s.muted,
          deafened: local ? deafened : s.deafened,
        }

        if (!local && channelPeers.length > 0) {
          const peer = channelPeers.find(p =>
            (p.user_id && p.user_id === s.user_id) ||
            (p.peer_id && p.peer_id === s.peer_id) ||
            (p.username && p.username === s.username),
          )
          if (peer) {
            merged = {
              ...merged,
              muted: peer.muted ?? merged.muted,
              deafened: peer.deafened ?? merged.deafened,
            }
          }
        }

        return merged
      })
    },
    get localSpeaking() { return localSpeaking },
    get error() { return error },
    get connected() { return state === 'connected' },
    get elapsed() { return formatElapsed(elapsedSeconds) },
  }
}

function normalizeSpeaker(raw: Partial<SpeakerData>): SpeakerData {
  const peerID = (raw.peer_id || '').trim()
  const userID = (raw.user_id || '').trim() || peerID
  const usernameRaw = (raw.username || '').trim()
  const username = usernameRaw || userID.slice(0, 12) || peerID.slice(0, 12) || 'user'
  return {
    peer_id: peerID,
    user_id: userID,
    username,
    avatar_url: (raw.avatar_url || '').trim(),
    volume: Number.isFinite(raw.volume as number) ? Number(raw.volume) : 0,
    speaking: !!raw.speaking,
    screenSharing: !!raw.screenSharing,
    muted: !!raw.muted,
    deafened: !!raw.deafened,
  }
}

function sortSpeakersStable(list: SpeakerData[]): SpeakerData[] {
  return [...list].sort((a, b) => {
    const aLocal = isLocalUser(a) ? 0 : 1
    const bLocal = isLocalUser(b) ? 0 : 1
    if (aLocal !== bLocal) return aLocal - bLocal
    const byName = a.username.localeCompare(b.username)
    if (byName !== 0) return byName
    return (a.peer_id || a.user_id).localeCompare(b.peer_id || b.user_id)
  })
}

function applyVoiceStatus(status: VoiceStatusData): void {
  state = status.state as VoiceStatusData['state']
  channelId = status.channel_id || null
  muted = status.muted
  deafened = status.deafened
  if (state === 'disconnected') {
    stopTimer()
    channelStartedAt = null
  }
  const startedAt = typeof status.channel_started_at === 'number' ? status.channel_started_at : null
  if (startedAt && startedAt > 0) {
    const changed = channelStartedAt !== startedAt
    channelStartedAt = startedAt
    // Restart timer with authoritative server timestamp when it changes
    if (changed && state === 'connected') startTimer(channelStartedAt)
  }
  // Start timer if connected with server timestamp but timer not yet ticking
  if (state === 'connected' && channelStartedAt && !timerInterval) {
    startTimer(channelStartedAt)
  }
  const normalized = (status.speakers ?? []).map(s => normalizeSpeaker(s))
  const newSpeakers = sortSpeakersStable(normalized)

  if (state === 'connected' && !deafened) {
    const oldCount = previousSpeakerCount
    const newCount = newSpeakers.length
    if (newCount > oldCount && oldCount > 0) playJoinSound()
    else if (newCount < oldCount && newCount > 0) playLeaveSound()
  }

  previousSpeakerCount = newSpeakers.length
  speakers = newSpeakers
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
const VOICE_STATUS_POLL_MS = 600
const SERVER_PARTICIPANTS_POLL_MS = 800
const P2P_PARTICIPANTS_POLL_MS = 1500

function startVoicePolling() {
  if (voicePolling) return
  voicePolling = setInterval(() => refreshVoiceStatus(), VOICE_STATUS_POLL_MS)
}

function stopVoicePolling() {
  if (voicePolling) {
    clearInterval(voicePolling)
    voicePolling = null
  }
}

// --- Translated Audio Playback ---

interface TranslatedAudioEvent {
  peerID: string
  audio: string  // base64
  format: string // "mp3"
}

let translatedAudioContext: AudioContext | null = null

function setupTranslatedAudioListener() {
  EventsOn('voice:translated-audio', (data: TranslatedAudioEvent) => {
    playTranslatedAudio(data)
  })
}

async function playTranslatedAudio(data: TranslatedAudioEvent) {
  try {
    if (!translatedAudioContext) {
      translatedAudioContext = new AudioContext()
    }
    const binaryStr = atob(data.audio)
    const bytes = new Uint8Array(binaryStr.length)
    for (let i = 0; i < binaryStr.length; i++) {
      bytes[i] = binaryStr.charCodeAt(i)
    }
    const audioBuffer = await translatedAudioContext.decodeAudioData(bytes.buffer)
    const source = translatedAudioContext.createBufferSource()
    source.buffer = audioBuffer
    source.connect(translatedAudioContext.destination)
    source.start()
  } catch (e) {
    console.error('Failed to play translated audio:', e)
  }
}

function teardownTranslatedAudioListener() {
  EventsOff('voice:translated-audio')
  if (translatedAudioContext) {
    translatedAudioContext.close().catch(() => {})
    translatedAudioContext = null
  }
}

export async function joinVoice(serverID: string, channelID: string, userID: string, username: string, avatarURL: string): Promise<void> {
  if (state !== 'disconnected') return

  state = 'connecting'
  error = null
  channelStartedAt = null

  try {
    const mode = isServerMode() ? 'server' : 'p2p'
    console.info(`[voice] joinVoice called: mode=${mode}, serverID=${serverID}, channelID=${channelID}, userID=${userID}, username=${username}`)

    if (isServerMode()) {
      // Server mode: voice runs entirely in the browser via WebRTC.
      // No Go backend involvement — connects directly to central signaling server.
      console.info('[voice] Step 1: ensuring valid token...')
      await ensureValidToken()
      console.info('[voice] Step 1: token OK')

      if (typeof RTCPeerConnection === 'undefined') {
        throw new Error('WebRTC is not supported in this browser. Voice requires RTCPeerConnection.')
      }
      console.info('[voice] Step 2: RTCPeerConnection available')

      // Pre-check microphone access before proceeding — fail fast with clear error.
      console.info('[voice] Step 2.5: checking microphone access...')
      try {
        const testStream = await navigator.mediaDevices.getUserMedia({ audio: true })
        testStream.getTracks().forEach(t => t.stop()) // release immediately
        console.info('[voice] Step 2.5: microphone access granted')
      } catch (micErr) {
        const detail = micErr instanceof DOMException ? `${micErr.name}: ${micErr.message}` : String(micErr)
        console.error('[voice] Microphone access FAILED:', detail)
        throw new Error(`Microphone access denied: ${detail}. Please allow microphone access and try again.`)
      }

      const settings = getSettings()
      const baseURL = apiClient.getBaseURL()
      if (!baseURL) {
        throw new Error('Server URL not configured. Cannot connect to voice signaling.')
      }
      console.info('[voice] Step 3: baseURL =', baseURL)

      rtcClient = new VoiceRTCClient(
        (status: VoiceRTCStatus) => {
          applyVoiceStatus(status as unknown as VoiceStatusData)
        },
        (message: string) => {
          console.error('[voice] RTC error callback:', message)
          error = message
        },
      )

      console.info('[voice] Step 4: calling rtcClient.join...')
      await rtcClient.join({
        baseURL,
        serverID,
        channelID,
        userID,
        username,
        avatarURL,
        inputDeviceId: settings.audioInputDevice,
        outputDeviceId: settings.audioOutputDevice,
        authToken: apiClient.getTokens()?.accessToken || '',
      })
      console.info('[voice] Step 5: rtcClient.join completed successfully')
    } else {
      // P2P mode: voice runs through Go backend (local signaling server + Pion WebRTC).
      // Completely independent path — no shared logic with server mode.
      console.info('[voice] P2P mode: connecting via Go voice engine')
      await App.JoinVoice(serverID, channelID, userID, username, avatarURL)
      console.info('[voice] P2P mode: JoinVoice completed')
    }

    await refreshVoiceStatus()
    playJoinSound()
    // Timer: prefer server-provided channel_started_at for synchronized display.
    // applyVoiceStatus may have already started it if peer_list arrived fast.
    if (!timerInterval) startTimer(channelStartedAt ?? Date.now())
    startVoicePolling()
    startVoiceActivityDetection()
    setupTranslatedAudioListener()
    console.info('[voice] Join complete — state:', state)
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'Failed to join voice channel'
    console.error('[voice] Join FAILED at:', msg, e)
    error = msg
    state = 'disconnected'
    // Clean up partial state
    if (rtcClient) {
      void rtcClient.leave().catch(() => {})
      rtcClient = null
    }
  }
}

export async function leaveVoice(): Promise<void> {
  if (state === 'disconnected') return

  try {
    if (rtcClient) {
      await rtcClient.leave()
      rtcClient = null
    } else {
      await App.LeaveVoice()
    }
  } catch (e) {
    console.error('Failed to leave voice channel:', e)
  } finally {
    stopVoicePolling()
    stopVoiceActivityDetection()
    stopScreenShare()
    teardownTranslatedAudioListener()
    stopTimer()
    state = 'disconnected'
    channelId = null
    channelStartedAt = null
    speakers = []
    muted = false
    deafened = false
    previousSpeakerCount = 0
  }
}

export async function toggleMute(): Promise<void> {
  try {
    const isMuted: boolean = rtcClient ? rtcClient.toggleMute() : await App.ToggleMute()
    muted = isMuted
  } catch (e) {
    console.error('Failed to toggle mute:', e)
  }
}

export async function toggleDeafen(): Promise<void> {
  try {
    const isDeafened: boolean = rtcClient ? rtcClient.toggleDeafen() : await App.ToggleDeafen()
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
    if (rtcClient) {
      applyVoiceStatus(rtcClient.getStatus() as unknown as VoiceStatusData)
      return
    }

    const status = await App.GetVoiceStatus()
    applyVoiceStatus(status as unknown as VoiceStatusData)
  } catch (e) {
    console.error('Failed to refresh voice status:', e)
  }
}

export async function setVoiceInputDevice(deviceId: string): Promise<void> {
  if (!rtcClient) return
  try {
    await rtcClient.setInputDevice(deviceId)
  } catch (e) {
    console.error('Failed to switch input device:', e)
  }
}

export async function setVoiceOutputDevice(deviceId: string): Promise<void> {
  if (!rtcClient) return
  try {
    await rtcClient.setOutputDevice(deviceId)
  } catch (e) {
    console.error('Failed to switch output device:', e)
  }
}

// Track the local user's username for VAD matching
let localUsername: string | null = null
let localUserID: string | null = null

export function setLocalUsername(username: string, userID?: string): void {
  localUsername = username
  localUserID = userID?.trim() || null
}

function isLocalUser(speaker: SpeakerData): boolean {
  if (localUserID && speaker.user_id === localUserID) return true
  if (!localUsername) return false
  return speaker.username === localUsername
}

// Channel participants cache: maps channelID -> participants list
// This shows who's in voice channels even when the local user is NOT connected.
let channelParticipants = $state<Record<string, SpeakerData[]>>({})
let participantsPolling: ReturnType<typeof setInterval> | null = null

export function getChannelParticipants(channelId: string): SpeakerData[] {
  return channelParticipants[channelId] ?? []
}

export async function refreshChannelParticipants(serverID: string, channelIDs: string[]): Promise<void> {
  try {
    const updated: Record<string, SpeakerData[]> = {}
    const inServerMode = isServerMode()

    if (inServerMode) {
      const byChannel = await apiClient.get<Record<string, any[]>>(`/api/v1/servers/${serverID}/voice/participants`)
      for (const chID of channelIDs) {
        const peers = byChannel?.[chID] ?? []
        if (peers.length === 0) continue
        updated[chID] = sortSpeakersStable(
          peers.map((p: any) => normalizeSpeaker({
            peer_id: p.peer_id || '',
            user_id: p.user_id || '',
            username: p.username || '',
            avatar_url: p.avatar_url || '',
            volume: 0,
            speaking: false,
            muted: !!p.muted,
            deafened: !!p.deafened,
          })),
        )
      }
    } else {
      const results = await Promise.all(
        channelIDs.map(async (chID) => {
          try {
            const peers = await App.GetVoiceParticipants(serverID, chID)
            if (!peers || peers.length === 0) return [chID, [] as SpeakerData[]] as const

            const mapped = sortSpeakersStable(
              peers.map((p: any) => normalizeSpeaker({
                peer_id: p.peer_id || '',
                user_id: p.user_id || '',
                username: p.username || '',
                avatar_url: p.avatar_url || '',
                volume: 0,
                speaking: false,
                muted: !!p.muted,
                deafened: !!p.deafened,
              })),
            )

            return [chID, mapped as SpeakerData[]] as const
          } catch {
            return [chID, [] as SpeakerData[]] as const
          }
        }),
      )

      for (const [chID, peers] of results) {
        if (peers.length > 0) updated[chID] = peers
      }
    }

    channelParticipants = updated
  } catch (e) {
    console.error('Failed to refresh channel participants:', e)
  }
}

export function startParticipantsPolling(serverID: string, channelIDs: string[]): void {
  stopParticipantsPolling()
  if (channelIDs.length === 0) return
  // Initial fetch
  refreshChannelParticipants(serverID, channelIDs)
  const intervalMs = isServerMode() ? SERVER_PARTICIPANTS_POLL_MS : P2P_PARTICIPANTS_POLL_MS
  participantsPolling = setInterval(() => {
    refreshChannelParticipants(serverID, channelIDs)
  }, intervalMs)
}

export function stopParticipantsPolling(): void {
  if (participantsPolling) {
    clearInterval(participantsPolling)
    participantsPolling = null
  }
  channelParticipants = {}
}

export function resetVoice(): void {
  if (rtcClient) {
    void rtcClient.leave()
    rtcClient = null
  }
  stopTimer()
  state = 'disconnected'
  channelId = null
  channelStartedAt = null
  muted = false
  deafened = false
  speakers = []
  error = null
  previousSpeakerCount = 0
  stopParticipantsPolling()
}
