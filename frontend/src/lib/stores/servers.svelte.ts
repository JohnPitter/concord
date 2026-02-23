// Server store using Svelte 5 runes
// Manages server, channel, and member state — Wails bindings (P2P) or HTTP API (Server)

import * as App from '../../../wailsjs/go/main/App'
import { ensureValidToken } from './auth.svelte'
import { isServerMode } from '../api/mode'
import { apiServers } from '../api/servers'

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

const MEMBER_POLL_INTERVAL = 15_000 // 15s polling for members

let servers = $state<ServerData[]>([])
let activeServerId = $state<string | null>(null)
let channels = $state<ChannelData[]>([])
let members = $state<MemberData[]>([])
let loading = $state(false)
let error = $state<string | null>(null)
let memberPollTimer: ReturnType<typeof setInterval> | null = null

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
    await ensureValidToken()
    let result
    if (isServerMode()) {
      result = await apiServers.list()
    } else {
      result = await App.ListUserServers(userID)
    }
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
  startMemberPolling(serverID)
}

export async function createServer(name: string, ownerID: string): Promise<ServerData | null> {
  error = null
  try {
    await ensureValidToken()
    let srv
    if (isServerMode()) {
      srv = await apiServers.create(name)
    } else {
      srv = await App.CreateServer(name, ownerID)
    }
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
    await ensureValidToken()
    if (isServerMode()) {
      await apiServers.update(serverID, name, iconURL)
    } else {
      await App.UpdateServer(serverID, userID, name, iconURL)
    }
    servers = servers.map(s => s.id === serverID ? { ...s, name, icon_url: iconURL } : s)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to update server'
  }
}

export async function deleteServer(serverID: string, userID: string): Promise<void> {
  error = null
  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiServers.delete(serverID)
    } else {
      await App.DeleteServer(serverID, userID)
    }
    servers = servers.filter(s => s.id !== serverID)
    if (activeServerId === serverID) {
      stopMemberPolling()
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
    await ensureValidToken()
    let result
    if (isServerMode()) {
      result = await apiServers.listChannels(serverID)
    } else {
      result = await App.ListChannels(serverID)
    }
    channels = (result ?? []) as unknown as ChannelData[]
  } catch (e) {
    console.error('Failed to load channels:', e)
    channels = []
  }
}

export async function createChannel(serverID: string, userID: string, name: string, type: 'text' | 'voice'): Promise<ChannelData | null> {
  error = null
  try {
    await ensureValidToken()
    let ch
    if (isServerMode()) {
      ch = await apiServers.createChannel(serverID, name, type)
    } else {
      ch = await App.CreateChannel(serverID, userID, name, type)
    }
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
    await ensureValidToken()
    // No REST equivalent for deleteChannel yet — use Wails binding
    await App.DeleteChannel(serverID, userID, channelID)
    channels = channels.filter(c => c.id !== channelID)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to delete channel'
  }
}

// --- Members ---

async function loadMembers(serverID: string): Promise<void> {
  try {
    await ensureValidToken()
    let result
    if (isServerMode()) {
      result = await apiServers.listMembers(serverID)
    } else {
      result = await App.ListMembers(serverID)
    }
    members = (result ?? []) as unknown as MemberData[]
  } catch (e) {
    console.error('Failed to load members:', e)
    members = []
  }
}

export async function kickMember(serverID: string, actorID: string, targetID: string): Promise<void> {
  error = null
  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiServers.kickMember(serverID, targetID)
    } else {
      await App.KickMember(serverID, actorID, targetID)
    }
    members = members.filter(m => m.user_id !== targetID)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to kick member'
  }
}

export async function updateMemberRole(serverID: string, actorID: string, targetID: string, role: string): Promise<void> {
  error = null
  try {
    await ensureValidToken()
    if (isServerMode()) {
      await apiServers.updateMemberRole(serverID, targetID, role)
    } else {
      await App.UpdateMemberRole(serverID, actorID, targetID, role)
    }
    members = members.map(m => m.user_id === targetID ? { ...m, role: role as MemberData['role'] } : m)
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to update role'
  }
}

// --- Invites ---

export async function generateInvite(serverID: string, userID: string): Promise<string | null> {
  error = null
  try {
    await ensureValidToken()
    let code: string
    if (isServerMode()) {
      const result = await apiServers.generateInvite(serverID)
      code = (result as { invite_code: string }).invite_code
    } else {
      code = await App.GenerateInvite(serverID, userID)
    }
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
    await ensureValidToken()
    let srv
    if (isServerMode()) {
      srv = await apiServers.redeemInvite(code)
    } else {
      srv = await App.RedeemInvite(code, userID)
    }
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
    // getInviteInfo stays local — no REST equivalent
    return await App.GetInviteInfo(code) as unknown as InviteInfoData
  } catch {
    return null
  }
}

// --- Member Polling ---

function startMemberPolling(serverID: string) {
  stopMemberPolling()
  memberPollTimer = setInterval(async () => {
    try {
      await loadMembers(serverID)
    } catch {
      // Silently ignore polling errors
    }
  }, MEMBER_POLL_INTERVAL)
}

export function stopMemberPolling() {
  if (memberPollTimer) {
    clearInterval(memberPollTimer)
    memberPollTimer = null
  }
}

export function clearServerError(): void {
  error = null
}
