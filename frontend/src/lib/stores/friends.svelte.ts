export type FriendStatus = 'online' | 'idle' | 'dnd' | 'offline'

export interface Friend {
  id: string
  username: string
  display_name: string
  avatar_url?: string
  status: FriendStatus
  activity?: string          // e.g. "Jogando Valorant"
  game?: string              // game name for ActiveNow card
  gameSince?: string         // e.g. "4 horas atrás"
  streaming?: boolean
  streamTitle?: string
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

type FriendsTab = 'online' | 'all' | 'pending' | 'blocked'

interface FriendsState {
  friends: Friend[]
  dms: DMConversation[]
  tab: FriendsTab
  loading: boolean
  activeDMId: string | null   // null = friends view, string = DM conversation
}

const state = $state<FriendsState>({
  friends: [],
  dms: [],
  tab: 'online',
  loading: false,
  activeDMId: null,
})

// ── Selectors ───────────────────────────────────────────────────────────────

export function getFriends() {
  return {
    get friends() { return state.friends },
    get dms() { return state.dms },
    get tab() { return state.tab },
    get loading() { return state.loading },
    get activeDMId() { return state.activeDMId },
    get onlineFriends() {
      return state.friends.filter(f => f.status !== 'offline')
    },
    get activeDM() {
      return state.dms.find(d => d.id === state.activeDMId) ?? null
    },
  }
}

// ── Actions ─────────────────────────────────────────────────────────────────

export function setFriendsTab(tab: FriendsTab) {
  state.tab = tab
}

export function openDM(dmId: string | null) {
  state.activeDMId = dmId
}

// Seed with realistic mock data — replaced by real API when backend supports it
export function loadFriends() {
  state.loading = true

  const mockFriends: Friend[] = [
    {
      id: 'f1',
      username: 'Capitão Sparrow',
      display_name: 'Capitão Sparrow',
      status: 'online',
      game: 'Valorant',
      activity: 'Jogando Valorant',
      gameSince: '4 horas atrás',
    },
    {
      id: 'f2',
      username: 'RobertPad',
      display_name: 'RobertPad ★ 4',
      status: 'online',
      streaming: true,
      streamTitle: 'GIANT SERVER — Minecraft',
      activity: 'Transmitindo GIANT SERVER',
    },
    {
      id: 'f3',
      username: 'Raid 🔥',
      display_name: 'Raid 🔥',
      status: 'online',
      activity: 'No Servidor de Voz',
    },
    {
      id: 'f4',
      username: 'rodrigopacera',
      display_name: 'rodrigopacera',
      status: 'online',
      activity: 'Ouvindo Spotify',
    },
    {
      id: 'f5',
      username: 'SamuraiX',
      display_name: 'SamuraiX',
      status: 'idle',
    },
    {
      id: 'f6',
      username: 'ShadowMaster',
      display_name: 'ShadowMaster',
      status: 'online',
      game: 'League of Legends',
      activity: 'Jogando League of Legends',
      gameSince: '2 horas atrás',
    },
    {
      id: 'f7',
      username: 'gq07',
      display_name: 'gq07',
      status: 'offline',
    },
    {
      id: 'f8',
      username: 'Colosseum Maycole',
      display_name: 'Colosseum Maycole',
      status: 'offline',
    },
    {
      id: 'f9',
      username: 'ZzDougie',
      display_name: 'ZzDougie',
      status: 'offline',
    },
    {
      id: 'f10',
      username: 'RobertPad2',
      display_name: 'RobertPad',
      status: 'offline',
    },
  ]

  const mockDMs: DMConversation[] = mockFriends.slice(0, 6).map(f => ({
    id: `dm-${f.id}`,
    friendId: f.id,
    username: f.username,
    display_name: f.display_name,
    avatar_url: f.avatar_url,
    status: f.status,
    lastMessage: undefined,
    unread: 0,
  }))

  state.friends = mockFriends
  state.dms = mockDMs
  state.loading = false
}
