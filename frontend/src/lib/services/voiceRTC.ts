export interface VoiceParticipant {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  volume: number
  speaking: boolean
  muted: boolean
  deafened: boolean
  screen_sharing?: boolean
  dominant_speaker?: boolean
  connection_quality?: VoiceConnectionQuality
  quality_score?: number
}

export type VoiceConnectionQuality = 'good' | 'fair' | 'poor' | 'unknown'

export interface VoiceScreenShare {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  stream: MediaStream
  local: boolean
}

export type VoiceDiagnosticLevel = 'info' | 'warn' | 'error'

export interface VoiceDiagnosticEvent {
  ts: number
  level: VoiceDiagnosticLevel
  code: string
  message: string
}

export interface VoicePeerConnectionStats {
  peer_id: string
  connection_state: RTCPeerConnectionState
  ice_connection_state: RTCIceConnectionState
  quality: VoiceConnectionQuality
  quality_score: number
  loss_ratio: number
  packets_received: number
  packets_lost: number
  audio_jitter_ms: number
  round_trip_time_ms: number
  available_outgoing_bitrate: number
}

export interface VoiceDiagnosticsSnapshot {
  ts: number
  ws_state: number
  reconnect_attempts: number
  muted: boolean
  deafened: boolean
  noise_suppression: boolean
  screen_sharing: boolean
  peers: VoicePeerConnectionStats[]
  events: VoiceDiagnosticEvent[]
}

export interface VoiceRTCStatus {
  state: 'disconnected' | 'connecting' | 'connected'
  channel_id: string
  muted: boolean
  deafened: boolean
  peer_count: number
  speakers: VoiceParticipant[]
  screen_shares?: VoiceScreenShare[]
  noise_suppression?: boolean
  channel_started_at?: number
  diagnostics?: VoiceDiagnosticsSnapshot
}

interface VoicePeerMeta {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  muted?: boolean
  deafened?: boolean
  screen_sharing?: boolean
}

interface VoiceJoinOptions {
  baseURL: string
  serverID: string
  channelID: string
  userID: string
  username: string
  avatarURL: string
  inputDeviceId?: string
  outputDeviceId?: string
  authToken?: string
  noiseSuppression?: boolean
}

interface SignalMessage {
  type: string
  from?: string
  to?: string
  server_id?: string
  channel_id?: string
  payload?: unknown
}

// Per-peer state including the RTCPeerConnection and role-based negotiation flags.
// Jitsi-style: roles (initiator/responder) are fixed per peer pair to prevent glare.
// The joiner (who receives peer_list) is ALWAYS the initiator for existing peers.
// Existing peers (who receive peer_joined) are ALWAYS responders — they never create offers.
interface PeerConnectionState {
  pc: RTCPeerConnection
  audio: HTMLAudioElement
  stream: MediaStream
  analyser: AnalyserNode | null
  analyserData: Uint8Array<ArrayBuffer> | null
  sourceNode: MediaStreamAudioSourceNode | null
  screenVideoSender: RTCRtpSender | null
  screenAudioSender: RTCRtpSender | null
  pendingRemoteCandidates: RTCIceCandidateInit[]
  // Role-based negotiation (Jitsi pattern — no Perfect Negotiation)
  role: 'initiator' | 'responder'
  makingOffer: boolean
}

interface IceConfigServer {
  urls?: unknown
  username?: unknown
  credential?: unknown
}

interface IceConfigResponse {
  servers?: unknown
}

type ScreenShareQoSProfile = 'high' | 'balanced' | 'low'

const SCREEN_SHARE_QOS_CONSTRAINTS: Record<ScreenShareQoSProfile, MediaTrackConstraints> = {
  high: {
    frameRate: { ideal: 24, max: 30 },
    width: { max: 1920 },
    height: { max: 1080 },
  },
  balanced: {
    frameRate: { ideal: 15, max: 18 },
    width: { max: 1600 },
    height: { max: 900 },
  },
  low: {
    frameRate: { ideal: 10, max: 12 },
    width: { max: 1280 },
    height: { max: 720 },
  },
}

// Default STUN servers for ICE candidate gathering.
// TURN relay servers are provided by the backend via /api/v1/voice/ice-config
// when CONCORD_TURN_ENABLED=true (requires coturn or similar).
const DEFAULT_ICE_SERVERS: RTCIceServer[] = [
  { urls: ['stun:stun.l.google.com:19302'] },
  { urls: ['stun:stun1.l.google.com:19302'] },
  { urls: ['stun:stun2.l.google.com:19302'] },
  { urls: ['stun:stun3.l.google.com:19302'] },
]

export class VoiceRTCClient {
  private ws: WebSocket | null = null
  private localStream: MediaStream | null = null
  private screenStream: MediaStream | null = null
  private screenVideoTrack: MediaStreamTrack | null = null
  private screenAudioTrack: MediaStreamTrack | null = null
  private remoteScreenStreams = new Map<string, MediaStream>()
  private peers = new Map<string, PeerConnectionState>()
  private peersMeta = new Map<string, VoicePeerMeta>()
  private participants = new Map<string, VoiceParticipant>()
  private selfPeerID = ''
  private serverID = ''
  private channelID = ''
  private muted = false
  private deafened = false
  private channelStartedAt: number | null = null
  private state: VoiceRTCStatus['state'] = 'disconnected'
  private outputDeviceId = ''
  private noiseSuppressionEnabled = true
  private audioContext: AudioContext | null = null
  private speakingLoop: ReturnType<typeof setInterval> | null = null
  private statsLoop: ReturnType<typeof setInterval> | null = null
  private disconnectTimers = new Map<string, ReturnType<typeof setTimeout>>()
  private peerStats = new Map<string, VoicePeerConnectionStats>()
  private diagnosticsEvents: VoiceDiagnosticEvent[] = []
  private iceServers: RTCIceServer[] = DEFAULT_ICE_SERVERS
  private iceRestartTimers = new Map<string, ReturnType<typeof setTimeout>>()
  private iceRestartAttempts = new Map<string, number>()
  private iceRestartLastAttempt = new Map<string, number>()
  private screenQoSProfile: ScreenShareQoSProfile = 'high'
  private lastScreenQoSAppliedAt = 0

  private readonly maxIceRestartAttempts = 3
  private readonly iceRestartCooldownMs = 6_000
  private readonly disconnectedRestartDelayMs = 1_200
  private readonly peerDisconnectGraceMs = 8_000
  private readonly screenQoSCooldownMs = 10_000

  // WebSocket reconnect state
  private joinOpts: VoiceJoinOptions | null = null
  private reconnectAttempts = 0
  private readonly maxReconnectAttempts = 5
  private readonly reconnectBaseDelay = 1000 // 1s, exponential backoff
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private intentionalDisconnect = false
  private keepaliveInterval: ReturnType<typeof setInterval> | null = null
  private signalQueue: Promise<void> = Promise.resolve()

  constructor(
    private readonly onStatusChange: (status: VoiceRTCStatus) => void,
    private readonly onError: (message: string) => void,
    private readonly onDiagnosticsChange?: (snapshot: VoiceDiagnosticsSnapshot) => void,
  ) {}

  async join(opts: VoiceJoinOptions): Promise<void> {
    if (this.state !== 'disconnected') return

    this.joinOpts = opts
    this.intentionalDisconnect = false
    this.reconnectAttempts = 0
    this.state = 'connecting'
    this.serverID = opts.serverID
    this.channelID = opts.channelID
    this.selfPeerID = crypto.randomUUID()
    this.outputDeviceId = opts.outputDeviceId ?? ''
    this.noiseSuppressionEnabled = opts.noiseSuppression ?? true
    this.channelStartedAt = null
    this.peerStats.clear()
    this.peersMeta.clear()
    this.participants.clear()
    this.remoteScreenStreams.clear()
    this.signalQueue = Promise.resolve()
    this.diagnosticsEvents = []
    this.clearAllIceRestartState()
    this.screenQoSProfile = 'high'
    this.lastScreenQoSAppliedAt = 0
    this.emitStatus()
    this.pushDiag('info', 'join:start', `joining ${opts.serverID}/${opts.channelID}`)

    const localUsername = this.safeUsername(opts.username, opts.userID, this.selfPeerID)
    this.participants.set(this.selfPeerID, {
      peer_id: this.selfPeerID,
      user_id: opts.userID,
      username: localUsername,
      avatar_url: opts.avatarURL,
      volume: 0,
      speaking: false,
      muted: this.muted,
      deafened: this.deafened,
      screen_sharing: false,
      dominant_speaker: false,
      connection_quality: 'unknown',
      quality_score: 0,
    })

    try {
      this.localStream = await this.createLocalStream(opts.inputDeviceId)
    } catch (e) {
      await this.leave()
      const detail = e instanceof DOMException ? e.message : String(e)
      throw new Error(`Microphone access denied or unavailable: ${detail}`)
    }

    this.applyMuteState()
    this.initAudioAnalysis()

    try {
      this.iceServers = await this.resolveIceServers(opts.baseURL, opts.authToken)
    } catch {
      this.pushDiag('warn', 'ice:config', 'failed to fetch ICE config, using defaults')
      this.iceServers = DEFAULT_ICE_SERVERS
    }
    this.pushDiag('info', 'ice:config', `resolved ${this.iceServers.length} ICE server entries`)

    try {
      await this.connectWebSocket(opts, localUsername)
      this.state = 'connected'
      this.startStatsLoop()
      this.emitStatus()
    } catch (e) {
      await this.leave()
      const detail = e instanceof Error ? e.message : String(e)
      this.pushDiag('error', 'join:failed', detail)
      throw new Error(`Failed to connect to voice signaling server: ${detail}`)
    }
  }

  private async connectWebSocket(opts: VoiceJoinOptions, localUsername: string): Promise<void> {
    const wsURL = this.toSignalingURL(opts.baseURL)
    this.pushDiag('info', 'ws:connect', wsURL)
    this.ws = await this.openWebSocket(wsURL)
    this.signalQueue = Promise.resolve()
    this.pushDiag('info', 'ws:open', 'signaling connected')

    this.ws.onmessage = (event) => {
      this.enqueueSignal(event.data)
    }

    this.ws.onclose = () => {
      this.stopKeepalive()
      if (this.intentionalDisconnect || this.state === 'disconnected') {
        if (this.state !== 'disconnected') {
          this.state = 'disconnected'
          this.cleanupPeers()
          this.cleanupLocalStream()
          this.stopScreenShare()
          this.cleanupAudioAnalysis()
          this.stopStatsLoop()
          this.participants.clear()
          this.peersMeta.clear()
          this.peerStats.clear()
          this.remoteScreenStreams.clear()
          this.channelStartedAt = null
          this.emitStatus()
        }
        return
      }

      this.pushDiag('warn', 'ws:closed', 'signaling closed unexpectedly, reconnecting')
      this.cleanupPeers()
      this.ws = null
      void this.scheduleReconnect()
    }

    this.sendSignal({
      type: 'join',
      server_id: this.serverID,
      channel_id: this.channelID,
      payload: {
        user_id: opts.userID,
        peer_id: this.selfPeerID,
        username: localUsername,
        avatar_url: opts.avatarURL,
        addresses: [],
        public_key: [],
        muted: this.muted,
        deafened: this.deafened,
        screen_sharing: !!this.screenVideoTrack,
      },
    })
    this.pushDiag('info', 'signal:join', `self peer ${this.selfPeerID}`)

    this.startKeepalive()
  }

  private async scheduleReconnect(): Promise<void> {
    if (this.intentionalDisconnect || this.state !== 'connected') return
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.pushDiag('error', 'ws:reconnect', `max attempts (${this.maxReconnectAttempts}) reached`)
      this.state = 'disconnected'
      this.cleanupPeers()
      this.cleanupLocalStream()
      this.stopScreenShare()
      this.cleanupAudioAnalysis()
      this.stopStatsLoop()
      this.participants.clear()
      this.peersMeta.clear()
      this.peerStats.clear()
      this.remoteScreenStreams.clear()
      this.channelStartedAt = null
      this.emitStatus()
      this.onError('voice connection lost')
      return
    }

    this.reconnectAttempts++
    const delay = Math.min(this.reconnectBaseDelay * Math.pow(2, this.reconnectAttempts - 1), 10_000)
    this.pushDiag('info', 'ws:reconnect', `retry in ${delay}ms (${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

    await new Promise<void>((resolve) => {
      this.reconnectTimer = setTimeout(resolve, delay)
    })

    if (this.intentionalDisconnect || this.state !== 'connected') return

    const opts = this.joinOpts
    if (!opts) return

    try {
      const localUsername = this.safeUsername(opts.username, opts.userID, this.selfPeerID)
      if (!this.participants.has(this.selfPeerID)) {
        this.participants.set(this.selfPeerID, {
          peer_id: this.selfPeerID,
          user_id: opts.userID,
          username: localUsername,
          avatar_url: opts.avatarURL,
          volume: 0,
          speaking: false,
          muted: this.muted,
          deafened: this.deafened,
          screen_sharing: !!this.screenVideoTrack,
          dominant_speaker: false,
          connection_quality: 'unknown',
          quality_score: 0,
        })
      }

      await this.connectWebSocket(opts, localUsername)
      this.reconnectAttempts = 0
      this.pushDiag('info', 'ws:reconnect', 'reconnected')
      this.emitStatus()
    } catch (e) {
      this.pushDiag('warn', 'ws:reconnect', e instanceof Error ? e.message : String(e))
      void this.scheduleReconnect()
    }
  }

  async leave(): Promise<void> {
    if (this.state === 'disconnected') return

    this.intentionalDisconnect = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    try {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.sendSignal({
          type: 'leave',
          from: this.selfPeerID,
          server_id: this.serverID,
          channel_id: this.channelID,
        })
      }
    } catch {
      // ignore leave send failures
    }

    this.state = 'disconnected'
    this.stopKeepalive()
    this.stopStatsLoop()
    this.cleanupPeers()
    this.cleanupLocalStream()
    this.stopScreenShare()
    this.remoteScreenStreams.clear()
    this.cleanupAudioAnalysis()
    this.peerStats.clear()
    this.participants.clear()
    this.peersMeta.clear()
    this.channelStartedAt = null
    this.joinOpts = null

    if (this.ws) {
      this.ws.onmessage = null
      this.ws.onclose = null
      this.ws.close()
      this.ws = null
    }

    this.serverID = ''
    this.channelID = ''
    this.selfPeerID = ''
    this.emitStatus()
  }

  getStatus(): VoiceRTCStatus {
    const speakers = Array.from(this.participants.values()).sort((a, b) => {
      const aLocal = a.peer_id === this.selfPeerID ? 0 : 1
      const bLocal = b.peer_id === this.selfPeerID ? 0 : 1
      if (aLocal !== bLocal) return aLocal - bLocal
      const aDominant = a.dominant_speaker ? 0 : 1
      const bDominant = b.dominant_speaker ? 0 : 1
      if (aDominant !== bDominant) return aDominant - bDominant
      const byUser = a.username.localeCompare(b.username)
      if (byUser !== 0) return byUser
      return a.peer_id.localeCompare(b.peer_id)
    })
    const screenShares = this.buildScreenShares()

    return {
      state: this.state,
      channel_id: this.channelID,
      muted: this.muted,
      deafened: this.deafened,
      peer_count: Math.max(0, this.participants.size - 1),
      speakers,
      screen_shares: screenShares,
      noise_suppression: this.noiseSuppressionEnabled,
      channel_started_at: this.channelStartedAt ?? undefined,
      diagnostics: this.getDiagnostics(),
    }
  }

  getDiagnostics(): VoiceDiagnosticsSnapshot {
    return {
      ts: Date.now(),
      ws_state: this.ws?.readyState ?? WebSocket.CLOSED,
      reconnect_attempts: this.reconnectAttempts,
      muted: this.muted,
      deafened: this.deafened,
      noise_suppression: this.noiseSuppressionEnabled,
      screen_sharing: !!this.screenVideoTrack,
      peers: Array.from(this.peerStats.values()),
      events: [...this.diagnosticsEvents],
    }
  }

  toggleMute(): boolean {
    this.muted = !this.muted
    this.applyMuteState()
    this.updateSelfParticipantState()
    this.sendPeerState()
    this.emitStatus()
    return this.muted
  }

  toggleDeafen(): boolean {
    this.deafened = !this.deafened
    if (this.deafened) {
      this.muted = true
    }
    this.applyMuteState()
    this.applyDeafenState()
    this.updateSelfParticipantState()
    this.sendPeerState()
    this.emitStatus()
    return this.deafened
  }

  async setInputDevice(deviceId: string): Promise<void> {
    if (this.state !== 'connected') return
    if (this.joinOpts) this.joinOpts.inputDeviceId = deviceId

    const nextStream = await this.createLocalStream(deviceId)
    await this.replaceLocalAudioTrack(nextStream)
  }

  async setOutputDevice(deviceId: string): Promise<void> {
    this.outputDeviceId = deviceId
    if (this.joinOpts) this.joinOpts.outputDeviceId = deviceId
    const applyOps: Promise<void>[] = []
    for (const peer of this.peers.values()) {
      applyOps.push(this.applyOutputDevice(peer.audio))
    }
    await Promise.allSettled(applyOps)
  }

  async setNoiseSuppression(enabled: boolean): Promise<boolean> {
    this.noiseSuppressionEnabled = enabled
    if (this.joinOpts) this.joinOpts.noiseSuppression = enabled
    this.pushDiag('info', 'audio:noise', enabled ? 'noise suppression enabled' : 'noise suppression disabled')

    if (this.state !== 'connected') {
      this.emitStatus()
      return this.noiseSuppressionEnabled
    }

    const nextStream = await this.createLocalStream(this.joinOpts?.inputDeviceId)
    await this.replaceLocalAudioTrack(nextStream)
    this.emitStatus()
    return this.noiseSuppressionEnabled
  }

  async startScreenShare(): Promise<boolean> {
    if (this.state !== 'connected') return false
    if (this.screenVideoTrack) return true

    let stream: MediaStream
    try {
      stream = await navigator.mediaDevices.getDisplayMedia({
        video: {
          frameRate: { ideal: 15, max: 30 },
          width: { max: 1920 },
          height: { max: 1080 },
        },
        audio: true,
      })
    } catch {
      this.pushDiag('warn', 'screen:start', 'screen picker canceled or denied')
      return false
    }

    const videoTrack = stream.getVideoTracks()[0]
    if (!videoTrack) {
      stream.getTracks().forEach(t => t.stop())
      this.pushDiag('warn', 'screen:start', 'display stream has no video track')
      return false
    }

    this.screenStream = stream
    this.screenVideoTrack = videoTrack
    this.screenAudioTrack = stream.getAudioTracks()[0] ?? null
    this.screenQoSProfile = 'high'
    this.lastScreenQoSAppliedAt = 0
    this.applyMuteState()

    videoTrack.addEventListener('ended', () => {
      this.stopScreenShare()
    })

    for (const peer of this.peers.values()) {
      peer.screenVideoSender = peer.pc.addTrack(videoTrack, stream)
      if (this.screenAudioTrack) {
        peer.screenAudioSender = peer.pc.addTrack(this.screenAudioTrack, stream)
      }
    }

    await this.applyScreenShareQoS('high', true)

    this.updateSelfParticipantState()
    this.sendPeerState()
    this.pushDiag('info', 'screen:start', 'screen sharing started')
    this.emitStatus()
    return true
  }

  stopScreenShare(): void {
    if (!this.screenStream && !this.screenVideoTrack && !this.screenAudioTrack) return

    for (const peer of this.peers.values()) {
      if (peer.screenVideoSender) {
        try {
          peer.pc.removeTrack(peer.screenVideoSender)
        } catch {
          // ignore
        }
        peer.screenVideoSender = null
      }
      if (peer.screenAudioSender) {
        try {
          peer.pc.removeTrack(peer.screenAudioSender)
        } catch {
          // ignore
        }
        peer.screenAudioSender = null
      }
    }

    if (this.screenStream) {
      this.screenStream.getTracks().forEach(t => t.stop())
      this.screenStream = null
    }
    if (this.screenVideoTrack) this.screenVideoTrack = null
    if (this.screenAudioTrack) this.screenAudioTrack = null
    this.screenQoSProfile = 'high'
    this.lastScreenQoSAppliedAt = 0

    this.updateSelfParticipantState()
    if (this.state === 'connected') {
      this.sendPeerState()
    }
    this.pushDiag('info', 'screen:stop', 'screen sharing stopped')
    this.emitStatus()
  }

  // ---------------------------------------------------------------------------
  // Signaling message handler
  // ---------------------------------------------------------------------------

  private async handleSignal(rawData: unknown): Promise<void> {
    if (typeof rawData !== 'string') return

    let signal: SignalMessage
    try {
      signal = JSON.parse(rawData) as SignalMessage
    } catch {
      this.pushDiag('warn', 'signal:parse', 'failed to parse signaling message')
      return
    }

    // Log every signal except keepalive pings
    if (signal.type !== 'ping') {
      this.pushDiag('info', `signal:${signal.type}`, `from=${signal.from ?? '-'} to=${signal.to ?? '-'}`)
    }

    try {
      switch (signal.type) {
        case 'peer_list':
          this.onPeerList(signal.payload)
          break
        case 'peer_joined':
          this.onPeerJoined(signal.payload)
          break
        case 'peer_left':
          if (signal.from) this.onPeerLeft(signal.from)
          break
        case 'peer_state':
          this.onPeerState(signal.from || '', signal.payload)
          break
        case 'sdp_offer':
          if (signal.from) await this.onDescription(signal.from, 'offer', signal.payload)
          break
        case 'sdp_answer':
          if (signal.from) await this.onDescription(signal.from, 'answer', signal.payload)
          break
        case 'ice_candidate':
          if (signal.from) await this.onICECandidate(signal.from, signal.payload)
          break
        case 'error': {
          const errPayload = signal.payload as { message?: string } | undefined
          this.pushDiag('error', 'signal:error', errPayload?.message ?? 'unknown')
          break
        }
        default:
          break
      }
    } catch (e) {
      // Log but do NOT propagate to onError — that would surface transient
      // signaling issues as user-visible errors and potentially break the flow.
      this.pushDiag('warn', `signal:${signal.type}:handler`, e instanceof Error ? e.message : String(e))
    }
  }

  private enqueueSignal(rawData: unknown): void {
    this.signalQueue = this.signalQueue
      .then(async () => {
        await this.handleSignal(rawData)
      })
      .catch((e) => {
        const msg = e instanceof Error ? e.message : String(e)
        this.pushDiag('warn', 'signal:queue', msg)
      })
  }

  // ---------------------------------------------------------------------------
  // Peer list / join / leave handlers
  // ---------------------------------------------------------------------------

  private onPeerList(payload: unknown): void {
    const peers = this.extractPeers(payload)
    const startedAt = this.extractChannelStartedAt(payload)
    if (startedAt) this.channelStartedAt = startedAt
    this.pushDiag('info', 'peer:list', `${peers.length} peers`)

    for (const peer of peers) {
      if (peer.peer_id === this.selfPeerID) continue
      this.registerPeer(peer)
      // Jitsi pattern: joiner is ALWAYS the initiator for existing peers.
      // ensurePeerConnection creates the PC, addTrack triggers onnegotiationneeded,
      // which creates and sends the offer because role === 'initiator'.
      this.ensurePeerConnection(peer.peer_id, 'initiator')
    }
    this.emitStatus()
  }

  private onPeerJoined(payload: unknown): void {
    const peer = this.extractJoinPeer(payload)
    if (!peer || peer.peer_id === this.selfPeerID) return
    this.pushDiag('info', 'peer:joined', `${peer.peer_id} (${peer.username})`)
    this.registerPeer(peer)
    // Jitsi pattern: existing peers are ALWAYS responders — they DO NOT create offers.
    // They create the PeerConnection and add tracks, but onnegotiationneeded is suppressed.
    // The joiner will send the offer via their onPeerList → ensurePeerConnection('initiator').
    this.ensurePeerConnection(peer.peer_id, 'responder')
    this.pushDiag('info', 'peer:joined:responder', `awaiting offer from ${peer.peer_id}`)
    this.emitStatus()
  }

  private registerPeer(peer: VoicePeerMeta): void {
    this.peersMeta.set(peer.peer_id, peer)
    this.participants.set(peer.peer_id, {
      peer_id: peer.peer_id,
      user_id: peer.user_id,
      username: peer.username,
      avatar_url: peer.avatar_url,
      volume: 0,
      speaking: false,
      muted: peer.deafened ? true : !!peer.muted,
      deafened: !!peer.deafened,
      screen_sharing: !!peer.screen_sharing,
      dominant_speaker: false,
      connection_quality: 'unknown',
      quality_score: 0,
    })
  }

  private onPeerLeft(peerID: string): void {
    this.pushDiag('info', 'peer:left', peerID)
    this.removePeerFromSession(peerID)
    this.emitStatus()
  }

  // ---------------------------------------------------------------------------
  // Role-Based Negotiation (Jitsi pattern — no Perfect Negotiation)
  // Glare is prevented architecturally: roles are fixed per peer pair.
  // Joiner = initiator (creates offers), existing peer = responder (only answers).
  // ---------------------------------------------------------------------------

  private async onDescription(fromPeerID: string, type: 'offer' | 'answer', payload: unknown): Promise<void> {
    const sdp = this.extractSDP(payload)
    if (!sdp) {
      console.warn(`[voice] Empty SDP in ${type} from ${fromPeerID}`)
      return
    }
    console.info(`[voice] ${type} from ${fromPeerID}, sdp=${sdp.length}B`)

    // Ensure we have metadata for this peer
    if (!this.participants.has(fromPeerID)) {
      const meta = this.peersMeta.get(fromPeerID) ?? {
        peer_id: fromPeerID,
        user_id: fromPeerID,
        username: this.safeUsername('', fromPeerID, fromPeerID),
        avatar_url: '',
      }
      this.registerPeer(meta)
    }

    // If we receive an offer, we are the responder for this peer.
    // If we receive an answer, we are the initiator for this peer.
    const role = type === 'offer' ? 'responder' : 'initiator'
    const peer = this.ensurePeerConnection(fromPeerID, role)
    const pc = peer.pc
    const description: RTCSessionDescriptionInit = { type, sdp }

    // Send debug breadcrumb via signaling so server logs show progress
    const dbg = (step: string) => {
      this.sendSignal({
        type: 'peer_state',
        server_id: this.serverID,
        channel_id: this.channelID,
        payload: { peer_id: this.selfPeerID, debug: `${type}:${step}`, muted: this.muted, deafened: this.deafened, screen_sharing: false },
      })
    }

    try {
      if (type === 'offer') {
        // Responder: accept the offer and send back an answer.
        // No collision possible — responders never send offers.
        const sigState = pc.signalingState
        console.info(`[voice] Accepting offer from ${fromPeerID}, signalingState=${sigState}, role=${peer.role}`)
        dbg(`accept:sigState=${sigState}:role=${peer.role}`)

        // If we're somehow in have-local-offer (shouldn't happen with role separation),
        // rollback gracefully before accepting the remote offer.
        if (sigState === 'have-local-offer') {
          console.warn(`[voice] Unexpected have-local-offer as responder for ${fromPeerID}, rolling back`)
          dbg('rollback:start')
          await pc.setLocalDescription({ type: 'rollback' })
          dbg('rollback:done')
        }

        dbg('setRemoteDesc:start')
        await pc.setRemoteDescription(description)
        dbg(`setRemoteDesc:done:sig=${pc.signalingState}`)
        await this.flushPendingICECandidates(fromPeerID, peer)
        console.info(`[voice] Remote offer set for ${fromPeerID}, signalingState=${pc.signalingState}`)

        // Create and send answer
        dbg('createAnswer:start')
        const answer = await pc.createAnswer()
        dbg(`createAnswer:done:type=${answer.type}`)
        await pc.setLocalDescription(answer)
        const localDesc = pc.localDescription
        dbg(`setLocalDesc:done:sdp=${(localDesc?.sdp ?? '').length}B`)
        console.info(`[voice] Answer created for ${fromPeerID}, type=${localDesc?.type}, sdp=${(localDesc?.sdp ?? '').length}B`)

        if (localDesc && localDesc.sdp) {
          console.info(`[voice] >> sdp_answer to ${fromPeerID}`)
          this.pushDiag('info', 'sdp:answer', `${fromPeerID} ${localDesc.sdp.length}B`)
          dbg('sendAnswer:start')
          this.sendSignal({
            type: 'sdp_answer',
            from: this.selfPeerID,
            to: fromPeerID,
            server_id: this.serverID,
            channel_id: this.channelID,
            payload: { sdp: localDesc.sdp },
          })
          dbg('sendAnswer:done')
        } else {
          dbg('sendAnswer:SKIP:noSdp')
        }
      } else {
        // Initiator: accept the answer from the responder.
        const sigState = pc.signalingState
        console.info(`[voice] Setting remote answer from ${fromPeerID}, signalingState=${sigState}`)
        dbg(`answer:sigState=${sigState}`)

        if (sigState !== 'have-local-offer') {
          console.warn(`[voice] Received answer but signalingState=${sigState} (expected have-local-offer), dropping`)
          dbg(`answer:DROP:sigState=${sigState}`)
          return
        }

        dbg('setRemoteAnswer:start')
        await pc.setRemoteDescription(description)
        await this.flushPendingICECandidates(fromPeerID, peer)
        dbg(`setRemoteAnswer:done:conn=${pc.connectionState}`)
        console.info(`[voice] Remote answer set for ${fromPeerID}, connectionState=${pc.connectionState}`)
      }
    } catch (e) {
      const errMsg = e instanceof Error ? e.message : String(e)
      this.pushDiag('warn', 'sdp:handle:error', `${type} ${fromPeerID} ${errMsg}`)
      console.error(`[voice] FAILED to handle ${type} from ${fromPeerID}: ${errMsg}`, e)
      dbg(`ERROR:${errMsg.substring(0, 100)}`)
    }
  }

  private async onICECandidate(fromPeerID: string, payload: unknown): Promise<void> {
    const candidate = this.extractICE(payload)
    if (!candidate) return
    const peer = this.peers.get(fromPeerID)
    if (!peer) return

    if (!peer.pc.remoteDescription) {
      // Buffer candidates until remote description is set
      peer.pendingRemoteCandidates.push(candidate)
      return
    }

    try {
      await peer.pc.addIceCandidate(candidate)
    } catch (e) {
      console.warn(`[voice] Failed to add ICE candidate from ${fromPeerID}:`, e)
    }
  }

  // ---------------------------------------------------------------------------
  // PeerConnection lifecycle
  // ---------------------------------------------------------------------------

  private ensurePeerConnection(peerID: string, role: 'initiator' | 'responder'): PeerConnectionState {
    const existing = this.peers.get(peerID)
    if (existing) return existing

    console.info(`[voice] Creating PeerConnection for ${peerID} as ${role}`)
    const pc = new RTCPeerConnection({ iceServers: this.iceServers })
    const stream = new MediaStream()
    const audio = new Audio()
    audio.autoplay = true
    audio.setAttribute('playsinline', 'true')
    audio.volume = 1.0
    audio.srcObject = stream
    audio.muted = this.deafened

    void this.applyOutputDevice(audio)

    const state: PeerConnectionState = {
      pc,
      audio,
      stream,
      analyser: null,
      analyserData: null,
      sourceNode: null,
      screenVideoSender: null,
      screenAudioSender: null,
      pendingRemoteCandidates: [],
      role,
      makingOffer: false,
    }
    this.peers.set(peerID, state)

    // --- Role-based onnegotiationneeded ---
    // Only initiators create and send offers. Responders suppress onnegotiationneeded
    // because they will receive an offer from the initiator and respond with an answer.
    pc.onnegotiationneeded = async () => {
      if (state.role !== 'initiator') {
        console.info(`[voice] negotiationneeded suppressed for ${peerID} (role=${state.role})`)
        return
      }
      if (state.makingOffer) {
        console.info(`[voice] negotiationneeded skipped for ${peerID} (already making offer)`)
        return
      }

      try {
        if (pc.signalingState !== 'stable') {
          this.pushDiag('info', 'sdp:offer:skip', `${peerID} signalingState=${pc.signalingState}`)
          return
        }

        console.info(`[voice] negotiationneeded for ${peerID} (initiator)`)
        state.makingOffer = true
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer)
        const localDesc = pc.localDescription
        if (localDesc && localDesc.sdp) {
          console.info(`[voice] >> sdp_offer to ${peerID}, sdp=${localDesc.sdp.length}B`)
          this.sendSignal({
            type: 'sdp_offer',
            from: this.selfPeerID,
            to: peerID,
            server_id: this.serverID,
            channel_id: this.channelID,
            payload: { sdp: localDesc.sdp },
          })
        }
      } catch (e) {
        console.error(`[voice] negotiationneeded error for ${peerID}:`, e)
      } finally {
        state.makingOffer = false
      }
    }

    // Add local tracks AFTER setting up onnegotiationneeded so it fires correctly.
    // For initiators, this triggers offer creation. For responders, it's suppressed.
    if (this.localStream) {
      for (const track of this.localStream.getAudioTracks()) {
        pc.addTrack(track, this.localStream)
      }
    }

    // If local screen share is active, publish it to late-joining peers.
    if (this.screenStream && this.screenVideoTrack) {
      state.screenVideoSender = pc.addTrack(this.screenVideoTrack, this.screenStream)
      if (this.screenAudioTrack) {
        state.screenAudioSender = pc.addTrack(this.screenAudioTrack, this.screenStream)
      }
    }

    // --- ICE candidates ---
    pc.onicecandidate = (event) => {
      if (!event.candidate) {
        console.info(`[voice] ICE gathering complete for ${peerID}`)
        return
      }
      const candStr = event.candidate.candidate
      const typeMatch = candStr.match(/typ\s+(\w+)/)
      console.info(`[voice] ICE candidate for ${peerID}: type=${typeMatch?.[1] ?? '?'} ${candStr.substring(0, 80)}`)
      this.sendSignal({
        type: 'ice_candidate',
        from: this.selfPeerID,
        to: peerID,
        server_id: this.serverID,
        channel_id: this.channelID,
        payload: {
          candidate: event.candidate.candidate,
          sdp_mid: event.candidate.sdpMid ?? '',
          sdp_mline_index: event.candidate.sdpMLineIndex ?? 0,
        },
      })
    }

    // --- Remote tracks ---
    pc.ontrack = (event) => {
      this.pushDiag('info', 'track:remote', `${peerID} ${event.track.kind}:${event.track.id}`)

      if (event.track.kind === 'video') {
        const current = this.remoteScreenStreams.get(peerID) ?? new MediaStream()
        if (!current.getTracks().some(t => t.id === event.track.id)) {
          current.addTrack(event.track)
        }
        this.remoteScreenStreams.set(peerID, current)
        const participant = this.participants.get(peerID)
        if (participant) participant.screen_sharing = true

        event.track.addEventListener('ended', () => {
          this.removeRemoteScreenTrack(peerID, event.track.id)
        })
        this.emitStatus()
        return
      }

      if (event.streams.length > 0 && event.streams[0]) {
        for (const track of event.streams[0].getAudioTracks()) {
          if (!stream.getTracks().some(t => t.id === track.id)) {
            stream.addTrack(track)
          }
        }
      } else if (event.track.kind === 'audio' && !stream.getTracks().some(t => t.id === event.track.id)) {
        stream.addTrack(event.track)
      }

      audio.srcObject = stream
      audio.volume = 1.0
      audio.muted = this.deafened

      this.setupPeerAnalyser(peerID, stream)

      audio.play().then(() => {
        this.pushDiag('info', 'audio:play', `remote audio playing for ${peerID}`)
      }).catch((e) => {
        this.pushDiag('warn', 'audio:autoplay', e instanceof Error ? e.message : String(e))
        const resume = () => {
          void audio.play().catch(() => {})
          document.removeEventListener('click', resume)
          document.removeEventListener('keydown', resume)
        }
        document.addEventListener('click', resume, { once: true })
        document.addEventListener('keydown', resume, { once: true })
      })
    }

    pc.onicecandidateerror = (event) => {
      const ev = event as RTCPeerConnectionIceErrorEvent
      this.pushDiag('warn', 'ice:error', `${peerID} ${ev.errorCode} ${ev.errorText ?? ''}`.trim())
    }

    pc.oniceconnectionstatechange = () => {
      const iceState = pc.iceConnectionState
      const level = iceState === 'failed' ? 'error' : (iceState === 'disconnected' ? 'warn' : 'info')
      this.pushDiag(level, 'ice:state', `${peerID} ${iceState}`)
      if (iceState === 'connected' || iceState === 'completed') {
        this.clearIceRestartState(peerID)
      } else if (iceState === 'failed') {
        this.scheduleIceRestart(peerID, 'failed')
      } else if (iceState === 'disconnected') {
        this.scheduleIceRestart(peerID, 'disconnected')
      }
    }

    pc.onconnectionstatechange = () => {
      const s = pc.connectionState
      const level = s === 'failed' || s === 'disconnected' ? 'warn' : 'info'
      this.pushDiag(level, 'pc:state', `${peerID} ${s}`)

      if (s === 'connected') {
        this.clearDisconnectTimer(peerID)
        this.clearIceRestartState(peerID)
        return
      }

      if (s === 'disconnected') {
        this.scheduleIceRestart(peerID, 'disconnected')
        this.schedulePeerDisconnectCleanup(peerID)
        return
      }

      if (s === 'failed') {
        this.scheduleIceRestart(peerID, 'failed')
        this.schedulePeerDisconnectCleanup(peerID)
        return
      }

      if (s === 'closed') {
        this.clearDisconnectTimer(peerID)
        this.clearIceRestartState(peerID)
        this.removePeerFromSession(peerID)
        this.emitStatus()
      }
    }

    return state
  }

  private async flushPendingICECandidates(peerID: string, peer: PeerConnectionState): Promise<void> {
    if (peer.pendingRemoteCandidates.length === 0) return
    const pending = peer.pendingRemoteCandidates.splice(0, peer.pendingRemoteCandidates.length)

    for (const candidate of pending) {
      try {
        await peer.pc.addIceCandidate(candidate)
      } catch (e) {
        console.warn(`[voice] Failed to add buffered ICE candidate from ${peerID}:`, e)
      }
    }
  }

  private removePeerFromSession(peerID: string): void {
    this.removePeer(peerID)
    this.remoteScreenStreams.delete(peerID)
    this.peerStats.delete(peerID)
    this.participants.delete(peerID)
    this.peersMeta.delete(peerID)
  }

  private removePeer(peerID: string): void {
    this.clearDisconnectTimer(peerID)
    this.clearIceRestartState(peerID)
    const peer = this.peers.get(peerID)
    if (!peer) return
    this.peers.delete(peerID)
    this.peerStats.delete(peerID)
    this.remoteScreenStreams.delete(peerID)

    try {
      peer.audio.pause()
      peer.audio.srcObject = null
    } catch {
      // ignore
    }

    for (const t of peer.stream.getTracks()) {
      t.stop()
    }
    if (peer.sourceNode) {
      peer.sourceNode.disconnect()
      peer.sourceNode = null
    }

    peer.pc.ontrack = null
    peer.pc.onicecandidate = null
    peer.pc.onconnectionstatechange = null
    peer.pc.oniceconnectionstatechange = null
    peer.pc.onicecandidateerror = null
    peer.pc.onnegotiationneeded = null
    peer.pc.close()
  }

  private cleanupPeers(): void {
    for (const timer of this.disconnectTimers.values()) {
      clearTimeout(timer)
    }
    this.disconnectTimers.clear()

    for (const peerID of this.peers.keys()) {
      this.removePeer(peerID)
    }
    this.clearAllIceRestartState()
    this.remoteScreenStreams.clear()
    this.peerStats.clear()
  }

  private clearDisconnectTimer(peerID: string): void {
    const timer = this.disconnectTimers.get(peerID)
    if (!timer) return
    clearTimeout(timer)
    this.disconnectTimers.delete(peerID)
  }

  private schedulePeerDisconnectCleanup(peerID: string): void {
    this.clearDisconnectTimer(peerID)
    const timer = setTimeout(() => {
      this.disconnectTimers.delete(peerID)
      const current = this.peers.get(peerID)
      if (!current) return

      const state = current.pc.connectionState
      if (state === 'connected') {
        this.clearIceRestartState(peerID)
        return
      }

      const attempts = this.iceRestartAttempts.get(peerID) ?? 0
      if ((state === 'disconnected' || state === 'failed') && attempts < this.maxIceRestartAttempts) {
        this.pushDiag('warn', 'peer:degraded', `${peerID} still ${state}; waiting for ICE restart`)
        this.scheduleIceRestart(peerID, state === 'failed' ? 'failed' : 'disconnected')
        this.schedulePeerDisconnectCleanup(peerID)
        return
      }

      this.pushDiag('warn', 'peer:drop', `${peerID} removed after ${state} (${attempts} restart attempts)`)
      this.removePeerFromSession(peerID)
      this.emitStatus()
    }, this.peerDisconnectGraceMs)
    this.disconnectTimers.set(peerID, timer)
  }

  private scheduleIceRestart(peerID: string, reason: 'failed' | 'disconnected'): void {
    if (this.state !== 'connected') return
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    if (!this.peers.has(peerID)) return
    if (this.iceRestartTimers.has(peerID)) return

    const attempts = this.iceRestartAttempts.get(peerID) ?? 0
    if (attempts >= this.maxIceRestartAttempts) return

    const delay = reason === 'failed' ? 0 : this.disconnectedRestartDelayMs
    this.queueIceRestart(peerID, delay, reason)
  }

  private queueIceRestart(peerID: string, delayMs: number, reason: 'failed' | 'disconnected'): void {
    if (this.iceRestartTimers.has(peerID)) return
    const timer = setTimeout(() => {
      this.iceRestartTimers.delete(peerID)
      void this.performIceRestart(peerID, reason)
    }, Math.max(0, delayMs))
    this.iceRestartTimers.set(peerID, timer)
  }

  private async performIceRestart(peerID: string, reason: 'failed' | 'disconnected'): Promise<void> {
    const peer = this.peers.get(peerID)
    if (!peer) return
    if (this.state !== 'connected') return
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    if (peer.pc.connectionState === 'closed') return

    const attempts = this.iceRestartAttempts.get(peerID) ?? 0
    if (attempts >= this.maxIceRestartAttempts) return

    const now = Date.now()
    const lastAttemptAt = this.iceRestartLastAttempt.get(peerID) ?? 0
    const remainingCooldown = this.iceRestartCooldownMs - (now - lastAttemptAt)
    if (remainingCooldown > 0) {
      this.queueIceRestart(peerID, remainingCooldown, reason)
      return
    }

    if (peer.makingOffer || peer.pc.signalingState !== 'stable') {
      this.queueIceRestart(peerID, 800, reason)
      return
    }

    const nextAttempt = attempts + 1
    this.iceRestartAttempts.set(peerID, nextAttempt)
    this.iceRestartLastAttempt.set(peerID, now)
    this.pushDiag('warn', 'ice:restart', `${peerID} attempt ${nextAttempt}/${this.maxIceRestartAttempts} (${reason})`)

    try {
      peer.makingOffer = true
      const offer = await peer.pc.createOffer({ iceRestart: true })
      await peer.pc.setLocalDescription(offer)

      const sdp = peer.pc.localDescription?.sdp ?? offer.sdp ?? ''
      if (!sdp) {
        throw new Error('empty restart offer')
      }

      this.sendSignal({
        type: 'sdp_offer',
        from: this.selfPeerID,
        to: peerID,
        server_id: this.serverID,
        channel_id: this.channelID,
        payload: { sdp },
      })
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e)
      this.pushDiag('warn', 'ice:restart:error', `${peerID} ${message}`)
      if (nextAttempt < this.maxIceRestartAttempts) {
        this.scheduleIceRestart(peerID, reason)
      }
    } finally {
      peer.makingOffer = false
    }
  }

  private clearIceRestartTimer(peerID: string): void {
    const timer = this.iceRestartTimers.get(peerID)
    if (!timer) return
    clearTimeout(timer)
    this.iceRestartTimers.delete(peerID)
  }

  private clearIceRestartState(peerID: string): void {
    this.clearIceRestartTimer(peerID)
    this.iceRestartAttempts.delete(peerID)
    this.iceRestartLastAttempt.delete(peerID)
  }

  private clearAllIceRestartState(): void {
    for (const timer of this.iceRestartTimers.values()) {
      clearTimeout(timer)
    }
    this.iceRestartTimers.clear()
    this.iceRestartAttempts.clear()
    this.iceRestartLastAttempt.clear()
  }

  private cleanupLocalStream(): void {
    if (!this.localStream) return
    for (const track of this.localStream.getTracks()) {
      track.stop()
    }
    this.localStream = null
  }

  // ---------------------------------------------------------------------------
  // Mute / deafen
  // ---------------------------------------------------------------------------

  private applyMuteState(): void {
    const enabled = !this.muted && !this.deafened
    for (const track of this.localStream?.getAudioTracks() ?? []) {
      track.enabled = enabled
    }
    if (this.screenAudioTrack) {
      this.screenAudioTrack.enabled = enabled
    }
  }

  private applyDeafenState(): void {
    for (const peer of this.peers.values()) {
      peer.audio.muted = this.deafened
    }
  }

  private async createLocalStream(deviceId?: string): Promise<MediaStream> {
    const audio: MediaTrackConstraints = {
      echoCancellation: true,
      autoGainControl: true,
      noiseSuppression: this.noiseSuppressionEnabled,
    }
    if (deviceId) {
      audio.deviceId = { exact: deviceId }
    }
    return navigator.mediaDevices.getUserMedia({ audio, video: false })
  }

  private async applyOutputDevice(audio: HTMLAudioElement): Promise<void> {
    if (!this.outputDeviceId) return
    const mediaEl = audio as HTMLAudioElement & { setSinkId?: (sinkId: string) => Promise<void> }
    if (typeof mediaEl.setSinkId === 'function') {
      await mediaEl.setSinkId(this.outputDeviceId)
    }
  }

  private async replaceLocalAudioTrack(nextStream: MediaStream): Promise<void> {
    const nextTrack = nextStream.getAudioTracks()[0]
    if (!nextTrack) return

    const replaceOps: Promise<void>[] = []
    for (const peer of this.peers.values()) {
      for (const sender of peer.pc.getSenders()) {
        if (sender.track?.kind !== 'audio') continue
        if (peer.screenAudioSender && sender === peer.screenAudioSender) continue
        replaceOps.push(sender.replaceTrack(nextTrack))
      }
    }
    await Promise.allSettled(replaceOps)

    this.cleanupLocalStream()
    this.localStream = nextStream
    this.applyMuteState()
  }

  private removeRemoteScreenTrack(peerID: string, trackID: string): void {
    const stream = this.remoteScreenStreams.get(peerID)
    if (!stream) return

    for (const track of stream.getVideoTracks()) {
      if (track.id === trackID) {
        stream.removeTrack(track)
      }
    }

    if (stream.getVideoTracks().length === 0) {
      this.remoteScreenStreams.delete(peerID)
      const participant = this.participants.get(peerID)
      if (participant) participant.screen_sharing = false
    }
    this.emitStatus()
  }

  private buildScreenShares(): VoiceScreenShare[] {
    const shares: VoiceScreenShare[] = []

    if (this.screenStream && this.screenVideoTrack && this.selfPeerID) {
      const local = this.participants.get(this.selfPeerID)
      shares.push({
        peer_id: this.selfPeerID,
        user_id: local?.user_id || this.selfPeerID,
        username: local?.username || this.safeUsername('', this.selfPeerID, this.selfPeerID),
        avatar_url: local?.avatar_url,
        stream: this.screenStream,
        local: true,
      })
    }

    for (const [peerID, stream] of this.remoteScreenStreams.entries()) {
      if (stream.getVideoTracks().length === 0) continue
      const participant = this.participants.get(peerID)
      shares.push({
        peer_id: peerID,
        user_id: participant?.user_id || peerID,
        username: participant?.username || this.safeUsername('', peerID, peerID),
        avatar_url: participant?.avatar_url,
        stream,
        local: false,
      })
    }

    return shares.sort((a, b) => {
      const aLocal = a.local ? 0 : 1
      const bLocal = b.local ? 0 : 1
      if (aLocal !== bLocal) return aLocal - bLocal
      return a.username.localeCompare(b.username)
    })
  }

  private pushDiag(level: VoiceDiagnosticLevel, code: string, message: string): void {
    const entry: VoiceDiagnosticEvent = {
      ts: Date.now(),
      level,
      code,
      message,
    }
    this.diagnosticsEvents.push(entry)
    if (this.diagnosticsEvents.length > 120) {
      this.diagnosticsEvents.splice(0, this.diagnosticsEvents.length - 120)
    }

    const log = `[voice][${code}] ${message}`
    if (level === 'error') console.error(log)
    else if (level === 'warn') console.warn(log)
    else console.info(log)

    this.emitDiagnostics()
  }

  private emitDiagnostics(): void {
    if (!this.onDiagnosticsChange) return
    this.onDiagnosticsChange(this.getDiagnostics())
  }

  private startStatsLoop(): void {
    this.stopStatsLoop()
    this.statsLoop = setInterval(() => {
      void this.collectPeerStats()
    }, 5_000)
    void this.collectPeerStats()
  }

  private stopStatsLoop(): void {
    if (!this.statsLoop) return
    clearInterval(this.statsLoop)
    this.statsLoop = null
  }

  private async collectPeerStats(): Promise<void> {
    let participantsChanged = false

    for (const [peerID, peer] of this.peers.entries()) {
      const snapshot: VoicePeerConnectionStats = {
        peer_id: peerID,
        connection_state: peer.pc.connectionState,
        ice_connection_state: peer.pc.iceConnectionState,
        quality: 'unknown',
        quality_score: 0,
        loss_ratio: 0,
        packets_received: 0,
        packets_lost: 0,
        audio_jitter_ms: 0,
        round_trip_time_ms: 0,
        available_outgoing_bitrate: 0,
      }

      try {
        const stats = await peer.pc.getStats()
        stats.forEach((report) => {
          const item = report as unknown as Record<string, unknown>
          const reportType = this.safeString(item.type)

          if (reportType === 'candidate-pair') {
            const state = this.safeString(item.state)
            const nominated = this.safeBool(item.nominated)
            if (state === 'succeeded' && nominated) {
              snapshot.round_trip_time_ms = this.safeNumber(item.currentRoundTripTime) * 1000
              snapshot.available_outgoing_bitrate = this.safeNumber(item.availableOutgoingBitrate)
            }
            return
          }

          if (reportType !== 'inbound-rtp') return
          const kind = this.safeString(item.kind || item.mediaType)
          if (kind !== 'audio') return

          snapshot.packets_received += Math.max(0, Math.floor(this.safeNumber(item.packetsReceived)))
          snapshot.packets_lost += Math.max(0, Math.floor(this.safeNumber(item.packetsLost)))
          snapshot.audio_jitter_ms = Math.max(snapshot.audio_jitter_ms, this.safeNumber(item.jitter) * 1000)
        })
      } catch {
        // Keep stale stats if getStats fails transiently.
      }

      const total = snapshot.packets_received + snapshot.packets_lost
      snapshot.loss_ratio = total > 0 ? snapshot.packets_lost / total : 0
      const quality = this.deriveConnectionQuality(snapshot)
      snapshot.quality = quality.quality
      snapshot.quality_score = quality.score

      this.peerStats.set(peerID, snapshot)

      const participant = this.participants.get(peerID)
      if (participant) {
        const prevQuality = participant.connection_quality
        const prevScore = participant.quality_score
        participant.connection_quality = snapshot.quality
        participant.quality_score = snapshot.quality_score
        if (prevQuality !== participant.connection_quality || prevScore !== participant.quality_score) {
          participantsChanged = true
        }
      }
    }

    const stale: string[] = []
    for (const peerID of this.peerStats.keys()) {
      if (!this.peers.has(peerID)) stale.push(peerID)
    }
    for (const peerID of stale) {
      this.peerStats.delete(peerID)
    }

    if (this.screenVideoTrack) {
      await this.maybeAdjustScreenShareQoS()
    }

    if (participantsChanged) {
      this.emitStatus()
    }
    this.emitDiagnostics()
  }

  private deriveConnectionQuality(stats: VoicePeerConnectionStats): { quality: VoiceConnectionQuality; score: number } {
    if (stats.connection_state === 'failed' || stats.ice_connection_state === 'failed') {
      return { quality: 'poor', score: 10 }
    }
    if (stats.connection_state === 'disconnected' || stats.ice_connection_state === 'disconnected') {
      return { quality: 'poor', score: 20 }
    }
    if (stats.connection_state === 'connecting' || stats.ice_connection_state === 'checking') {
      return { quality: 'fair', score: 45 }
    }
    if (stats.connection_state !== 'connected') {
      return { quality: 'unknown', score: 0 }
    }

    let score = 100
    if (stats.round_trip_time_ms > 0) {
      score -= Math.min(35, Math.floor(stats.round_trip_time_ms / 6))
    }
    if (stats.audio_jitter_ms > 0) {
      score -= Math.min(25, Math.floor(stats.audio_jitter_ms))
    }
    if (stats.loss_ratio > 0) {
      score -= Math.min(40, Math.floor(stats.loss_ratio * 250))
    }

    const normalized = Math.max(0, Math.min(100, score))
    if (normalized >= 70) return { quality: 'good', score: normalized }
    if (normalized >= 40) return { quality: 'fair', score: normalized }
    return { quality: 'poor', score: normalized }
  }

  private async maybeAdjustScreenShareQoS(): Promise<void> {
    if (!this.screenVideoTrack) return
    if (this.screenVideoTrack.readyState !== 'live') return

    const nextProfile = this.pickScreenShareQoSProfile()
    await this.applyScreenShareQoS(nextProfile)
  }

  private pickScreenShareQoSProfile(): ScreenShareQoSProfile {
    if (!this.screenVideoTrack) return 'high'

    const snapshots = Array.from(this.peerStats.values())
      .filter(s => this.peers.has(s.peer_id))

    if (snapshots.length === 0) return 'high'

    let poorCount = 0
    let fairOrUnknownCount = 0
    let lowestScore = 100

    for (const snapshot of snapshots) {
      if (snapshot.quality === 'poor') poorCount++
      if (snapshot.quality === 'fair' || snapshot.quality === 'unknown') fairOrUnknownCount++
      if (snapshot.quality_score > 0) {
        lowestScore = Math.min(lowestScore, snapshot.quality_score)
      }
    }

    if (poorCount > 0 || lowestScore <= 35) return 'low'
    if (fairOrUnknownCount > 0 || lowestScore <= 65) return 'balanced'
    return 'high'
  }

  private async applyScreenShareQoS(profile: ScreenShareQoSProfile, force = false): Promise<void> {
    if (!this.screenVideoTrack) return
    if (this.screenVideoTrack.readyState !== 'live') return

    const now = Date.now()
    if (!force && profile === this.screenQoSProfile) return
    if (!force && now - this.lastScreenQoSAppliedAt < this.screenQoSCooldownMs) return

    const constraints = SCREEN_SHARE_QOS_CONSTRAINTS[profile]
    try {
      await this.screenVideoTrack.applyConstraints(constraints)
      this.screenQoSProfile = profile
      this.lastScreenQoSAppliedAt = now
      this.pushDiag('info', 'screen:qos', `profile=${profile}`)
    } catch (e) {
      this.lastScreenQoSAppliedAt = now
      const message = e instanceof Error ? e.message : String(e)
      this.pushDiag('warn', 'screen:qos:error', `${profile} ${message}`)
    }
  }

  // ---------------------------------------------------------------------------
  // Signal helpers
  // ---------------------------------------------------------------------------

  private sendSignal(signal: SignalMessage): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.pushDiag('warn', 'signal:send', `cannot send ${signal.type}; ws state=${this.ws?.readyState}`)
      return
    }
    const payload: SignalMessage = signal.from ? signal : { ...signal, from: this.selfPeerID || undefined }
    this.ws.send(JSON.stringify(payload))
  }

  private startKeepalive(): void {
    this.stopKeepalive()
    this.keepaliveInterval = setInterval(() => {
      this.sendSignal({ type: 'ping' })
    }, 10_000)
  }

  private stopKeepalive(): void {
    if (this.keepaliveInterval) {
      clearInterval(this.keepaliveInterval)
      this.keepaliveInterval = null
    }
  }

  private sendPeerState(): void {
    if (!this.serverID || !this.channelID) return
    const muted = this.deafened ? true : this.muted
    this.sendSignal({
      type: 'peer_state',
      server_id: this.serverID,
      channel_id: this.channelID,
      payload: {
        peer_id: this.selfPeerID,
        muted,
        deafened: this.deafened,
        screen_sharing: !!this.screenVideoTrack,
      },
    })
  }

  private updateSelfParticipantState(): void {
    const entry = this.participants.get(this.selfPeerID)
    if (!entry) return
    entry.deafened = this.deafened
    entry.muted = this.deafened ? true : this.muted
    entry.screen_sharing = !!this.screenVideoTrack
  }

  private onPeerState(fromPeerID: string, payload: unknown): void {
    const data = payload as Record<string, unknown> | null
    if (!data) return
    const peerID = this.safeString(data.peer_id) || fromPeerID
    if (!peerID) return
    let entry = this.participants.get(peerID)
    if (!entry) {
      const meta = this.peersMeta.get(peerID) ?? {
        peer_id: peerID,
        user_id: peerID,
        username: this.safeUsername('', peerID, peerID),
        avatar_url: '',
      }
      this.registerPeer(meta)
      entry = this.participants.get(peerID)
      if (!entry) return
    }
    const muted = this.safeBool(data.muted)
    const deafened = this.safeBool(data.deafened)
    const screenSharing = this.safeBool(data.screen_sharing)
    entry.deafened = deafened
    entry.muted = deafened ? true : muted
    entry.screen_sharing = screenSharing
    if (!screenSharing) {
      this.remoteScreenStreams.delete(peerID)
    }
    this.emitStatus()
  }

  private emitStatus(): void {
    const status = this.getStatus()
    this.onStatusChange(status)
    if (this.onDiagnosticsChange && status.diagnostics) {
      this.onDiagnosticsChange(status.diagnostics)
    }
  }

  // ---------------------------------------------------------------------------
  // Payload extraction
  // ---------------------------------------------------------------------------

  private extractPeers(payload: unknown): VoicePeerMeta[] {
    const raw = (payload as { peers?: unknown[] } | undefined)?.peers
    if (!Array.isArray(raw)) return []
    return raw.map(entry => this.normalizePeerMeta(entry)).filter((v): v is VoicePeerMeta => !!v)
  }

  private extractJoinPeer(payload: unknown): VoicePeerMeta | null {
    return this.normalizePeerMeta(payload)
  }

  private normalizePeerMeta(payload: unknown): VoicePeerMeta | null {
    const raw = payload as Record<string, unknown> | null
    if (!raw) return null

    const peerID = this.safeString(raw.peer_id)
    const userID = this.safeString(raw.user_id) || peerID
    if (!peerID) return null

    return {
      peer_id: peerID,
      user_id: userID,
      username: this.safeUsername(this.safeString(raw.username), userID, peerID),
      avatar_url: this.safeString(raw.avatar_url),
      muted: this.safeBool(raw.muted),
      deafened: this.safeBool(raw.deafened),
      screen_sharing: this.safeBool(raw.screen_sharing),
    }
  }

  private extractSDP(payload: unknown): string {
    return this.safeString((payload as { sdp?: unknown } | undefined)?.sdp)
  }

  private extractICE(payload: unknown): RTCIceCandidateInit | null {
    const data = payload as Record<string, unknown> | null
    if (!data) return null

    const candidate = this.safeString(data.candidate)
    if (!candidate) return null

    const init: RTCIceCandidateInit = { candidate }

    const sdpMid = this.safeString(data.sdp_mid)
    if (sdpMid) init.sdpMid = sdpMid

    const rawIndex = data.sdp_mline_index
    if (typeof rawIndex === 'number' && Number.isInteger(rawIndex)) {
      init.sdpMLineIndex = rawIndex
    }

    return init
  }

  private extractChannelStartedAt(payload: unknown): number | null {
    const raw = (payload as { channel_started_at?: unknown } | undefined)?.channel_started_at
    if (typeof raw === 'number' && Number.isFinite(raw) && raw > 0) {
      return Math.floor(raw)
    }
    if (typeof raw === 'string' && raw.trim() !== '') {
      const parsed = Number(raw)
      if (Number.isFinite(parsed) && parsed > 0) return Math.floor(parsed)
    }
    return null
  }

  // ---------------------------------------------------------------------------
  // URL helpers
  // ---------------------------------------------------------------------------

  private toSignalingURL(baseURL: string): string {
    const trimmed = baseURL.replace(/\/$/, '')
    if (trimmed.startsWith('https://')) {
      return `wss://${trimmed.slice('https://'.length)}/ws/signaling`
    }
    if (trimmed.startsWith('http://')) {
      return `ws://${trimmed.slice('http://'.length)}/ws/signaling`
    }
    if (trimmed.startsWith('wss://') || trimmed.startsWith('ws://')) {
      return `${trimmed}/ws/signaling`
    }
    return `ws://${trimmed}/ws/signaling`
  }

  private async openWebSocket(url: string): Promise<WebSocket> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(url)

      const onOpen = () => {
        ws.removeEventListener('open', onOpen)
        ws.removeEventListener('error', onError)
        resolve(ws)
      }

      const onError = () => {
        ws.removeEventListener('open', onOpen)
        ws.removeEventListener('error', onError)
        this.pushDiag('error', 'ws:error', 'failed to connect to signaling')
        reject(new Error('failed to connect to voice signaling'))
      }

      ws.addEventListener('open', onOpen)
      ws.addEventListener('error', onError)
    })
  }

  private async resolveIceServers(baseURL: string, authToken?: string): Promise<RTCIceServer[]> {
    const trimmed = baseURL.replace(/\/$/, '')
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 4000)

    try {
      const headers: Record<string, string> = {
        'Accept': 'application/json',
      }
      const token = (authToken || '').trim()
      if (token) headers.Authorization = `Bearer ${token}`

      const res = await fetch(`${trimmed}/api/v1/voice/ice-config`, {
        method: 'GET',
        cache: 'no-store',
        headers,
        signal: controller.signal,
      })
      if (!res.ok) return DEFAULT_ICE_SERVERS

      const data = (await res.json()) as IceConfigResponse
      const rawServers = Array.isArray(data.servers) ? data.servers as IceConfigServer[] : []
      const parsed: RTCIceServer[] = []

      for (const raw of rawServers) {
        if (!raw || typeof raw !== 'object') continue
        const urls = this.normalizeIceURLs(raw.urls)
        if (urls.length === 0) continue

        const server: RTCIceServer = { urls }
        const username = this.safeString(raw.username)
        const credential = this.safeString(raw.credential)
        if (username) server.username = username
        if (credential) server.credential = credential
        parsed.push(server)
      }

      return parsed.length > 0 ? parsed : DEFAULT_ICE_SERVERS
    } catch {
      return DEFAULT_ICE_SERVERS
    } finally {
      clearTimeout(timer)
    }
  }

  private normalizeIceURLs(raw: unknown): string[] {
    if (typeof raw === 'string') {
      const url = raw.trim()
      return url ? [url] : []
    }
    if (!Array.isArray(raw)) return []

    const urls: string[] = []
    for (const entry of raw) {
      if (typeof entry !== 'string') continue
      const url = entry.trim()
      if (!url) continue
      urls.push(url)
    }
    return urls
  }

  // ---------------------------------------------------------------------------
  // Safe type helpers
  // ---------------------------------------------------------------------------

  private safeString(value: unknown): string {
    return typeof value === 'string' ? value.trim() : ''
  }

  private safeBool(value: unknown): boolean {
    if (typeof value === 'boolean') return value
    if (typeof value === 'number') return value !== 0
    if (typeof value === 'string') {
      const normalized = value.trim().toLowerCase()
      if (normalized === 'true') return true
      if (normalized === 'false') return false
    }
    return false
  }

  private safeNumber(value: unknown): number {
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string') {
      const parsed = Number(value)
      if (Number.isFinite(parsed)) return parsed
    }
    return 0
  }

  private safeUsername(username: string, userID: string, peerID: string): string {
    if (username.trim().length > 0) return username.trim()
    if (userID.trim().length > 0) return userID.trim().slice(0, 12)
    return peerID.trim().slice(0, 12) || 'user'
  }

  // ---------------------------------------------------------------------------
  // Audio analysis (VAD for remote peers)
  // ---------------------------------------------------------------------------

  private initAudioAnalysis(): void {
    try {
      this.audioContext = new AudioContext()
      void this.audioContext.resume().catch(() => {})
      this.startSpeakingLoop()
    } catch {
      this.audioContext = null
    }
  }

  private cleanupAudioAnalysis(): void {
    if (this.speakingLoop) {
      clearInterval(this.speakingLoop)
      this.speakingLoop = null
    }

    if (this.audioContext) {
      void this.audioContext.close().catch(() => {})
      this.audioContext = null
    }
  }

  private setupPeerAnalyser(peerID: string, stream: MediaStream): void {
    if (!this.audioContext) return
    const peer = this.peers.get(peerID)
    if (!peer || peer.analyser) return

    try {
      const source = this.audioContext.createMediaStreamSource(stream)
      const analyser = this.audioContext.createAnalyser()
      analyser.fftSize = 256
      analyser.smoothingTimeConstant = 0.2
      analyser.minDecibels = -90
      analyser.maxDecibels = -10
      source.connect(analyser)

      peer.sourceNode = source
      peer.analyser = analyser
      peer.analyserData = new Uint8Array(analyser.frequencyBinCount) as Uint8Array<ArrayBuffer>
    } catch {
      // Keep voice functional even if analyser fails.
    }
  }

  private startSpeakingLoop(): void {
    if (this.speakingLoop) return

    this.speakingLoop = setInterval(() => {
      let changed = false
      let dominantPeerID = ''
      let dominantVolume = 0

      for (const [peerID, peer] of this.peers.entries()) {
        const participant = this.participants.get(peerID)
        if (!participant) continue

        if (!peer.analyser || !peer.analyserData) {
          if (participant.speaking || participant.volume !== 0) {
            participant.speaking = false
            participant.volume = 0
            changed = true
          }
          continue
        }

        peer.analyser.getByteTimeDomainData(peer.analyserData)
        let sumSquares = 0
        for (let i = 0; i < peer.analyserData.length; i++) {
          const normalized = (peer.analyserData[i] - 128) / 128
          sumSquares += normalized * normalized
        }
        const rms = Math.sqrt(sumSquares / peer.analyserData.length)
        const volume = Math.max(0, Math.min(1, rms))
        const speaking = volume > 0.02 && !participant.muted && !participant.deafened

        if (participant.speaking !== speaking) {
          participant.speaking = speaking
          changed = true
        }
        if (Math.abs(participant.volume - volume) > 0.005) {
          participant.volume = Number(volume.toFixed(4))
          changed = true
        }

        if (speaking && volume > dominantVolume) {
          dominantVolume = volume
          dominantPeerID = peerID
        }
      }

      const hasDominant = dominantPeerID !== '' && dominantVolume > 0.03
      for (const [peerID, participant] of this.participants.entries()) {
        const shouldBeDominant = hasDominant && peerID === dominantPeerID
        if (!!participant.dominant_speaker !== shouldBeDominant) {
          participant.dominant_speaker = shouldBeDominant
          changed = true
        }
      }

      if (changed) this.emitStatus()
    }, 120)
  }
}
