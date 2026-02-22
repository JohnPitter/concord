// Servers/Channels/Members/Invites API for central server communication

import { apiClient } from './client'

export const apiServers = {
  list: () =>
    apiClient.get<unknown[]>('/api/v1/servers'),

  create: (name: string) =>
    apiClient.post('/api/v1/servers', { name }),

  update: (id: string, name: string, iconUrl: string) =>
    apiClient.put(`/api/v1/servers/${encodeURIComponent(id)}`, { name, icon_url: iconUrl }),

  delete: (id: string) =>
    apiClient.del(`/api/v1/servers/${encodeURIComponent(id)}`),

  // Channels
  listChannels: (serverId: string) =>
    apiClient.get<unknown[]>(`/api/v1/servers/${encodeURIComponent(serverId)}/channels`),

  createChannel: (serverId: string, name: string, type: string) =>
    apiClient.post(`/api/v1/servers/${encodeURIComponent(serverId)}/channels`, { name, type }),

  // Members
  listMembers: (serverId: string) =>
    apiClient.get<unknown[]>(`/api/v1/servers/${encodeURIComponent(serverId)}/members`),

  kickMember: (serverId: string, userId: string) =>
    apiClient.del(`/api/v1/servers/${encodeURIComponent(serverId)}/members/${encodeURIComponent(userId)}`),

  updateMemberRole: (serverId: string, userId: string, role: string) =>
    apiClient.put(`/api/v1/servers/${encodeURIComponent(serverId)}/members/${encodeURIComponent(userId)}/role`, { role }),

  // Invites
  generateInvite: (serverId: string) =>
    apiClient.post<{ invite_code: string }>(`/api/v1/servers/${encodeURIComponent(serverId)}/invite`),

  redeemInvite: (code: string) =>
    apiClient.post(`/api/v1/invite/${encodeURIComponent(code)}/redeem`),
}
