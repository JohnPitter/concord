// Friends API client for central server communication

import { apiClient } from './client'

export interface FriendRequestView {
  id: string
  username: string
  display_name: string
  avatar_url: string
  direction: 'incoming' | 'outgoing'
  createdAt: string
}

export interface FriendView {
  id: string
  username: string
  display_name: string
  avatar_url: string
  status: 'online' | 'idle' | 'dnd' | 'offline' | string
  activity?: string
  game?: string
  gameSince?: string
  streaming?: boolean
  streamTitle?: string
}

export interface DirectMessageView {
  id: string
  sender_id: string
  receiver_id: string
  content: string
  created_at: string
}

export const apiFriends = {
  sendRequest: (username: string) =>
    apiClient.request<void>('POST', '/api/v1/friends/request', { username }),

  getPendingRequests: () =>
    apiClient.get<FriendRequestView[]>('/api/v1/friends/requests'),

  acceptRequest: (requestId: string) =>
    apiClient.request<void>('PUT', `/api/v1/friends/requests/${encodeURIComponent(requestId)}/accept`),

  rejectRequest: (requestId: string) =>
    apiClient.del(`/api/v1/friends/requests/${encodeURIComponent(requestId)}`),

  getFriends: () =>
    apiClient.get<FriendView[]>('/api/v1/friends'),

  removeFriend: (friendId: string) =>
    apiClient.del(`/api/v1/friends/${encodeURIComponent(friendId)}`),

  blockUser: (friendId: string) =>
    apiClient.post(`/api/v1/friends/${encodeURIComponent(friendId)}/block`),

  unblockUser: (friendId: string) =>
    apiClient.del(`/api/v1/friends/${encodeURIComponent(friendId)}/block`),

  getDirectMessages: (friendId: string, after = '', limit = 100) => {
    const qs = new URLSearchParams()
    if (after) qs.set('after', after)
    if (limit > 0) qs.set('limit', String(limit))
    return apiClient.get<DirectMessageView[]>(
      `/api/v1/friends/${encodeURIComponent(friendId)}/messages${qs.toString() ? `?${qs.toString()}` : ''}`
    )
  },

  sendDirectMessage: (friendId: string, content: string) =>
    apiClient.request<DirectMessageView>('POST', `/api/v1/friends/${encodeURIComponent(friendId)}/messages`, { content }),
}
