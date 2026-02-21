// Chat store using Svelte 5 runes
// Manages messages for the active channel via Wails bindings

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'

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

export interface AttachmentData {
  id: string
  filename: string
  size_bytes: number
  mime_type: string
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
let attachmentsByMessage = $state<Record<string, AttachmentData[]>>({})

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
    get attachmentsByMessage() { return attachmentsByMessage },
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
    await ensureValidToken()
    const result = await App.GetMessages(channelID, '', '', 50)
    // API returns newest first, reverse for display (oldest at top)
    const msgs = (result ?? []) as unknown as MessageData[]
    messages = msgs.reverse()
    hasMore = msgs.length >= 50
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
    await ensureValidToken()
    const result = await App.GetMessages(
      activeChannelId, oldestMessage.id, '', 50
    )
    const older = ((result ?? []) as unknown as MessageData[]).reverse()
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
    await ensureValidToken()
    const msg = await App.SendMessage(channelID, authorID, content)
    const data = msg as unknown as MessageData
    messages = [...messages, data]
    return data
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
    await ensureValidToken()
    const updated = await App.EditMessage(messageID, authorID, content)
    const data = updated as unknown as MessageData
    messages = messages.map(m => m.id === messageID ? data : m)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to edit message'
  }
}

export async function deleteMessage(messageID: string, actorID: string, isManager: boolean): Promise<void> {
  error = null
  try {
    await ensureValidToken()
    await App.DeleteMessage(messageID, actorID, isManager)
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
    const results = await App.SearchMessages(channelID, query, 20)
    searchResults = (results ?? []) as unknown as SearchResultData[]
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

export async function loadAttachments(messageID: string): Promise<void> {
  try {
    const result = await App.GetAttachments(messageID)
    const atts = (result ?? []) as unknown as AttachmentData[]
    if (atts.length > 0) {
      attachmentsByMessage = { ...attachmentsByMessage, [messageID]: atts }
    }
  } catch (e) {
    console.error('Failed to load attachments:', e)
  }
}

export async function uploadFile(messageID: string, filename: string, data: number[]): Promise<AttachmentData | null> {
  error = null
  try {
    const att = await App.UploadFile(messageID, filename, data)
    const attData = att as unknown as AttachmentData
    const existing = attachmentsByMessage[messageID] ?? []
    attachmentsByMessage = { ...attachmentsByMessage, [messageID]: [...existing, attData] }
    return attData
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to upload file'
    return null
  }
}

export async function downloadFile(attachmentID: string): Promise<number[] | null> {
  try {
    const data = await App.DownloadFile(attachmentID)
    return data
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to download file'
    return null
  }
}

export async function deleteAttachment(attachmentID: string): Promise<void> {
  error = null
  try {
    await App.DeleteAttachment(attachmentID)
    // Remove from local state
    const updated = { ...attachmentsByMessage }
    for (const msgId of Object.keys(updated)) {
      updated[msgId] = updated[msgId].filter(a => a.id !== attachmentID)
      if (updated[msgId].length === 0) delete updated[msgId]
    }
    attachmentsByMessage = updated
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to delete attachment'
  }
}

export function resetChat(): void {
  messages = []
  activeChannelId = null
  hasMore = true
  searchResults = []
  searchQuery = ''
  error = null
  attachmentsByMessage = {}
}
