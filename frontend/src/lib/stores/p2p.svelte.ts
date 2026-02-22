// P2P store — gerencia peers descobertos, mensagens e sala DHT
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

// Estado reativo (module-level $state para SSR safety)
let peers = $state<P2PPeer[]>([])
let activePeerID = $state<string | null>(null)
let messages = $state<Record<string, P2PMessage[]>>({})
let roomCode = $state('')
let joining = $state(false)
let sending = $state(false)
let initialized = $state(false)

// Cache de nomes conhecidos por peer ID
const knownProfiles = new Map<string, { displayName: string; avatarDataUrl?: string }>()

export function getP2P() {
  return {
    get peers() { return peers },
    get activePeerID() { return activePeerID },
    get messages() { return messages },
    get roomCode() { return roomCode },
    get joining() { return joining },
    get sending() { return sending },
    get initialized() { return initialized },
  }
}

export async function initP2PStore(profile: { displayName: string; avatarDataUrl?: string } | null) {
  if (initialized) return
  try {
    await App.InitP2PHost()
    if (profile) {
      await App.SendP2PProfile(profile.displayName, profile.avatarDataUrl ?? '')
    }
    initialized = true
    startPeerPolling()
    listenMessages()
  } catch {
    // fora do Wails runtime (dev/E2E)
    initialized = true
    peers = []
  }
}

let pollingInterval: ReturnType<typeof setInterval> | null = null

function startPeerPolling() {
  if (pollingInterval) return
  pollingInterval = setInterval(async () => {
    try {
      const raw = await App.GetP2PPeers()
      peers = raw.map(p => ({
        id: p.id,
        displayName: knownProfiles.get(p.id)?.displayName ?? p.id.slice(0, 8),
        avatarDataUrl: knownProfiles.get(p.id)?.avatarDataUrl,
        connected: p.connected,
        source: 'lan' as const,
      }))
    } catch { /* silencioso */ }
  }, 3000)
}

function listenMessages() {
  try {
    EventsOn('p2p:message', (msg: { id: string; peer_id: string; direction: string; content: string; sent_at: string }) => {
      const m: P2PMessage = {
        id: msg.id,
        peerID: msg.peer_id,
        direction: msg.direction as 'sent' | 'received',
        content: msg.content,
        sentAt: msg.sent_at,
      }
      messages = {
        ...messages,
        [m.peerID]: [...(messages[m.peerID] ?? []), m],
      }
    })
  } catch { /* fora do Wails */ }
}

export function setActivePeer(id: string | null) {
  activePeerID = id
  if (id && !messages[id]) {
    loadMessages(id)
  }
}

export async function loadMessages(peerID: string) {
  try {
    const raw = await App.GetP2PMessages(peerID, 50)
    messages = {
      ...messages,
      [peerID]: raw.map(m => ({
        id: m.id,
        peerID: m.peer_id,
        direction: m.direction as 'sent' | 'received',
        content: m.content,
        sentAt: m.sent_at,
      })),
    }
  } catch { /* silencioso */ }
}

export async function sendMessage(peerID: string, content: string) {
  sending = true
  try {
    await App.SendP2PMessage(peerID, content)
    const msg: P2PMessage = {
      id: `local-${Date.now()}`,
      peerID,
      direction: 'sent',
      content,
      sentAt: new Date().toISOString(),
    }
    messages = {
      ...messages,
      [peerID]: [...(messages[peerID] ?? []), msg],
    }
  } catch (e) {
    console.error('p2p: send failed', e)
  } finally {
    sending = false
  }
}

export async function joinRoom(code: string) {
  joining = true
  try {
    await App.JoinP2PRoom(code)
  } catch (e) {
    console.error('p2p: join room failed', e)
  } finally {
    joining = false
  }
}

function generateRoomCode(): string {
  const words = ['amber', 'coral', 'delta', 'echo', 'frost', 'glow', 'haze', 'iris', 'jade', 'kite',
    'luna', 'mesa', 'nova', 'opal', 'peak', 'quartz', 'reef', 'sage', 'tide', 'vale']
  const word = words[Math.floor(Math.random() * words.length)]
  const num = Math.floor(1000 + Math.random() * 9000)
  return `${word}-${num}`
}

export async function createRoom() {
  try {
    const code = await App.GetP2PRoomCode()
    roomCode = code
    // Advertise on DHT so joiners can find us
    await App.JoinP2PRoom(code)
  } catch {
    roomCode = generateRoomCode()
  }
}

export function stopP2PStore() {
  if (pollingInterval) {
    clearInterval(pollingInterval)
    pollingInterval = null
  }
  initialized = false
}
