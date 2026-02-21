// P2P store using Svelte 5 runes
// Manages P2P host, peers, messages and room code via Wails bindings

import * as App from '../../../wailsjs/go/main/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'

export interface P2PPeer {
  id: string
  displayName: string
  avatarDataUrl?: string
  connected: boolean
  source: 'lan' | 'room'
}

export interface P2PMessage {
  id: string
  peerID: string
  direction: 'sent' | 'received'
  content: string
  sentAt: string
}

// ── State ────────────────────────────────────────────────────────────

let peers = $state<P2PPeer[]>([])
let activePeerID = $state<string | null>(null)
let messages = $state<Record<string, P2PMessage[]>>({})
let roomCode = $state('')
let joining = $state(false)
let sending = $state(false)
let initialized = $state(false)
let error = $state<string | null>(null)

/** Peer display names learned via profile handshake or GetP2PPeerName */
let peerNames = $state<Record<string, string>>({})

let pollingTimer: ReturnType<typeof setInterval> | null = null
let cleanupEvents: (() => void) | null = null

// ── Helpers ──────────────────────────────────────────────────────────

function isLocalAddress(addr: string): boolean {
  return /\/(127\.|192\.168\.|10\.|172\.(1[6-9]|2\d|3[01])\.|fd|fe80)/.test(addr)
}

function toP2PPeer(raw: { id: string; addresses: string[]; connected: boolean }, name?: string): P2PPeer {
  const hasLocal = (raw.addresses ?? []).some(isLocalAddress)
  return {
    id: raw.id,
    displayName: name || peerNames[raw.id] || '',
    connected: raw.connected,
    source: hasLocal ? 'lan' : 'room',
  }
}

function rawToMessage(raw: { id: string; peer_id: string; direction: string; content: string; sent_at: string }): P2PMessage {
  return {
    id: raw.id,
    peerID: raw.peer_id,
    direction: raw.direction as 'sent' | 'received',
    content: raw.content,
    sentAt: raw.sent_at,
  }
}

// ── Initialization ───────────────────────────────────────────────────

export async function initP2PStore(): Promise<void> {
  if (initialized) return
  try {
    await App.InitP2PHost()
    const code = await App.GetP2PRoomCode()
    roomCode = code ?? ''
    initialized = true
    startPeerPolling()
    listenMessages()
  } catch (e) {
    // Fallback: outside Wails runtime (dev/E2E) — silently degrade
    console.warn('[p2p] InitP2PHost failed (expected outside Wails):', e)
    roomCode = 'offline'
    initialized = true
  }
}

// ── Peer polling ─────────────────────────────────────────────────────

export function startPeerPolling(): void {
  if (pollingTimer) return
  fetchPeers()
  pollingTimer = setInterval(fetchPeers, 3000)
}

export function stopPeerPolling(): void {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

async function fetchPeers(): Promise<void> {
  try {
    const raw = await App.GetP2PPeers()
    const list = raw ?? []

    // Resolve display names for new peers
    for (const p of list) {
      if (!peerNames[p.id]) {
        try {
          const name = await App.GetP2PPeerName(p.id)
          if (name) peerNames[p.id] = name
        } catch { /* peer name not available yet */ }
      }
    }

    peers = list.map(p => toP2PPeer(p, peerNames[p.id]))
  } catch {
    // Silently ignore polling errors
  }
}

// ── Event listener ───────────────────────────────────────────────────

export function listenMessages(): void {
  if (cleanupEvents) return
  try {
    const off = EventsOn('p2p:message', (raw: any) => {
      const msg = rawToMessage(raw)
      const existing = messages[msg.peerID] ?? []
      messages = { ...messages, [msg.peerID]: [...existing, msg] }
    })
    cleanupEvents = off
  } catch {
    // Outside Wails runtime
  }
}

// ── Actions ──────────────────────────────────────────────────────────

export function setActivePeer(id: string | null): void {
  activePeerID = id
  if (id && !messages[id]) {
    loadMessages(id)
  }
}

export async function sendMessage(content: string): Promise<void> {
  if (!activePeerID || !content.trim() || sending) return
  sending = true
  error = null
  try {
    await App.SendP2PMessage(activePeerID, content)
    // Optimistically add to local state
    const msg: P2PMessage = {
      id: crypto.randomUUID(),
      peerID: activePeerID,
      direction: 'sent',
      content,
      sentAt: new Date().toISOString(),
    }
    const existing = messages[activePeerID] ?? []
    messages = { ...messages, [activePeerID]: [...existing, msg] }
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to send message'
  } finally {
    sending = false
  }
}

export async function joinRoom(code: string): Promise<void> {
  if (!code.trim() || joining) return
  joining = true
  error = null
  try {
    await App.JoinP2PRoom(code)
    // Trigger immediate peer refresh
    await fetchPeers()
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to join room'
  } finally {
    joining = false
  }
}

export async function loadMessages(peerID: string): Promise<void> {
  try {
    const raw = await App.GetP2PMessages(peerID, 100)
    const msgs = (raw ?? []).map(rawToMessage)
    // API returns newest first, reverse for display
    messages = { ...messages, [peerID]: msgs.reverse() }
  } catch {
    // Silently ignore — messages stay empty
  }
}

export async function sendProfile(displayName: string, avatarDataUrl?: string): Promise<void> {
  try {
    await App.SendP2PProfile(displayName, avatarDataUrl ?? '')
  } catch {
    // Best-effort
  }
}

// ── Cleanup ──────────────────────────────────────────────────────────

export function destroyP2PStore(): void {
  stopPeerPolling()
  if (cleanupEvents) {
    cleanupEvents()
    cleanupEvents = null
  }
  peers = []
  activePeerID = null
  messages = {}
  roomCode = ''
  joining = false
  sending = false
  initialized = false
  error = null
  peerNames = {}
}

// ── Reactive getters ─────────────────────────────────────────────────

export function getP2P() {
  return {
    get peers() { return peers },
    get activePeerID() { return activePeerID },
    get activePeer() { return peers.find(p => p.id === activePeerID) ?? null },
    get messages() { return activePeerID ? (messages[activePeerID] ?? []) : [] },
    get allMessages() { return messages },
    get roomCode() { return roomCode },
    get joining() { return joining },
    get sending() { return sending },
    get initialized() { return initialized },
    get error() { return error },
  }
}
