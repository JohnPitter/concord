// Friends store — dual-mode (Wails bindings or HTTP API) with localStorage cache
// Svelte 5 runes

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'
import { getAuth } from './auth.svelte'
import { getSettings } from './settings.svelte'
import { isServerMode } from '../api/mode'
import { apiFriends } from '../api/friends'
import type { FriendRequestView, FriendView } from '../api/friends'
import { notify, requestNotificationPermission } from '../services/notifications'
import { toastInfo } from './toast.svelte'

export type FriendStatus = 'online' | 'idle' | 'dnd' | 'offline'

export interface Friend {
  id: string
  username: string
  display_name: string
  avatar_url?: string
  status: FriendStatus
  activity?: string
  game?: string
  gameSince?: string
  streaming?: boolean
  streamTitle?: string
}

export interface FriendRequest {
  id: string
  username: string
  display_name: string
  avatar_url?: string
  direction: 'incoming' | 'outgoing'
  createdAt: string
}

export interface DMConversation {
  id: string
  friendId: string
  username: string
  display_name: string
  avatar_url?: string
  status: FriendStatus
  lastMessage?: string
  unread?: number
}

export interface DMMessage {
  id: string
  dmId: string
  senderId: string
  content: string
  timestamp: string
}

type FriendsTab = 'online' | 'all' | 'pending' | 'blocked'

interface FriendsState {
  friends: Friend[]
  pendingRequests: FriendRequest[]
  blocked: string[]
  dms: DMConversation[]
  dmMessages: Record<string, DMMessage[]>
  tab: FriendsTab
  loading: boolean
  activeDMId: string | null
  addFriendError: string | null
  addFriendSuccess: string | null
}

const STORAGE_KEY = 'concord_friends'
const DM_MESSAGES_KEY = 'concord_dm_messages'
const POLL_INTERVAL = 2_500 // near real-time polling for friends/pending requests
const WAILS_STORE_TIMEOUT_MS = 10_000
const STALE_PRESENCE_MS = 20_000

let pollTimer: ReturnType<typeof setInterval> | null = null
let loadFriendsInFlight = false
let lastPresenceSyncAt = 0
let consecutivePresenceSyncFailures = 0

// Track recently rejected request IDs so polling doesn't re-add them
const recentlyRejected = new Set<string>()
const seenIncomingRequestIDs = new Set<string>()

async function withTimeout<T>(promise: Promise<T>, timeoutMs: number, errorMessage: string): Promise<T> {
  let handle: ReturnType<typeof setTimeout> | null = null
  const timeoutPromise = new Promise<T>((_, reject) => {
    handle = setTimeout(() => reject(new Error(errorMessage)), timeoutMs)
  })
  try {
    return await Promise.race([promise, timeoutPromise])
  } finally {
    if (handle) clearTimeout(handle)
  }
}

const state = $state<FriendsState>({
  friends: [],
  pendingRequests: [],
  blocked: [],
  dms: [],
  dmMessages: {},
  tab: 'online',
  loading: false,
  activeDMId: null,
  addFriendError: null,
  addFriendSuccess: null,
})

// ── Helpers ────────────────────────────────────────────────────────────

function currentUserID(): string | null {
  const auth = getAuth()
  return auth.user?.id ?? null
}

function currentUsername(): string {
  const auth = getAuth()
  return auth.user?.username ?? ''
}

function notificationsEnabled(): boolean {
  return getSettings().notificationsEnabled
}

function maybeNotify(title: string, body: string): void {
  if (!notificationsEnabled()) return

  toastInfo(title, body)
  void requestNotificationPermission().then((granted) => {
    if (!granted) return
    notify(title, body)
  })
}

function forceOfflineFriend(friend: Friend): Friend {
  return {
    ...friend,
    status: 'offline',
    activity: undefined,
    game: undefined,
    gameSince: undefined,
    streaming: false,
    streamTitle: undefined,
  }
}

function forceAllPresenceOffline(): void {
  state.friends = state.friends.map(forceOfflineFriend)
  state.dms = state.dms.map(dm => ({ ...dm, status: 'offline' }))
}

function markPresenceSyncSuccess(): void {
  lastPresenceSyncAt = Date.now()
  consecutivePresenceSyncFailures = 0
}

function markPresenceSyncFailure(): void {
  consecutivePresenceSyncFailures += 1
  const now = Date.now()
  if (lastPresenceSyncAt === 0 || now - lastPresenceSyncAt >= STALE_PRESENCE_MS) {
    forceAllPresenceOffline()
    persist()
  }
}

// ── Persistence (localStorage cache) ────────────────────────────────────

function persist() {
  try {
    const data = {
      friends: state.friends,
      pendingRequests: state.pendingRequests,
      blocked: state.blocked,
      dms: state.dms,
      cachedAt: Date.now(),
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  } catch { /* localStorage unavailable */ }
}

function persistDMMessages() {
  try {
    localStorage.setItem(DM_MESSAGES_KEY, JSON.stringify(state.dmMessages))
  } catch { /* localStorage unavailable */ }
}

function loadFromStorage() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const data = JSON.parse(raw)
      const cachedAt = typeof data.cachedAt === 'number' ? data.cachedAt : 0
      const stalePresence = cachedAt <= 0 || Date.now() - cachedAt > STALE_PRESENCE_MS

      if (data.friends) {
        state.friends = stalePresence
          ? data.friends.map((f: Friend) => forceOfflineFriend(f))
          : data.friends
      }
      if (data.pendingRequests) {
        state.pendingRequests = data.pendingRequests
        seenIncomingRequestIDs.clear()
        for (const req of state.pendingRequests) {
          if (req.direction === 'incoming') seenIncomingRequestIDs.add(req.id)
        }
      }
      if (data.blocked) state.blocked = data.blocked
      if (data.dms) {
        state.dms = stalePresence
          ? data.dms.map((dm: DMConversation) => ({ ...dm, status: 'offline' }))
          : data.dms
      }
    }
    const dmRaw = localStorage.getItem(DM_MESSAGES_KEY)
    if (dmRaw) {
      state.dmMessages = JSON.parse(dmRaw)
    }
  } catch { /* ignore parse errors */ }
}

// ── Backend sync helpers ──────────────────────────────────────────────

function mapBackendFriend(f: FriendView): Friend {
  const statusRaw = (f.status || '').toLowerCase()
  const status: FriendStatus =
    statusRaw === 'online' || statusRaw === 'idle' || statusRaw === 'dnd' || statusRaw === 'offline'
      ? statusRaw
      : 'offline'

  const activity = (f.activity || '').trim()

  return {
    id: f.id,
    username: f.username,
    display_name: f.display_name,
    avatar_url: f.avatar_url || undefined,
    status,
    activity: activity || (status === 'online' ? 'Online' : undefined),
    game: f.game,
    gameSince: f.gameSince,
    streaming: !!f.streaming,
    streamTitle: f.streamTitle,
  }
}

function mapBackendRequest(r: FriendRequestView): FriendRequest {
  return {
    id: r.id,
    username: r.username,
    display_name: r.display_name,
    avatar_url: r.avatar_url || undefined,
    direction: r.direction as 'incoming' | 'outgoing',
    createdAt: r.createdAt,
  }
}

function syncDMsFromFriends(friends: Friend[]) {
  // Keep existing DM metadata in sync with the latest friend state.
  const byFriendID = new Map(friends.map(f => [f.id, f] as const))
  state.dms = state.dms
    .filter(d => byFriendID.has(d.friendId))
    .map(d => {
      const friend = byFriendID.get(d.friendId)!
      return {
        ...d,
        username: friend.username,
        display_name: friend.display_name,
        avatar_url: friend.avatar_url,
        status: friend.status,
      }
    })

  // Ensure every friend has a DM conversation entry
  const existingDMFriendIds = new Set(state.dms.map(d => d.friendId))
  const newDMs: DMConversation[] = []

  for (const f of friends) {
    if (!existingDMFriendIds.has(f.id)) {
      newDMs.push({
        id: `dm-${f.id}`,
        friendId: f.id,
        username: f.username,
        display_name: f.display_name,
        avatar_url: f.avatar_url,
        status: f.status,
      })
    }
  }

  // Remove DMs for users no longer in friends list
  state.dms = [...state.dms, ...newDMs]
}

async function fetchFriendsFromBackend(): Promise<Friend[]> {
  const uid = currentUserID()
  if (!uid) return []

  await ensureValidToken()
  let raw: FriendView[]
  if (isServerMode()) {
    raw = await apiFriends.getFriends()
  } else {
    raw = await withTimeout(
      App.GetFriends(uid) as unknown as Promise<FriendView[]>,
      WAILS_STORE_TIMEOUT_MS,
      'GetFriends timeout',
    )
  }
  const mapped = (raw ?? []).map(mapBackendFriend)
  markPresenceSyncSuccess()
  return mapped
}

async function fetchPendingFromBackend(): Promise<FriendRequest[]> {
  const uid = currentUserID()
  if (!uid) return []

  await ensureValidToken()
  let raw: FriendRequestView[]
  if (isServerMode()) {
    raw = await apiFriends.getPendingRequests()
  } else {
    raw = await withTimeout(
      App.GetPendingRequests(uid) as unknown as Promise<FriendRequestView[]>,
      WAILS_STORE_TIMEOUT_MS,
      'GetPendingRequests timeout',
    )
  }
  return (raw ?? []).map(mapBackendRequest)
}

// ── Selectors ────────────────────────────────────────────────────────────

export function getFriends() {
  return {
    get friends() { return state.friends },
    get pendingRequests() { return state.pendingRequests },
    get dms() { return state.dms },
    get dmMessages() { return state.dmMessages },
    get tab() { return state.tab },
    get loading() { return state.loading },
    get activeDMId() { return state.activeDMId },
    get addFriendError() { return state.addFriendError },
    get addFriendSuccess() { return state.addFriendSuccess },
    get onlineFriends() {
      return state.friends.filter(f => f.status !== 'offline')
    },
    get activeDM() {
      return state.dms.find(d => d.id === state.activeDMId) ?? null
    },
    get activeDMMessages(): DMMessage[] {
      if (!state.activeDMId) return []
      return state.dmMessages[state.activeDMId] ?? []
    },
    get pendingCount() {
      return state.pendingRequests.filter(r => r.direction === 'incoming').length
    },
  }
}

// ── Actions ──────────────────────────────────────────────────────────────

export function setFriendsTab(tab: FriendsTab) {
  state.tab = tab
  state.addFriendError = null
  state.addFriendSuccess = null
}

export function openDM(dmId: string | null) {
  state.activeDMId = dmId
  if (!dmId) return
  const dm = state.dms.find(d => d.id === dmId)
  if (!dm) return
  if (dm.unread) {
    dm.unread = 0
    state.dms = [...state.dms]
    persist()
  }
}

export async function loadFriends() {
  if (loadFriendsInFlight) return
  loadFriendsInFlight = true
  state.loading = true
  // Load cached data immediately so the UI isn't blank
  loadFromStorage()

  try {
    const [friends, pending] = await Promise.all([
      fetchFriendsFromBackend(),
      fetchPendingFromBackend(),
    ])
    state.friends = friends
    state.pendingRequests = pending
    syncDMsFromFriends(friends)
    for (const req of pending) {
      if (req.direction === 'incoming') seenIncomingRequestIDs.add(req.id)
    }
    persist()
  } catch {
    // Keep cached data on error
    markPresenceSyncFailure()
  } finally {
    state.loading = false
    loadFriendsInFlight = false
  }

  // Start polling for incoming requests
  startPolling()
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const [friends, pending] = await Promise.all([
        fetchFriendsFromBackend(),
        fetchPendingFromBackend(),
      ])
      state.friends = friends
      // Filter out recently rejected requests to avoid the "reappear" glitch
      state.pendingRequests = pending.filter(r => !recentlyRejected.has(r.id))
      for (const req of state.pendingRequests) {
        if (req.direction !== 'incoming') continue
        if (seenIncomingRequestIDs.has(req.id)) continue
        seenIncomingRequestIDs.add(req.id)
        maybeNotify('Novo pedido de amizade', `${req.display_name} enviou um pedido.`)
      }
      syncDMsFromFriends(friends)
      persist()
    } catch {
      markPresenceSyncFailure()
    }
  }, POLL_INTERVAL)
}

export function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

export async function sendFriendRequest(username: string) {
  state.addFriendError = null
  state.addFriendSuccess = null

  const trimmed = username.trim().replace(/^@/, '')
  if (!trimmed) {
    state.addFriendError = 'Digite um nome de usuario.'
    return
  }

  const uid = currentUserID()
  if (!uid) {
    state.addFriendError = 'Voce precisa estar logado.'
    return
  }

  if (trimmed.toLowerCase() === currentUsername().toLowerCase()) {
    state.addFriendError = 'Voce nao pode enviar convite para si mesmo.'
    return
  }

  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiFriends.sendRequest(trimmed)
    } else {
      await App.SendFriendRequest(uid, trimmed)
    }

    state.addFriendSuccess = `Pedido de amizade enviado para ${trimmed}!`

    // Refresh pending requests from backend
    const pending = await fetchPendingFromBackend()
    state.pendingRequests = pending
    persist()
  } catch (e) {
    state.addFriendError = e instanceof Error ? e.message : 'Falha ao enviar pedido.'
  }
}

export async function acceptFriendRequest(requestId: string) {
  const uid = currentUserID()
  if (!uid) return

  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiFriends.acceptRequest(requestId)
    } else {
      await App.AcceptFriendRequest(requestId, uid)
    }

    // Refresh both lists from backend
    const [friends, pending] = await Promise.all([
      fetchFriendsFromBackend(),
      fetchPendingFromBackend(),
    ])
    state.friends = friends
    state.pendingRequests = pending
    seenIncomingRequestIDs.delete(requestId)
    syncDMsFromFriends(friends)
    persist()
  } catch {
    // Optimistic remove on error is too risky — keep state
  }
}

export async function rejectFriendRequest(requestId: string) {
  const uid = currentUserID()
  if (!uid) return

  // Mark as recently rejected so polling won't re-add it
  recentlyRejected.add(requestId)
  seenIncomingRequestIDs.delete(requestId)

  // Optimistic removal — update UI immediately
  const previous = state.pendingRequests
  state.pendingRequests = state.pendingRequests.filter(r => r.id !== requestId)
  persist()

  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiFriends.rejectRequest(requestId)
    } else {
      await App.RejectFriendRequest(requestId, uid)
    }
    // Backend confirmed — clear guard after next poll cycle
    setTimeout(() => recentlyRejected.delete(requestId), POLL_INTERVAL + 5000)
  } catch {
    // Rollback on failure
    recentlyRejected.delete(requestId)
    state.pendingRequests = previous
    persist()
  }
}

export async function removeFriend(friendId: string) {
  const uid = currentUserID()
  if (!uid) return

  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiFriends.removeFriend(friendId)
    } else {
      await App.RemoveFriend(uid, friendId)
    }

    state.friends = state.friends.filter(f => f.id !== friendId)
    state.dms = state.dms.filter(d => d.friendId !== friendId)
    persist()
  } catch {
    // Refresh on error
    const friends = await fetchFriendsFromBackend()
    state.friends = friends
    syncDMsFromFriends(friends)
    persist()
  }
}

export async function blockUser(friendId: string) {
  const uid = currentUserID()
  if (!uid) return

  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiFriends.blockUser(friendId)
    } else {
      await App.BlockUser(uid, friendId)
    }

    const friend = state.friends.find(f => f.id === friendId)
    if (friend) {
      state.blocked = [...state.blocked, friend.username]
    }
    state.friends = state.friends.filter(f => f.id !== friendId)
    state.dms = state.dms.filter(d => d.friendId !== friendId)
    persist()
  } catch {
    // Refresh
    const friends = await fetchFriendsFromBackend()
    state.friends = friends
    syncDMsFromFriends(friends)
    persist()
  }
}

export async function unblockUser(username: string) {
  const uid = currentUserID()
  if (!uid) return

  try {
    await ensureValidToken()
    if (isServerMode()) {
      // For server mode, we need the user ID — but unblock takes friendID in the API
      // The API handler does a username lookup, so we pass the username as the friendID param
      await apiFriends.unblockUser(username)
    } else {
      await App.UnblockUser(uid, username)
    }

    state.blocked = state.blocked.filter(u => u !== username)
    persist()
  } catch {
    // keep state
  }
}

export function clearFriendNotifications() {
  state.addFriendError = null
  state.addFriendSuccess = null
}

export function sendDMMessage(dmId: string, senderId: string, content: string) {
  const trimmed = content.trim()
  if (!trimmed) return

  const msg: DMMessage = {
    id: `dm-msg-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    dmId,
    senderId,
    content: trimmed,
    timestamp: new Date().toISOString(),
  }

  if (!state.dmMessages[dmId]) {
    state.dmMessages[dmId] = []
  }
  state.dmMessages = { ...state.dmMessages, [dmId]: [...(state.dmMessages[dmId] ?? []), msg] }

  // Update lastMessage on the DM conversation
  const currentID = currentUserID()
  const isIncoming = currentID !== null && senderId !== currentID
  const dm = state.dms.find(d => d.id === dmId)
  if (dm) {
    dm.lastMessage = trimmed
    if (isIncoming && state.activeDMId !== dmId) {
      dm.unread = (dm.unread ?? 0) + 1
      maybeNotify(dm.display_name, trimmed)
    }
    state.dms = [...state.dms]
    persist()
  }

  persistDMMessages()
}

export function getDMMessages(dmId: string): DMMessage[] {
  return state.dmMessages[dmId] ?? []
}
