// Messages API for central server communication

import { apiClient } from './client'

export const apiChat = {
  getMessages: (channelId: string, before: string, after: string, limit: number) => {
    const params = new URLSearchParams()
    if (before) params.set('before', before)
    if (after) params.set('after', after)
    if (limit) params.set('limit', String(limit))
    const qs = params.toString()
    return apiClient.get<unknown[]>(
      `/api/v1/channels/${encodeURIComponent(channelId)}/messages${qs ? '?' + qs : ''}`
    )
  },

  sendMessage: (channelId: string, content: string) =>
    apiClient.post(
      `/api/v1/channels/${encodeURIComponent(channelId)}/messages`,
      { content }
    ),

  editMessage: (messageId: string, content: string) =>
    apiClient.put(
      `/api/v1/messages/${encodeURIComponent(messageId)}`,
      { content }
    ),

  deleteMessage: (messageId: string, isManager: boolean) =>
    apiClient.del(
      `/api/v1/messages/${encodeURIComponent(messageId)}${isManager ? '?is_manager=true' : ''}`
    ),

  searchMessages: (channelId: string, query: string, limit: number) =>
    apiClient.get<unknown[]>(
      `/api/v1/channels/${encodeURIComponent(channelId)}/messages/search?q=${encodeURIComponent(query)}&limit=${limit}`
    ),
}
