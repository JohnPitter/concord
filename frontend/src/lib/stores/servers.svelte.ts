// Server store using Svelte 5 runes
// Manages server, channel, and member state via Wails bindings

import * as App from '../../../wailsjs/go/main/App'

export interface ServerData {
  id: string
  name: string
  icon_url: string
  owner_id: string
  invite_code: string
  created_at: string
}

export interface ChannelData {
  id: string
  server_id: string
  name: string
  type: 'text' | 'voice'
  position: number
  created_at: string
}

export interface MemberData {
  server_id: string
  user_id: string
  username: string
  avatar_url: string
  role: 'owner' | 'admin' | 'moderator' | 'member'
  joined_at: string
}

export interface InviteInfoData {
  server_id: string
  server_name: string
  invite_code: string
  member_count: number
}

let servers = $state<ServerData[]>([])
let activeServerId = $state<string | null>(null)
let channels = $state<ChannelData[]>([])
let members = $state<MemberData[]>([])
let loading = $state(false)
let error = $state<string | null>(null)

export function getServers() {
  return {
    get list() { return servers },
    get activeId() { return activeServerId },
    get active() { return servers.find(s => s.id === activeServerId) ?? null },
    get channels() { return channels },
    get textChannels() { return channels.filter(c => c.type === 'text') },
    get voiceChannels() { return channels.filter(c => c.type === 'voice') },
    get members() { return members },
    get loading() { return loading },
    get error() { return error },
  }
}

export async function loadUserServers(userID: string): Promise<void> {
  loading = true
  error = null
  try {
    const result = await App.ListUserServers(userID)
    servers = (result ?? []) as unknown as ServerData[]
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load servers'
  } finally {
    loading = false
  }
}

export async function selectServer(serverID: string): Promise<void> {
  activeServerId = serverID
  await Promise.all([loadChannels(serverID), loadMembers(serverID)])
}

export async function createServer(name: string, ownerID: string): Promise<ServerData | null> {
  error = null
  try {
    const srv = await App.CreateServer(name, ownerID)
    const data = srv as unknown as ServerData
    servers = [...servers, data]
    return data
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to create server'
    return null
  }
}

export async function updateServer(serverID: string, userID: string, name: string, iconURL: string): Promise<void> {
  error = null
  try {
    await App.UpdateServer(serverID, userID, name, iconURL)
    servers = servers.map(s => s.id === serverID ? { ...s, name, icon_url: iconURL } : s)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to update server'
  }
}

export async function deleteServer(serverID: string, userID: string): Promise<void> {
  error = null
  try {
    await App.DeleteServer(serverID, userID)
    servers = servers.filter(s => s.id !== serverID)
    if (activeServerId === serverID) {
      activeServerId = servers[0]?.id ?? null
      if (activeServerId) await selectServer(activeServerId)
    }
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to delete server'
  }
}

// --- Channels ---

async function loadChannels(serverID: string): Promise<void> {
  try {
    const result = await App.ListChannels(serverID)
    channels = (result ?? []) as unknown as ChannelData[]
  } catch (e) {
    console.error('Failed to load channels:', e)
    channels = []
  }
}

export async function createChannel(serverID: string, userID: string, name: string, type: 'text' | 'voice'): Promise<ChannelData | null> {
  error = null
  try {
    const ch = await App.CreateChannel(serverID, userID, name, type)
    const data = ch as unknown as ChannelData
    channels = [...channels, data]
    return data
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to create channel'
    return null
  }
}

export async function deleteChannel(serverID: string, userID: string, channelID: string): Promise<void> {
  error = null
  try {
    await App.DeleteChannel(serverID, userID, channelID)
    channels = channels.filter(c => c.id !== channelID)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to delete channel'
  }
}

// --- Members ---

async function loadMembers(serverID: string): Promise<void> {
  try {
    const result = await App.ListMembers(serverID)
    members = (result ?? []) as unknown as MemberData[]
  } catch (e) {
    console.error('Failed to load members:', e)
    members = []
  }
}

export async function kickMember(serverID: string, actorID: string, targetID: string): Promise<void> {
  error = null
  try {
    await App.KickMember(serverID, actorID, targetID)
    members = members.filter(m => m.user_id !== targetID)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to kick member'
  }
}

export async function updateMemberRole(serverID: string, actorID: string, targetID: string, role: string): Promise<void> {
  error = null
  try {
    await App.UpdateMemberRole(serverID, actorID, targetID, role)
    members = members.map(m => m.user_id === targetID ? { ...m, role: role as MemberData['role'] } : m)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to update role'
  }
}

// --- Invites ---

export async function generateInvite(serverID: string, userID: string): Promise<string | null> {
  error = null
  try {
    const code: string = await App.GenerateInvite(serverID, userID)
    // Update local server's invite code
    servers = servers.map(s => s.id === serverID ? { ...s, invite_code: code } : s)
    return code
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to generate invite'
    return null
  }
}

export async function redeemInvite(code: string, userID: string): Promise<ServerData | null> {
  error = null
  try {
    const srv = await App.RedeemInvite(code, userID)
    const data = srv as unknown as ServerData
    if (!servers.find(s => s.id === data.id)) {
      servers = [...servers, data]
    }
    return data
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to join server'
    return null
  }
}

export async function getInviteInfo(code: string): Promise<InviteInfoData | null> {
  try {
    return await App.GetInviteInfo(code) as unknown as InviteInfoData
  } catch {
    return null
  }
}

export function clearServerError(): void {
  error = null
}
