<script lang="ts">
  import { getAuth, initAuth, logout } from './lib/stores/auth.svelte'
  import {
    getServers, loadUserServers, selectServer,
    createServer, redeemInvite, generateInvite,
  } from './lib/stores/servers.svelte'
  import {
    getChat, loadMessages, loadOlderMessages,
    sendMessage, editMessage, deleteMessage, resetChat,
    uploadFile, downloadFile, deleteAttachment, loadAttachments,
  } from './lib/stores/chat.svelte'
  import {
    getVoice, joinVoice, leaveVoice,
    toggleMute, toggleDeafen, resetVoice,
  } from './lib/stores/voice.svelte'
  import { loadSettings } from './lib/stores/settings.svelte'
  import Login from './lib/components/auth/Login.svelte'
  import CreateServerModal from './lib/components/server/CreateServer.svelte'
  import JoinServerModal from './lib/components/server/JoinServer.svelte'
  import ServerSidebar from './lib/components/layout/ServerSidebar.svelte'
  import ChannelSidebar from './lib/components/layout/ChannelSidebar.svelte'
  import MainContent from './lib/components/layout/MainContent.svelte'
  import MemberSidebar from './lib/components/layout/MemberSidebar.svelte'
  import SettingsPanel from './lib/components/settings/SettingsPanel.svelte'
  import Toast from './lib/components/ui/Toast.svelte'

  const auth = getAuth()
  const srv = getServers()
  const chat = getChat()
  const vc = getVoice()

  let showCreateServer = $state(false)
  let showJoinServer = $state(false)
  let showSettings = $state(false)
  let showMembers = $state(false)
  let activeChannelId = $state<string | null>(null)

  // Load persisted settings
  $effect(() => {
    loadSettings()
  })

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
  const voiceChannelName = $derived(
    srv.channels.find(c => c.id === vc.channelId)?.name ?? ''
  )

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
      avatarUrl: m.avatar_url || undefined,
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

  // Load attachments for each message when messages change
  $effect(() => {
    for (const msg of chat.messages) {
      if (!chat.attachmentsByMessage[msg.id]) {
        loadAttachments(msg.id)
      }
    }
  })

  async function handleFileSelect(file: { name: string; data: number[] }) {
    if (!auth.user || !activeChannelId) return
    // Send a placeholder message first, then upload file to it
    const msg = await sendMessage(activeChannelId, auth.user.id, `[file: ${file.name}]`)
    if (msg) {
      await uploadFile(msg.id, file.name, file.data)
    }
  }

  async function handleDownloadFile(attachmentId: string) {
    const data = await downloadFile(attachmentId)
    if (!data) return

    // Find the attachment info for filename
    let filename = 'download'
    for (const atts of Object.values(chat.attachmentsByMessage)) {
      const att = atts.find(a => a.id === attachmentId)
      if (att) {
        filename = att.filename
        break
      }
    }

    // Trigger browser download
    const blob = new Blob([new Uint8Array(data)])
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  async function handleDeleteFile(attachmentId: string) {
    await deleteAttachment(attachmentId)
  }

  async function handleJoinVoice(channelId: string) {
    if (vc.channelId === channelId) {
      await leaveVoice()
    } else {
      if (vc.connected) await leaveVoice()
      await joinVoice(channelId)
    }
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
      currentUser={auth.user ? { username: auth.user.username, display_name: auth.user.display_name, avatar_url: auth.user.avatar_url } : null}
    />
    <ChannelSidebar
      serverName={srv.active?.name ?? 'Server'}
      channels={sidebarChannels}
      activeChannelId={activeChannelId ?? ''}
      onSelectChannel={(id) => activeChannelId = id}
      currentUser={auth.user ? { username: auth.user.username, display_name: auth.user.display_name, avatar_url: auth.user.avatar_url } : null}
      voiceConnected={vc.connected}
      voiceChannelName={voiceChannelName}
      voiceMuted={vc.muted}
      voiceDeafened={vc.deafened}
      voiceSpeakers={vc.speakers}
      voiceChannelId={vc.channelId}
      onJoinVoice={handleJoinVoice}
      onLeaveVoice={leaveVoice}
      onToggleMute={toggleMute}
      onToggleDeafen={toggleDeafen}
      onOpenSettings={() => showSettings = true}
    />
    <MainContent
      channelName={activeChannel?.name ?? 'general'}
      messages={chat.messages}
      currentUserId={auth.user?.id ?? ''}
      loading={chat.loading}
      hasMore={chat.hasMore}
      sending={chat.sending}
      attachmentsByMessage={chat.attachmentsByMessage}
      membersVisible={showMembers}
      onSend={handleSendMessage}
      onLoadMore={loadOlderMessages}
      onDelete={handleDeleteMessage}
      onFileSelect={handleFileSelect}
      onDownloadFile={handleDownloadFile}
      onDeleteFile={handleDeleteFile}
      onToggleMembers={() => showMembers = !showMembers}
    />
    {#if showMembers}
      <MemberSidebar members={sidebarMembers} />
    {/if}
  </div>

  <CreateServerModal
    bind:open={showCreateServer}
    onCreate={handleCreateServer}
  />

  <JoinServerModal
    bind:open={showJoinServer}
    onJoin={handleJoinServer}
  />

  <SettingsPanel
    bind:open={showSettings}
    currentUser={auth.user ? { username: auth.user.username, display_name: auth.user.display_name, avatar_url: auth.user.avatar_url } : null}
    onLogout={() => { logout() }}
  />
{/if}

<Toast />
