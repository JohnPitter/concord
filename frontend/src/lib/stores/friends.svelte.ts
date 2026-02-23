// Friends store — dual-mode (Wails bindings or HTTP API) with localStorage cache
// Svelte 5 runes

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'
import { getAuth } from './auth.svelte'
import { isServerMode } from '../api/mode'
import { apiFriends } from '../api/friends'
import type { FriendRequestView, FriendView } from '../api/friends'

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
const POLL_INTERVAL = 30_000 // 30s polling for pending requests

let pollTimer: ReturnType<typeof setInterval> | null = null

// Track recently rejected request IDs so polling doesn't re-add them
const recentlyRejected = new Set<string>()

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

// ── Persistence (localStorage cache) ────────────────────────────────────

function persist() {
  try {
    const data = {
      friends: state.friends,
      pendingRequests: state.pendingRequests,
      blocked: state.blocked,
      dms: state.dms,
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
      if (data.friends) state.friends = data.friends
      if (data.pendingRequests) state.pendingRequests = data.pendingRequests
      if (data.blocked) state.blocked = data.blocked
      if (data.dms) state.dms = data.dms
    }
    const dmRaw = localStorage.getItem(DM_MESSAGES_KEY)
    if (dmRaw) {
      state.dmMessages = JSON.parse(dmRaw)
    }
  } catch { /* ignore parse errors */ }
}

// ── Backend sync helpers ──────────────────────────────────────────────

function mapBackendFriend(f: FriendView): Friend {
  return {
    id: f.id,
    username: f.username,
    display_name: f.display_name,
    avatar_url: f.avatar_url || undefined,
    status: 'offline' as FriendStatus,
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
  const friendIds = new Set(friends.map(f => f.id))
  state.dms = [
    ...state.dms.filter(d => friendIds.has(d.friendId)),
    ...newDMs,
  ]
}

async function fetchFriendsFromBackend(): Promise<Friend[]> {
  const uid = currentUserID()
  if (!uid) return []

  await ensureValidToken()
  let raw: FriendView[]
  if (isServerMode()) {
    raw = await apiFriends.getFriends()
  } else {
    raw = await App.GetFriends(uid) as unknown as FriendView[]
  }
  return (raw ?? []).map(mapBackendFriend)
}

async function fetchPendingFromBackend(): Promise<FriendRequest[]> {
  const uid = currentUserID()
  if (!uid) return []

  await ensureValidToken()
  let raw: FriendRequestView[]
  if (isServerMode()) {
    raw = await apiFriends.getPendingRequests()
  } else {
    raw = await App.GetPendingRequests(uid) as unknown as FriendRequestView[]
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
}

export async function loadFriends() {
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
    persist()
  } catch {
    // Keep cached data on error
  } finally {
    state.loading = false
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
      syncDMsFromFriends(friends)
      persist()
    } catch {
      // Silently ignore polling errors
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
  const dm = state.dms.find(d => d.id === dmId)
  if (dm) {
    dm.lastMessage = trimmed
    state.dms = [...state.dms]
    persist()
  }

  persistDMMessages()
}

export function getDMMessages(dmId: string): DMMessage[] {
  return state.dmMessages[dmId] ?? []
}
