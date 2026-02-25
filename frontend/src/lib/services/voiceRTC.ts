export interface VoiceParticipant {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  volume: number
  speaking: boolean
  muted: boolean
  deafened: boolean
}

export interface VoiceRTCStatus {
  state: 'disconnected' | 'connecting' | 'connected'
  channel_id: string
  muted: boolean
  deafened: boolean
  peer_count: number
  speakers: VoiceParticipant[]
  channel_started_at?: number
}

interface VoicePeerMeta {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  muted?: boolean
  deafened?: boolean
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
}

interface SignalMessage {
  type: string
  from?: string
  to?: string
  server_id?: string
  channel_id?: string
  payload?: unknown
}

// Per-peer state including the RTCPeerConnection and perfect negotiation flags.
interface PeerConnectionState {
  pc: RTCPeerConnection
  audio: HTMLAudioElement
  stream: MediaStream
  analyser: AnalyserNode | null
  analyserData: Uint8Array<ArrayBuffer> | null
  sourceNode: MediaStreamAudioSourceNode | null
  // Perfect negotiation state (MDN pattern)
  makingOffer: boolean
  ignoreOffer: boolean
  isSettingRemoteAnswerPending: boolean
  polite: boolean
}

interface IceConfigServer {
  urls?: unknown
  username?: unknown
  credential?: unknown
}

interface IceConfigResponse {
  servers?: unknown
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
  private audioContext: AudioContext | null = null
  private speakingLoop: ReturnType<typeof setInterval> | null = null
  private disconnectTimers = new Map<string, ReturnType<typeof setTimeout>>()
  private iceServers: RTCIceServer[] = DEFAULT_ICE_SERVERS

  // WebSocket reconnect state
  private joinOpts: VoiceJoinOptions | null = null
  private reconnectAttempts = 0
  private readonly maxReconnectAttempts = 5
  private readonly reconnectBaseDelay = 1000 // 1s, exponential backoff
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private intentionalDisconnect = false
  private keepaliveInterval: ReturnType<typeof setInterval> | null = null

  constructor(
    private readonly onStatusChange: (status: VoiceRTCStatus) => void,
    private readonly onError: (message: string) => void,
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
    this.channelStartedAt = null
    this.peersMeta.clear()
    this.participants.clear()
    this.emitStatus()

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
      console.warn('[voice] Failed to fetch ICE config, using defaults')
      this.iceServers = DEFAULT_ICE_SERVERS
    }
    console.info('[voice] ICE servers:', JSON.stringify(this.iceServers.map(s => ({ urls: s.urls, hasCredentials: !!(s.username || s.credential) }))))

    try {
      await this.connectWebSocket(opts, localUsername)
      this.state = 'connected'
      this.emitStatus()
    } catch (e) {
      await this.leave()
      const detail = e instanceof Error ? e.message : String(e)
      throw new Error(`Failed to connect to voice signaling server: ${detail}`)
    }
  }

  private async connectWebSocket(opts: VoiceJoinOptions, localUsername: string): Promise<void> {
    const wsURL = this.toSignalingURL(opts.baseURL)
    console.info('[voice] Connecting to signaling:', wsURL)
    this.ws = await this.openWebSocket(wsURL)
    console.info('[voice] WebSocket connected')

    this.ws.onmessage = (event) => {
      void this.handleSignal(event.data)
    }

    this.ws.onclose = () => {
      this.stopKeepalive()
      if (this.intentionalDisconnect || this.state === 'disconnected') {
        if (this.state !== 'disconnected') {
          this.state = 'disconnected'
          this.cleanupPeers()
          this.cleanupLocalStream()
          this.cleanupAudioAnalysis()
          this.participants.clear()
          this.peersMeta.clear()
          this.channelStartedAt = null
          this.emitStatus()
        }
        return
      }

      console.warn('[voice] WebSocket closed unexpectedly, attempting reconnect...')
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
      },
    })
    console.info('[voice] Join signal sent, selfPeerID:', this.selfPeerID)

    this.startKeepalive()
  }

  private async scheduleReconnect(): Promise<void> {
    if (this.intentionalDisconnect || this.state === 'disconnected') return
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error(`[voice] Max reconnect attempts (${this.maxReconnectAttempts}) reached, disconnecting`)
      this.state = 'disconnected'
      this.cleanupPeers()
      this.cleanupLocalStream()
      this.cleanupAudioAnalysis()
      this.participants.clear()
      this.peersMeta.clear()
      this.channelStartedAt = null
      this.emitStatus()
      this.onError('voice connection lost')
      return
    }

    this.reconnectAttempts++
    const delay = Math.min(this.reconnectBaseDelay * Math.pow(2, this.reconnectAttempts - 1), 10_000)
    console.info(`[voice] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})...`)

    await new Promise<void>((resolve) => {
      this.reconnectTimer = setTimeout(resolve, delay)
    })

    if (this.intentionalDisconnect || this.state === 'disconnected') return

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
        })
      }

      await this.connectWebSocket(opts, localUsername)
      this.reconnectAttempts = 0
      console.info('[voice] WebSocket reconnected successfully')
      this.emitStatus()
    } catch (e) {
      console.warn('[voice] Reconnect attempt failed:', e)
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
    this.cleanupPeers()
    this.cleanupLocalStream()
    this.cleanupAudioAnalysis()
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
      const byUser = a.username.localeCompare(b.username)
      if (byUser !== 0) return byUser
      return a.peer_id.localeCompare(b.peer_id)
    })

    return {
      state: this.state,
      channel_id: this.channelID,
      muted: this.muted,
      deafened: this.deafened,
      peer_count: Math.max(0, this.participants.size - 1),
      speakers,
      channel_started_at: this.channelStartedAt ?? undefined,
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

    const nextStream = await this.createLocalStream(deviceId)
    const nextTrack = nextStream.getAudioTracks()[0]
    if (!nextTrack) return

    const replaceOps: Promise<void>[] = []
    for (const peer of this.peers.values()) {
      for (const sender of peer.pc.getSenders()) {
        if (sender.track?.kind === 'audio') {
          replaceOps.push(sender.replaceTrack(nextTrack))
        }
      }
    }
    await Promise.allSettled(replaceOps)

    this.cleanupLocalStream()
    this.localStream = nextStream
    this.applyMuteState()
  }

  async setOutputDevice(deviceId: string): Promise<void> {
    this.outputDeviceId = deviceId
    const applyOps: Promise<void>[] = []
    for (const peer of this.peers.values()) {
      applyOps.push(this.applyOutputDevice(peer.audio))
    }
    await Promise.allSettled(applyOps)
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
      console.warn('[voice] Failed to parse signaling message')
      return
    }

    // Log every signal except keepalive pings
    if (signal.type !== 'ping') {
      console.info(`[voice] << ${signal.type} from=${signal.from ?? '-'} to=${signal.to ?? '-'}`)
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
          console.error('[voice] Server error:', errPayload?.message ?? 'unknown')
          break
        }
        default:
          break
      }
    } catch (e) {
      // Log but do NOT propagate to onError — that would surface transient
      // signaling issues as user-visible errors and potentially break the flow.
      console.error(`[voice] Error handling signal '${signal.type}':`, e)
    }
  }

  // ---------------------------------------------------------------------------
  // Peer list / join / leave handlers
  // ---------------------------------------------------------------------------

  private onPeerList(payload: unknown): void {
    const peers = this.extractPeers(payload)
    const startedAt = this.extractChannelStartedAt(payload)
    if (startedAt) this.channelStartedAt = startedAt
    console.info(`[voice] peer_list: ${peers.length} peers`, peers.map(p => p.peer_id))

    for (const peer of peers) {
      if (peer.peer_id === this.selfPeerID) continue
      this.registerPeer(peer)
      // Create a PeerConnection with tracks — onnegotiationneeded will fire
      // and the perfect negotiation pattern will handle the offer/answer exchange.
      this.ensurePeerConnection(peer.peer_id)
    }
    this.emitStatus()
  }

  private onPeerJoined(payload: unknown): void {
    const peer = this.extractJoinPeer(payload)
    if (!peer || peer.peer_id === this.selfPeerID) return
    console.info(`[voice] peer_joined: ${peer.peer_id} (${peer.username})`)
    this.registerPeer(peer)
    // Create a PeerConnection — onnegotiationneeded fires automatically
    this.ensurePeerConnection(peer.peer_id)
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
    })
  }

  private onPeerLeft(peerID: string): void {
    this.removePeer(peerID)
    this.participants.delete(peerID)
    this.peersMeta.delete(peerID)
    this.emitStatus()
  }

  // ---------------------------------------------------------------------------
  // Perfect Negotiation Pattern (MDN)
  // https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Perfect_negotiation
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

    const peer = this.ensurePeerConnection(fromPeerID)
    const pc = peer.pc
    const description: RTCSessionDescriptionInit = { type, sdp }

    try {
      if (type === 'offer') {
        // Perfect negotiation: check for collision
        const sigState = pc.signalingState
        const readyForOffer =
          !peer.makingOffer &&
          (sigState === 'stable' || peer.isSettingRemoteAnswerPending)
        const offerCollision = !readyForOffer

        console.info(`[voice] offer check: polite=${peer.polite}, makingOffer=${peer.makingOffer}, signalingState=${sigState}, readyForOffer=${readyForOffer}, collision=${offerCollision}`)

        peer.ignoreOffer = !peer.polite && offerCollision
        if (peer.ignoreOffer) {
          console.info(`[voice] Ignoring colliding offer from ${fromPeerID} (we are impolite)`)
          return
        }

        if (offerCollision) {
          console.info(`[voice] Offer collision with ${fromPeerID}, rolling back (we are polite)`)
        }

        peer.isSettingRemoteAnswerPending = false
        await pc.setRemoteDescription(description)
        console.info(`[voice] Remote offer set for ${fromPeerID}, signalingState=${pc.signalingState}`)

        await pc.setLocalDescription()
        const answer = pc.localDescription
        console.info(`[voice] Local answer created for ${fromPeerID}, type=${answer?.type}, sdp=${(answer?.sdp ?? '').length}B`)
        if (answer && answer.sdp) {
          console.info(`[voice] >> sdp_answer to ${fromPeerID}`)
          this.sendSignal({
            type: 'sdp_answer',
            from: this.selfPeerID,
            to: fromPeerID,
            server_id: this.serverID,
            channel_id: this.channelID,
            payload: { sdp: answer.sdp },
          })
        }
      } else {
        // Answer
        const sigState = pc.signalingState
        console.info(`[voice] Setting remote answer for ${fromPeerID}, signalingState=${sigState}`)
        peer.isSettingRemoteAnswerPending = true
        await pc.setRemoteDescription(description)
        peer.isSettingRemoteAnswerPending = false
        console.info(`[voice] Remote answer set for ${fromPeerID}, connectionState=${pc.connectionState}`)
      }
    } catch (e) {
      peer.isSettingRemoteAnswerPending = false
      const errMsg = e instanceof Error ? e.message : String(e)
      console.error(`[voice] FAILED to handle ${type} from ${fromPeerID}: ${errMsg}`, e)
    }
  }

  private async onICECandidate(fromPeerID: string, payload: unknown): Promise<void> {
    const candidate = this.extractICE(payload)
    if (!candidate) return
    const peer = this.peers.get(fromPeerID)
    if (!peer) return

    try {
      await peer.pc.addIceCandidate(candidate)
    } catch (e) {
      if (!peer.ignoreOffer) {
        console.warn(`[voice] Failed to add ICE candidate from ${fromPeerID}:`, e)
      }
    }
  }

  // ---------------------------------------------------------------------------
  // PeerConnection lifecycle
  // ---------------------------------------------------------------------------

  private ensurePeerConnection(peerID: string): PeerConnectionState {
    const existing = this.peers.get(peerID)
    if (existing) return existing

    console.info(`[voice] Creating PeerConnection for ${peerID}`)
    const pc = new RTCPeerConnection({ iceServers: this.iceServers })
    const stream = new MediaStream()
    const audio = new Audio()
    audio.autoplay = true
    audio.setAttribute('playsinline', 'true')
    audio.volume = 1.0
    audio.srcObject = stream
    audio.muted = this.deafened

    void this.applyOutputDevice(audio)

    // Add local tracks — this triggers onnegotiationneeded
    if (this.localStream) {
      for (const track of this.localStream.getAudioTracks()) {
        pc.addTrack(track, this.localStream)
      }
    }

    // Polite peer: the one with the smaller peerID rolls back on collision.
    const polite = this.selfPeerID < peerID

    const state: PeerConnectionState = {
      pc,
      audio,
      stream,
      analyser: null,
      analyserData: null,
      sourceNode: null,
      makingOffer: false,
      ignoreOffer: false,
      isSettingRemoteAnswerPending: false,
      polite,
    }
    this.peers.set(peerID, state)

    // --- Perfect Negotiation: onnegotiationneeded ---
    pc.onnegotiationneeded = async () => {
      try {
        console.info(`[voice] negotiationneeded for ${peerID}`)
        state.makingOffer = true
        await pc.setLocalDescription()
        const offer = pc.localDescription
        if (offer) {
          console.info(`[voice] >> sdp_offer to ${peerID}, sdp=${(offer.sdp ?? '').length}B`)
          this.sendSignal({
            type: 'sdp_offer',
            from: this.selfPeerID,
            to: peerID,
            server_id: this.serverID,
            channel_id: this.channelID,
            payload: { sdp: offer.sdp ?? '' },
          })
        }
      } catch (e) {
        console.error(`[voice] negotiationneeded error for ${peerID}:`, e)
      } finally {
        state.makingOffer = false
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
      console.info(`[voice] ontrack from ${peerID}: kind=${event.track.kind}, id=${event.track.id}`)
      if (event.streams.length > 0 && event.streams[0]) {
        for (const track of event.streams[0].getTracks()) {
          if (!stream.getTracks().some(t => t.id === track.id)) {
            stream.addTrack(track)
          }
        }
      } else if (!stream.getTracks().some(t => t.id === event.track.id)) {
        stream.addTrack(event.track)
      }

      audio.srcObject = stream
      audio.volume = 1.0
      audio.muted = this.deafened

      this.setupPeerAnalyser(peerID, stream)

      audio.play().then(() => {
        console.info(`[voice] Audio playing for ${peerID}`)
      }).catch((e) => {
        console.warn(`[voice] Autoplay blocked for ${peerID}:`, e)
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
      console.warn(`[voice] ICE error for ${peerID}: ${ev.errorCode} ${ev.errorText ?? ''}`)
    }

    pc.oniceconnectionstatechange = () => {
      console.info(`[voice] ICE state ${peerID}: ${pc.iceConnectionState}`)
    }

    pc.onconnectionstatechange = () => {
      const s = pc.connectionState
      console.info(`[voice] Connection state ${peerID}: ${s}`)

      if (s === 'connected') {
        this.clearDisconnectTimer(peerID)
        return
      }

      if (s === 'disconnected') {
        this.clearDisconnectTimer(peerID)
        const timer = setTimeout(() => {
          const current = this.peers.get(peerID)
          if (!current || current.pc.connectionState === 'connected') return
          this.removePeer(peerID)
          this.participants.delete(peerID)
          this.peersMeta.delete(peerID)
          this.emitStatus()
        }, 8_000)
        this.disconnectTimers.set(peerID, timer)
        return
      }

      if (s === 'failed' || s === 'closed') {
        this.clearDisconnectTimer(peerID)
        this.removePeer(peerID)
        this.participants.delete(peerID)
        this.peersMeta.delete(peerID)
        this.emitStatus()
      }
    }

    return state
  }

  private removePeer(peerID: string): void {
    this.clearDisconnectTimer(peerID)
    const peer = this.peers.get(peerID)
    if (!peer) return
    this.peers.delete(peerID)

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
  }

  private clearDisconnectTimer(peerID: string): void {
    const timer = this.disconnectTimers.get(peerID)
    if (!timer) return
    clearTimeout(timer)
    this.disconnectTimers.delete(peerID)
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
  }

  private applyDeafenState(): void {
    for (const peer of this.peers.values()) {
      peer.audio.muted = this.deafened
    }
  }

  private async createLocalStream(deviceId?: string): Promise<MediaStream> {
    const audio = deviceId
      ? { deviceId: { exact: deviceId } }
      : true
    return navigator.mediaDevices.getUserMedia({ audio, video: false })
  }

  private async applyOutputDevice(audio: HTMLAudioElement): Promise<void> {
    if (!this.outputDeviceId) return
    const mediaEl = audio as HTMLAudioElement & { setSinkId?: (sinkId: string) => Promise<void> }
    if (typeof mediaEl.setSinkId === 'function') {
      await mediaEl.setSinkId(this.outputDeviceId)
    }
  }

  // ---------------------------------------------------------------------------
  // Signal helpers
  // ---------------------------------------------------------------------------

  private sendSignal(signal: SignalMessage): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn(`[voice] Cannot send signal '${signal.type}': WS not open (state=${this.ws?.readyState})`)
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
      },
    })
  }

  private updateSelfParticipantState(): void {
    const entry = this.participants.get(this.selfPeerID)
    if (!entry) return
    entry.deafened = this.deafened
    entry.muted = this.deafened ? true : this.muted
  }

  private onPeerState(fromPeerID: string, payload: unknown): void {
    const data = payload as Record<string, unknown> | null
    if (!data) return
    const peerID = this.safeString(data.peer_id) || fromPeerID
    if (!peerID) return
    const entry = this.participants.get(peerID)
    if (!entry) return
    const muted = this.safeBool(data.muted)
    const deafened = this.safeBool(data.deafened)
    entry.deafened = deafened
    entry.muted = deafened ? true : muted
    this.emitStatus()
  }

  private emitStatus(): void {
    this.onStatusChange(this.getStatus())
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

      for (const [peerID, peer] of this.peers.entries()) {
        const participant = this.participants.get(peerID)
        if (!participant) continue

        if (!peer.analyser || !peer.analyserData) {
          if (participant.speaking) {
            participant.speaking = false
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
        const speaking = rms > 0.02

        if (participant.speaking !== speaking) {
          participant.speaking = speaking
          changed = true
        }
      }

      if (changed) this.emitStatus()
    }, 120)
  }
}
