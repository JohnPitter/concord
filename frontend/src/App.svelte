<script lang="ts">
  import { getAuth, initAuth, logout } from './lib/stores/auth.svelte'
  import {
    getServers, loadUserServers, selectServer,
    createServer, redeemInvite, generateInvite,
  } from './lib/stores/servers.svelte'
  import {
    getChat, loadMessages, loadOlderMessages,
    sendMessage, editMessage, deleteMessage, resetChat,
  } from './lib/stores/chat.svelte'
  import Login from './lib/components/auth/Login.svelte'
  import CreateServerModal from './lib/components/server/CreateServer.svelte'
  import JoinServerModal from './lib/components/server/JoinServer.svelte'
  import ServerSidebar from './lib/components/layout/ServerSidebar.svelte'
  import ChannelSidebar from './lib/components/layout/ChannelSidebar.svelte'
  import MainContent from './lib/components/layout/MainContent.svelte'
  import MemberSidebar from './lib/components/layout/MemberSidebar.svelte'

  const auth = getAuth()
  const srv = getServers()
  const chat = getChat()

  let showCreateServer = $state(false)
  let showJoinServer = $state(false)
  let activeChannelId = $state<string | null>(null)

  // Initialize auth + load servers on mount
  $effect(() => {
    initAuth()
  })

  // Load servers when authenticated
  $effect(() => {
    if (auth.authenticated && auth.user) {
      loadUserServers(auth.user.id)
    }
  })

  // Auto-select first server when servers load
  $effect(() => {
    if (srv.list.length > 0 && !srv.activeId) {
      selectServer(srv.list[0].id)
    }
  })

  // Auto-select first text channel when channels change
  $effect(() => {
    if (srv.textChannels.length > 0 && !activeChannelId) {
      activeChannelId = srv.textChannels[0].id
    }
  })

  // Load messages when active channel changes
  $effect(() => {
    if (activeChannelId) {
      loadMessages(activeChannelId)
    } else {
      resetChat()
    }
  })

  const activeChannel = $derived(srv.channels.find(c => c.id === activeChannelId))

  // Map server data for ServerSidebar format
  const sidebarServers = $derived(
    srv.list.map(s => ({
      id: s.id,
      name: s.name,
      iconUrl: s.icon_url || undefined,
    }))
  )

  // Map channel data for ChannelSidebar format
  const sidebarChannels = $derived(
    srv.channels.map(c => ({
      id: c.id,
      name: c.name,
      type: c.type as 'text' | 'voice',
    }))
  )

  // Map member data for MemberSidebar format
  const sidebarMembers = $derived(
    srv.members.map(m => ({
      id: m.user_id,
      name: m.username,
      status: 'online' as const, // TODO: real presence in Phase 5
      role: m.role === 'owner' ? 'Owner' : m.role === 'admin' ? 'Admin' : m.role === 'moderator' ? 'Moderator' : undefined,
    }))
  )

  async function handleCreateServer(name: string) {
    if (!auth.user) return
    const created = await createServer(name, auth.user.id)
    if (created) {
      await selectServer(created.id)
    }
  }

  async function handleJoinServer(code: string) {
    if (!auth.user) return
    const joined = await redeemInvite(code, auth.user.id)
    if (joined) {
      await loadUserServers(auth.user.id)
      await selectServer(joined.id)
    }
  }

  async function handleSendMessage(content: string) {
    if (!auth.user || !activeChannelId) return
    await sendMessage(activeChannelId, auth.user.id, content)
  }

  async function handleDeleteMessage(messageId: string) {
    if (!auth.user) return
    const member = srv.members.find(m => m.user_id === auth.user!.id)
    const isManager = member?.role === 'owner' || member?.role === 'admin' || member?.role === 'moderator'
    await deleteMessage(messageId, auth.user.id, isManager)
  }
</script>

{#if auth.loading}
  <!-- Loading splash -->
  <div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
    <div class="text-center">
      <div class="mx-auto mb-4 h-10 w-10 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
      <p class="text-sm text-void-text-muted">Loading Concord...</p>
    </div>
  </div>
{:else if !auth.authenticated}
  <Login />
{:else}
  <div class="flex h-screen w-screen overflow-hidden">
    <ServerSidebar
      servers={sidebarServers}
      activeServerId={srv.activeId ?? ''}
      onSelectServer={(id) => {
        selectServer(id)
        activeChannelId = null
      }}
      onAddServer={() => showCreateServer = true}
    />
    <ChannelSidebar
      serverName={srv.active?.name ?? 'Server'}
      channels={sidebarChannels}
      activeChannelId={activeChannelId ?? ''}
      onSelectChannel={(id) => activeChannelId = id}
    />
    <MainContent
      channelName={activeChannel?.name ?? 'general'}
      messages={chat.messages}
      currentUserId={auth.user?.id ?? ''}
      loading={chat.loading}
      hasMore={chat.hasMore}
      sending={chat.sending}
      onSend={handleSendMessage}
      onLoadMore={loadOlderMessages}
      onDelete={handleDeleteMessage}
    />
    <MemberSidebar members={sidebarMembers} />
  </div>

  <CreateServerModal
    bind:open={showCreateServer}
    onCreate={handleCreateServer}
  />

  <JoinServerModal
    bind:open={showJoinServer}
    onJoin={handleJoinServer}
  />
{/if}
