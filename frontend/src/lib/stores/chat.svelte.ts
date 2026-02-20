// Chat store using Svelte 5 runes
// Manages messages for the active channel via Wails bindings

export interface MessageData {
  id: string
  channel_id: string
  author_id: string
  content: string
  type: 'text' | 'file' | 'system'
  edited_at?: string
  created_at: string
  author_name: string
  author_avatar: string
}

export interface SearchResultData extends MessageData {
  snippet: string
}

let messages = $state<MessageData[]>([])
let activeChannelId = $state<string | null>(null)
let loading = $state(false)
let sending = $state(false)
let hasMore = $state(true)
let searchResults = $state<SearchResultData[]>([])
let searchQuery = $state('')
let error = $state<string | null>(null)

export function getChat() {
  return {
    get messages() { return messages },
    get activeChannelId() { return activeChannelId },
    get loading() { return loading },
    get sending() { return sending },
    get hasMore() { return hasMore },
    get searchResults() { return searchResults },
    get searchQuery() { return searchQuery },
    get error() { return error },
  }
}

export async function loadMessages(channelID: string): Promise<void> {
  if (activeChannelId === channelID && messages.length > 0) return

  activeChannelId = channelID
  loading = true
  error = null
  searchResults = []
  searchQuery = ''

  try {
    // @ts-ignore - Wails binding
    const result: MessageData[] = await window.go.main.App.GetMessages(channelID, '', '', 50)
    // API returns newest first, reverse for display (oldest at top)
    messages = (result ?? []).reverse()
    hasMore = (result?.length ?? 0) >= 50
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load messages'
    messages = []
  } finally {
    loading = false
  }
}

export async function loadOlderMessages(): Promise<void> {
  if (!activeChannelId || !hasMore || loading) return

  const oldestMessage = messages[0]
  if (!oldestMessage) return

  loading = true
  try {
    // @ts-ignore - Wails binding
    const result: MessageData[] = await window.go.main.App.GetMessages(
      activeChannelId, oldestMessage.id, '', 50
    )
    const older = (result ?? []).reverse()
    messages = [...older, ...messages]
    hasMore = (result?.length ?? 0) >= 50
  } catch (e) {
    console.error('Failed to load older messages:', e)
  } finally {
    loading = false
  }
}

export async function sendMessage(channelID: string, authorID: string, content: string): Promise<MessageData | null> {
  sending = true
  error = null

  try {
    // @ts-ignore - Wails binding
    const msg: MessageData = await window.go.main.App.SendMessage(channelID, authorID, content)
    messages = [...messages, msg]
    return msg
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to send message'
    return null
  } finally {
    sending = false
  }
}

export async function editMessage(messageID: string, authorID: string, content: string): Promise<void> {
  error = null
  try {
    // @ts-ignore - Wails binding
    const updated: MessageData = await window.go.main.App.EditMessage(messageID, authorID, content)
    messages = messages.map(m => m.id === messageID ? updated : m)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to edit message'
  }
}

export async function deleteMessage(messageID: string, actorID: string, isManager: boolean): Promise<void> {
  error = null
  try {
    // @ts-ignore - Wails binding
    await window.go.main.App.DeleteMessage(messageID, actorID, isManager)
    messages = messages.filter(m => m.id !== messageID)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to delete message'
  }
}

export async function searchMessages(channelID: string, query: string): Promise<void> {
  if (!query.trim()) {
    searchResults = []
    searchQuery = ''
    return
  }

  searchQuery = query
  error = null

  try {
    // @ts-ignore - Wails binding
    const results: SearchResultData[] = await window.go.main.App.SearchMessages(channelID, query, 20)
    searchResults = results ?? []
  } catch (e) {
    error = e instanceof Error ? e.message : 'Search failed'
    searchResults = []
  }
}

export function clearSearch(): void {
  searchResults = []
  searchQuery = ''
}

export function clearChatError(): void {
  error = null
}

export function resetChat(): void {
  messages = []
  activeChannelId = null
  hasMore = true
  searchResults = []
  searchQuery = ''
  error = null
}
