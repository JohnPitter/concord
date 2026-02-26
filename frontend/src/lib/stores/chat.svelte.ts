// Chat store using Svelte 5 runes
// Manages messages for the active channel — Wails bindings (P2P) or HTTP API (Server)

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'
import { isServerMode } from '../api/mode'
import { apiChat } from '../api/chat'

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

const MESSAGE_POLL_INTERVAL = 5_000 // 5s polling for new messages
const UNREAD_POLL_INTERVAL = 5_000 // 5s polling for unread counts
const WAILS_STORE_TIMEOUT_MS = 10_000
const LAST_READ_KEY = 'concord:lastReadMessages'

let messages = $state<MessageData[]>([])
let activeChannelId = $state<string | null>(null)
let loading = $state(false)
let sending = $state(false)
let hasMore = $state(true)
let searchResults = $state<SearchResultData[]>([])
let searchQuery = $state('')
let error = $state<string | null>(null)
let attachmentsByMessage = $state<Record<string, AttachmentData[]>>({})
let messagePollTimer: ReturnType<typeof setInterval> | null = null

// --- Unread tracking ---
let unreadCounts = $state<Record<string, number>>({})
let unreadPollTimer: ReturnType<typeof setInterval> | null = null
let trackedChannelIds: string[] = []
let lastReadMap: Record<string, string> = {}

function loadLastRead(): Record<string, string> {
  try {
    const raw = localStorage.getItem(LAST_READ_KEY)
    return raw ? JSON.parse(raw) : {}
  } catch { return {} }
}

function saveLastRead() {
  try {
    localStorage.setItem(LAST_READ_KEY, JSON.stringify(lastReadMap))
  } catch { /* ignore */ }
}

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
    get unreadCounts() { return unreadCounts },
  }
}

export async function loadMessages(channelID: string): Promise<void> {
  // Stop polling for previous channel
  stopMessagePolling()

  activeChannelId = channelID
  loading = true
  error = null
  searchResults = []
  searchQuery = ''

  try {
    await ensureValidToken()
    let result
    if (isServerMode()) {
      result = await apiChat.getMessages(channelID, '', '', 50)
    } else {
      result = await withTimeout(
        App.GetMessages(channelID, '', '', 50),
        WAILS_STORE_TIMEOUT_MS,
        'GetMessages timeout',
      )
    }
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

  // Mark channel as read now that the user is viewing it
  markChannelAsRead(channelID)

  // Start polling for new messages
  startMessagePolling(channelID)
}

export async function loadOlderMessages(): Promise<void> {
  if (!activeChannelId || !hasMore || loading) return

  const oldestMessage = messages[0]
  if (!oldestMessage) return

  loading = true
  try {
    await ensureValidToken()
    let result
    if (isServerMode()) {
      result = await apiChat.getMessages(activeChannelId, oldestMessage.id, '', 50)
    } else {
      result = await withTimeout(
        App.GetMessages(
          activeChannelId, oldestMessage.id, '', 50,
        ),
        WAILS_STORE_TIMEOUT_MS,
        'GetMessages timeout',
      )
    }
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
    let msg
    if (isServerMode()) {
      msg = await apiChat.sendMessage(channelID, content)
    } else {
      msg = await App.SendMessage(channelID, authorID, content)
    }
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
    let updated
    if (isServerMode()) {
      updated = await apiChat.editMessage(messageID, content)
    } else {
      updated = await App.EditMessage(messageID, authorID, content)
    }
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
    if (isServerMode()) {
      await apiChat.deleteMessage(messageID, isManager)
    } else {
      await App.DeleteMessage(messageID, actorID, isManager)
    }
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
    let results
    if (isServerMode()) {
      results = await apiChat.searchMessages(channelID, query, 20)
    } else {
      results = await App.SearchMessages(channelID, query, 20)
    }
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

// File operations — local only, no REST equivalent
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

// --- Message Polling ---

async function pollNewMessages(channelID: string) {
  if (!channelID || activeChannelId !== channelID) return

  try {
    await ensureValidToken()
    const lastMsg = messages[messages.length - 1]
    const after = lastMsg?.id ?? ''
    let result
    if (isServerMode()) {
      result = await apiChat.getMessages(channelID, '', after, 50)
    } else {
      result = await withTimeout(
        App.GetMessages(channelID, '', after, 50),
        WAILS_STORE_TIMEOUT_MS,
        'GetMessages timeout',
      )
    }
    const newMsgs = (result ?? []) as unknown as MessageData[]
    if (newMsgs.length > 0) {
      // API returns newest first — reverse for chronological order
      const reversed = newMsgs.reverse()
      // Deduplicate: only add messages not already in state
      const existingIds = new Set(messages.map(m => m.id))
      const fresh = reversed.filter(m => !existingIds.has(m.id))
      if (fresh.length > 0) {
        messages = [...messages, ...fresh]
        // User is viewing this channel — keep lastRead up to date
        const newest = fresh[fresh.length - 1]
        if (newest) {
          lastReadMap[channelID] = newest.id
          saveLastRead()
        }
      }
    }
  } catch {
    // Silently ignore polling errors
  }
}

function startMessagePolling(channelID: string) {
  stopMessagePolling()
  messagePollTimer = setInterval(() => pollNewMessages(channelID), MESSAGE_POLL_INTERVAL)
}

export function stopMessagePolling() {
  if (messagePollTimer) {
    clearInterval(messagePollTimer)
    messagePollTimer = null
  }
}

export function resetChat(): void {
  stopMessagePolling()
  stopUnreadPolling()
  messages = []
  activeChannelId = null
  hasMore = true
  searchResults = []
  searchQuery = ''
  error = null
  attachmentsByMessage = {}
}

// --- Unread Polling ---

export function markChannelAsRead(channelId: string): void {
  const lastMsg = activeChannelId === channelId
    ? messages[messages.length - 1]
    : null
  if (lastMsg) {
    lastReadMap[channelId] = lastMsg.id
  }
  // Clear unread count immediately for responsiveness
  if (unreadCounts[channelId]) {
    const updated = { ...unreadCounts }
    delete updated[channelId]
    unreadCounts = updated
  }
  saveLastRead()
}

async function pollUnreadCounts(): Promise<void> {
  if (trackedChannelIds.length === 0) return
  try {
    await ensureValidToken()
    // Build the map of channelId → lastReadMessageId
    const req: Record<string, string> = {}
    for (const chId of trackedChannelIds) {
      req[chId] = lastReadMap[chId] ?? ''
    }
    let result: Record<string, number>
    if (isServerMode()) {
      result = await apiChat.getUnreadCounts(req) as Record<string, number>
    } else {
      result = await withTimeout(
        App.GetUnreadCounts(req),
        WAILS_STORE_TIMEOUT_MS,
        'GetUnreadCounts timeout',
      ) as unknown as Record<string, number>
    }
    // Don't show unread for the active channel (user is viewing it)
    if (activeChannelId && result[activeChannelId]) {
      delete result[activeChannelId]
    }
    unreadCounts = result ?? {}
  } catch {
    // Silently ignore polling errors
  }
}

export function startUnreadPolling(channelIds: string[]): void {
  stopUnreadPolling()
  trackedChannelIds = channelIds
  lastReadMap = loadLastRead()
  // Initial fetch
  void pollUnreadCounts()
  unreadPollTimer = setInterval(() => pollUnreadCounts(), UNREAD_POLL_INTERVAL)
}

export function stopUnreadPolling(): void {
  if (unreadPollTimer) {
    clearInterval(unreadPollTimer)
    unreadPollTimer = null
  }
  trackedChannelIds = []
}
