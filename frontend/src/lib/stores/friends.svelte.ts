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

// ── Persistence ────────────────────────────────────────────────────────────

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

export function loadFriends() {
  state.loading = true
  loadFromStorage()
  state.loading = false
}

export function sendFriendRequest(username: string) {
  state.addFriendError = null
  state.addFriendSuccess = null

  const trimmed = username.trim().replace(/^@/, '')
  if (!trimmed) {
    state.addFriendError = 'Digite um nome de usuario.'
    return
  }

  // Check if already friends
  if (state.friends.some(f => f.username.toLowerCase() === trimmed.toLowerCase())) {
    state.addFriendError = `Voce ja e amigo de ${trimmed}.`
    return
  }

  // Check if already pending
  if (state.pendingRequests.some(r => r.username.toLowerCase() === trimmed.toLowerCase())) {
    state.addFriendError = `Ja existe um pedido pendente para ${trimmed}.`
    return
  }

  // Since there's no backend API for friends yet, we simulate:
  // Add as pending outgoing request
  const request: FriendRequest = {
    id: `req-${Date.now()}`,
    username: trimmed,
    display_name: trimmed,
    direction: 'outgoing',
    createdAt: new Date().toISOString(),
  }
  state.pendingRequests = [...state.pendingRequests, request]
  state.addFriendSuccess = `Pedido de amizade enviado para ${trimmed}!`
  persist()
}

export function acceptFriendRequest(requestId: string) {
  const request = state.pendingRequests.find(r => r.id === requestId)
  if (!request) return

  const friend: Friend = {
    id: `friend-${Date.now()}`,
    username: request.username,
    display_name: request.display_name,
    avatar_url: request.avatar_url,
    status: 'online',
  }

  state.friends = [...state.friends, friend]
  state.pendingRequests = state.pendingRequests.filter(r => r.id !== requestId)

  // Auto-create DM
  const dm: DMConversation = {
    id: `dm-${friend.id}`,
    friendId: friend.id,
    username: friend.username,
    display_name: friend.display_name,
    avatar_url: friend.avatar_url,
    status: friend.status,
  }
  state.dms = [...state.dms, dm]
  persist()
}

export function rejectFriendRequest(requestId: string) {
  state.pendingRequests = state.pendingRequests.filter(r => r.id !== requestId)
  persist()
}

export function removeFriend(friendId: string) {
  state.friends = state.friends.filter(f => f.id !== friendId)
  state.dms = state.dms.filter(d => d.friendId !== friendId)
  persist()
}

export function blockUser(friendId: string) {
  const friend = state.friends.find(f => f.id === friendId)
  if (friend) {
    state.blocked = [...state.blocked, friend.username]
    removeFriend(friendId)
  }
  persist()
}

export function unblockUser(username: string) {
  state.blocked = state.blocked.filter(u => u !== username)
  persist()
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
