export interface VoiceParticipant {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
  volume: number
  speaking: boolean
}

export interface VoiceRTCStatus {
  state: 'disconnected' | 'connecting' | 'connected'
  channel_id: string
  muted: boolean
  deafened: boolean
  peer_count: number
  speakers: VoiceParticipant[]
}

interface VoicePeerMeta {
  peer_id: string
  user_id: string
  username: string
  avatar_url?: string
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
}

interface SignalMessage {
  type: string
  from?: string
  to?: string
  server_id?: string
  channel_id?: string
  payload?: unknown
}

interface PeerConnectionState {
  pc: RTCPeerConnection
  audio: HTMLAudioElement
  stream: MediaStream
  analyser: AnalyserNode | null
  analyserData: Uint8Array | null
  sourceNode: MediaStreamAudioSourceNode | null
}

const ICE_SERVERS: RTCIceServer[] = [
  { urls: ['stun:stun.l.google.com:19302'] },
  { urls: ['stun:stun1.l.google.com:19302'] },
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
  private state: VoiceRTCStatus['state'] = 'disconnected'
  private outputDeviceId = ''
  private audioContext: AudioContext | null = null
  private speakingLoop: ReturnType<typeof setInterval> | null = null

  constructor(
    private readonly onStatusChange: (status: VoiceRTCStatus) => void,
    private readonly onError: (message: string) => void,
  ) {}

  async join(opts: VoiceJoinOptions): Promise<void> {
    if (this.state !== 'disconnected') return

    this.state = 'connecting'
    this.serverID = opts.serverID
    this.channelID = opts.channelID
    this.selfPeerID = crypto.randomUUID()
    this.outputDeviceId = opts.outputDeviceId ?? ''
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
    })

    try {
      this.localStream = await this.createLocalStream(opts.inputDeviceId)
      this.applyMuteState()
      this.initAudioAnalysis()

      const wsURL = this.toSignalingURL(opts.baseURL)
      this.ws = await this.openWebSocket(wsURL)

      this.ws.onmessage = (event) => {
        void this.handleSignal(event.data)
      }

      this.ws.onclose = () => {
        if (this.state !== 'disconnected') {
          this.state = 'disconnected'
          this.cleanupPeers()
          this.cleanupLocalStream()
          this.cleanupAudioAnalysis()
          this.participants.clear()
          this.peersMeta.clear()
          this.emitStatus()
        }
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
        },
      })

      this.state = 'connected'
      this.emitStatus()
    } catch (e) {
      await this.leave()
      throw e
    }
  }

  async leave(): Promise<void> {
    if (this.state === 'disconnected') return

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
    this.cleanupPeers()
    this.cleanupLocalStream()
    this.cleanupAudioAnalysis()
    this.participants.clear()
    this.peersMeta.clear()

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
    }
  }

  toggleMute(): boolean {
    this.muted = !this.muted
    this.applyMuteState()
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

  private async handleSignal(rawData: unknown): Promise<void> {
    try {
      if (typeof rawData !== 'string') return
      const signal = JSON.parse(rawData) as SignalMessage

      switch (signal.type) {
        case 'peer_list':
          await this.onPeerList(signal.payload)
          break
        case 'peer_joined':
          this.onPeerJoined(signal.payload)
          break
        case 'peer_left':
          if (signal.from) this.onPeerLeft(signal.from)
          break
        case 'sdp_offer':
          if (signal.from) await this.onSDPOffer(signal.from, signal.payload)
          break
        case 'sdp_answer':
          if (signal.from) await this.onSDPAnswer(signal.from, signal.payload)
          break
        case 'ice_candidate':
          if (signal.from) await this.onICECandidate(signal.from, signal.payload)
          break
        case 'error':
          this.onError('voice signaling error')
          break
        default:
          break
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'invalid signaling message'
      this.onError(msg)
    }
  }

  private async onPeerList(payload: unknown): Promise<void> {
    const peers = this.extractPeers(payload)
    for (const peer of peers) {
      if (peer.peer_id === this.selfPeerID) continue

      this.peersMeta.set(peer.peer_id, peer)
      this.participants.set(peer.peer_id, {
        peer_id: peer.peer_id,
        user_id: peer.user_id,
        username: peer.username,
        avatar_url: peer.avatar_url,
        volume: 0,
        speaking: false,
      })

      const state = await this.ensurePeerConnection(peer.peer_id)
      const offer = await state.pc.createOffer()
      await state.pc.setLocalDescription(offer)

      this.sendSignal({
        type: 'sdp_offer',
        from: this.selfPeerID,
        to: peer.peer_id,
        server_id: this.serverID,
        channel_id: this.channelID,
        payload: { sdp: offer.sdp ?? '' },
      })
    }
    this.emitStatus()
  }

  private onPeerJoined(payload: unknown): void {
    const peer = this.extractJoinPeer(payload)
    if (!peer || peer.peer_id === this.selfPeerID) return

    this.peersMeta.set(peer.peer_id, peer)
    this.participants.set(peer.peer_id, {
      peer_id: peer.peer_id,
      user_id: peer.user_id,
      username: peer.username,
      avatar_url: peer.avatar_url,
      volume: 0,
      speaking: false,
    })
    this.emitStatus()
  }

  private onPeerLeft(peerID: string): void {
    this.removePeer(peerID)
    this.participants.delete(peerID)
    this.peersMeta.delete(peerID)
    this.emitStatus()
  }

  private async onSDPOffer(fromPeerID: string, payload: unknown): Promise<void> {
    const sdp = this.extractSDP(payload)
    if (!sdp) return

    const meta = this.peersMeta.get(fromPeerID) ?? {
      peer_id: fromPeerID,
      user_id: fromPeerID,
      username: this.safeUsername('', fromPeerID, fromPeerID),
      avatar_url: '',
    }

    if (!this.participants.has(fromPeerID)) {
      this.participants.set(fromPeerID, {
        peer_id: fromPeerID,
        user_id: meta.user_id,
        username: meta.username,
        avatar_url: meta.avatar_url,
        volume: 0,
        speaking: false,
      })
    }

    const state = await this.ensurePeerConnection(fromPeerID)
    await state.pc.setRemoteDescription({ type: 'offer', sdp })
    const answer = await state.pc.createAnswer()
    await state.pc.setLocalDescription(answer)

    this.sendSignal({
      type: 'sdp_answer',
      from: this.selfPeerID,
      to: fromPeerID,
      server_id: this.serverID,
      channel_id: this.channelID,
      payload: { sdp: answer.sdp ?? '' },
    })
    this.emitStatus()
  }

  private async onSDPAnswer(fromPeerID: string, payload: unknown): Promise<void> {
    const sdp = this.extractSDP(payload)
    if (!sdp) return
    const peer = this.peers.get(fromPeerID)
    if (!peer) return
    await peer.pc.setRemoteDescription({ type: 'answer', sdp })
  }

  private async onICECandidate(fromPeerID: string, payload: unknown): Promise<void> {
    const candidate = this.extractICE(payload)
    if (!candidate) return
    const peer = this.peers.get(fromPeerID)
    if (!peer) return
    await peer.pc.addIceCandidate(candidate)
  }

  private async ensurePeerConnection(peerID: string): Promise<PeerConnectionState> {
    const existing = this.peers.get(peerID)
    if (existing) return existing

    const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS })
    const stream = new MediaStream()
    const audio = new Audio()
    audio.autoplay = true
    audio.setAttribute('playsinline', 'true')
    audio.srcObject = stream
    audio.muted = this.deafened

    await this.applyOutputDevice(audio)

    if (this.localStream) {
      for (const track of this.localStream.getAudioTracks()) {
        pc.addTrack(track, this.localStream)
      }
    }

    pc.onicecandidate = (event) => {
      if (!event.candidate) return
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

    pc.ontrack = (event) => {
      if (event.streams.length > 0 && event.streams[0]) {
        for (const track of event.streams[0].getTracks()) {
          if (!stream.getTracks().some((t) => t.id === track.id)) {
            stream.addTrack(track)
          }
        }
      } else if (!stream.getTracks().some((t) => t.id === event.track.id)) {
        stream.addTrack(event.track)
      }

      this.setupPeerAnalyser(peerID, stream)
      void audio.play().catch(() => {
        this.onError('audio playback was blocked; click the app window and rejoin voice')
      })
    }

    pc.onconnectionstatechange = () => {
      const s = pc.connectionState
      if (s === 'failed' || s === 'disconnected' || s === 'closed') {
        this.removePeer(peerID)
        this.participants.delete(peerID)
        this.peersMeta.delete(peerID)
        this.emitStatus()
      }
    }

    const state: PeerConnectionState = {
      pc,
      audio,
      stream,
      analyser: null,
      analyserData: null,
      sourceNode: null,
    }
    this.peers.set(peerID, state)
    return state
  }

  private removePeer(peerID: string): void {
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
    peer.pc.close()
  }

  private cleanupPeers(): void {
    for (const peerID of this.peers.keys()) {
      this.removePeer(peerID)
    }
  }

  private cleanupLocalStream(): void {
    if (!this.localStream) return
    for (const track of this.localStream.getTracks()) {
      track.stop()
    }
    this.localStream = null
  }

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

  private extractPeers(payload: unknown): VoicePeerMeta[] {
    const raw = (payload as { peers?: unknown[] } | undefined)?.peers
    if (!Array.isArray(raw)) return []
    return raw.map((entry) => this.normalizePeerMeta(entry)).filter((v): v is VoicePeerMeta => !!v)
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

  private sendSignal(signal: SignalMessage): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    const payload: SignalMessage = signal.from ? signal : { ...signal, from: this.selfPeerID || undefined }
    this.ws.send(JSON.stringify(payload))
  }

  private emitStatus(): void {
    this.onStatusChange(this.getStatus())
  }

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

  private safeString(value: unknown): string {
    return typeof value === 'string' ? value.trim() : ''
  }

  private safeUsername(username: string, userID: string, peerID: string): string {
    if (username.trim().length > 0) return username.trim()
    if (userID.trim().length > 0) return userID.trim().slice(0, 12)
    return peerID.trim().slice(0, 12) || 'user'
  }

  private initAudioAnalysis(): void {
    try {
      this.audioContext = new AudioContext()
      void this.audioContext.resume().catch(() => {
        // ignore resume failures; VAD is best-effort
      })
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
      void this.audioContext.close().catch(() => {
        // ignore close failures
      })
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
      peer.analyserData = new Uint8Array(analyser.frequencyBinCount)
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
