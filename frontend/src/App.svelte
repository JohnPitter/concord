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
  import { getSettings, loadSettings, setNetworkMode, setP2PProfile } from './lib/stores/settings.svelte'
  import {
    getFriends, loadFriends, setFriendsTab, openDM,
  } from './lib/stores/friends.svelte'

  import Login from './lib/components/auth/Login.svelte'
  import ModeSelector from './lib/components/auth/ModeSelector.svelte'
  import P2PProfile from './lib/components/auth/P2PProfile.svelte'
  import P2PApp from './lib/components/p2p/P2PApp.svelte'
  import CreateServerModal from './lib/components/server/CreateServer.svelte'
  import JoinServerModal from './lib/components/server/JoinServer.svelte'
  import ServerSidebar from './lib/components/layout/ServerSidebar.svelte'
  import ChannelSidebar from './lib/components/layout/ChannelSidebar.svelte'
  import DMSidebar from './lib/components/layout/DMSidebar.svelte'
  import MainContent from './lib/components/layout/MainContent.svelte'
  import MemberSidebar from './lib/components/layout/MemberSidebar.svelte'
  import FriendsList from './lib/components/layout/FriendsList.svelte'
  import ActiveNow from './lib/components/layout/ActiveNow.svelte'
  import SettingsPanel from './lib/components/settings/SettingsPanel.svelte'
  import Toast from './lib/components/ui/Toast.svelte'

  const auth = getAuth()
  const srv = getServers()
  const chat = getChat()
  const vc = getVoice()
  const friends = getFriends()

  const settings = getSettings()
  const networkMode = $derived(settings.networkMode)
  const p2pProfile = $derived(settings.p2pProfile)
  const needsModeSelection = $derived(networkMode === null)
  const needsP2PProfile = $derived(networkMode === 'p2p' && !p2pProfile)
  const isP2PMode = $derived(networkMode === 'p2p' && !!p2pProfile)

  // 'home' = DMs/friends view, any other string = server id
  let activeServerId = $state<string>('home')

  let showCreateServer = $state(false)
  let showJoinServer = $state(false)
  let showSettings = $state(false)
  let showMembers = $state(false)
  let activeChannelId = $state<string | null>(null)

  const isHome = $derived(activeServerId === 'home')

  // Load persisted settings on mount
  $effect(() => { loadSettings() })

  // Initialize auth on mount
  $effect(() => { initAuth() })

  // Load data once authenticated
  $effect(() => {
    if (auth.authenticated && auth.user) {
      loadUserServers(auth.user.id)
      loadFriends()
    }
  })

  // Auto-select first server when in server mode and servers load
  $effect(() => {
    if (!isHome && srv.list.length > 0 && !srv.activeId) {
      selectServer(srv.list[0].id)
    }
  })

  // Auto-select first text channel when channels change
  $effect(() => {
    if (!isHome && srv.textChannels.length > 0 && !activeChannelId) {
      activeChannelId = srv.textChannels[0].id
    }
  })

  // Load messages when active channel changes
  $effect(() => {
    if (!isHome && activeChannelId) {
      loadMessages(activeChannelId)
    } else if (!activeChannelId) {
      resetChat()
    }
  })

  function handleSelectServer(id: string) {
    if (id === 'home') {
      activeServerId = 'home'
      activeChannelId = null
      resetChat()
      return
    }
    activeServerId = id
    activeChannelId = null
    selectServer(id)
  }

  function handleModeSelect(mode: 'p2p' | 'server') {
    setNetworkMode(mode)
  }

  function handleP2PProfileConfirm(profile: { displayName: string; avatarDataUrl?: string }) {
    setP2PProfile(profile)
  }

  const activeChannel = $derived(srv.channels.find(c => c.id === activeChannelId))
  const voiceChannelName = $derived(
    srv.channels.find(c => c.id === vc.channelId)?.name ?? ''
  )

  const sidebarServers = $derived(
    srv.list.map(s => ({
      id: s.id,
      name: s.name,
      iconUrl: s.icon_url || undefined,
    }))
  )

  const sidebarChannels = $derived(
    srv.channels.map(c => ({
      id: c.id,
      name: c.name,
      type: c.type as 'text' | 'voice',
    }))
  )

  const sidebarMembers = $derived(
    srv.members.map(m => ({
      id: m.user_id,
      name: m.username,
      avatarUrl: m.avatar_url || undefined,
      status: 'online' as const,
      role: m.role === 'owner' ? 'Owner' : m.role === 'admin' ? 'Admin' : m.role === 'moderator' ? 'Moderator' : undefined,
    }))
  )

  async function handleCreateServer(name: string) {
    if (!auth.user) return
    const created = await createServer(name, auth.user.id)
    if (created) {
      activeServerId = created.id
      await selectServer(created.id)
    }
  }

  async function handleJoinServer(code: string) {
    if (!auth.user) return
    const joined = await redeemInvite(code, auth.user.id)
    if (joined) {
      await loadUserServers(auth.user.id)
      activeServerId = joined.id
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

  $effect(() => {
    for (const msg of chat.messages) {
      if (!chat.attachmentsByMessage[msg.id]) {
        loadAttachments(msg.id)
      }
    }
  })

  async function handleFileSelect(file: { name: string; data: number[] }) {
    if (!auth.user || !activeChannelId) return
    const msg = await sendMessage(activeChannelId, auth.user.id, `[file: ${file.name}]`)
    if (msg) {
      await uploadFile(msg.id, file.name, file.data)
    }
  }

  async function handleDownloadFile(attachmentId: string) {
    const data = await downloadFile(attachmentId)
    if (!data) return
    let filename = 'download'
    for (const atts of Object.values(chat.attachmentsByMessage)) {
      const att = atts.find(a => a.id === attachmentId)
      if (att) { filename = att.filename; break }
    }
    const blob = new Blob([new Uint8Array(data)])
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = filename
    document.body.appendChild(a); a.click()
    document.body.removeChild(a); URL.revokeObjectURL(url)
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

  const currentUserProp = $derived(
    auth.user
      ? { username: auth.user.username, display_name: auth.user.display_name, avatar_url: auth.user.avatar_url }
      : null
  )
</script>

{#if needsModeSelection}
  <ModeSelector onSelectMode={handleModeSelect} />

{:else if needsP2PProfile}
  <P2PProfile onConfirm={handleP2PProfileConfirm} />

{:else if isP2PMode}
  <P2PApp profile={settings.p2pProfile} />

{:else if auth.loading}
  <div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
    <div class="text-center">
      <div class="mx-auto mb-4 h-10 w-10 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
      <p class="text-sm text-void-text-muted">Carregando Concord...</p>
    </div>
  </div>

{:else if !auth.authenticated}
  <Login />

{:else}
  <div class="flex h-screen w-screen overflow-hidden">

    <!-- Col 1: Server rail (72px) -->
    <ServerSidebar
      servers={sidebarServers}
      activeServerId={activeServerId}
      onSelectServer={handleSelectServer}
      onAddServer={() => showCreateServer = true}
      currentUser={currentUserProp}
    />

    {#if isHome}
      <!-- ══ HOME VIEW (DMs + Friends + Active Now) ══════════════════════ -->

      <!-- Col 2: DM sidebar (240px) -->
      <DMSidebar
        dms={friends.dms}
        activeDMId={friends.activeDMId}
        onSelectDM={(id) => openDM(id)}
        onOpenFriends={() => openDM(null)}
        currentUser={currentUserProp}
        voiceConnected={vc.connected}
        voiceChannelName={voiceChannelName}
        voiceMuted={vc.muted}
        voiceDeafened={vc.deafened}
        voiceSpeakers={vc.speakers}
        onToggleMute={toggleMute}
        onToggleDeafen={toggleDeafen}
        onLeaveVoice={leaveVoice}
        onOpenSettings={() => showSettings = true}
      />

      <!-- Col 3: Friends list (flex-1) -->
      <FriendsList
        friends={friends.friends}
        tab={friends.tab}
        onTabChange={(t) => setFriendsTab(t)}
      />

      <!-- Col 4: Active Now (340px) -->
      <ActiveNow friends={friends.friends} />

    {:else}
      <!-- ══ SERVER VIEW (Channels + Chat + Members) ═════════════════════ -->

      <!-- Col 2: Channel sidebar (240px) -->
      <ChannelSidebar
        serverName={srv.active?.name ?? 'Servidor'}
        channels={sidebarChannels}
        activeChannelId={activeChannelId ?? ''}
        onSelectChannel={(id) => activeChannelId = id}
        currentUser={currentUserProp}
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

      <!-- Col 3: Main chat (flex-1) -->
      <MainContent
        channelName={activeChannel?.name ?? 'geral'}
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

      <!-- Col 4: Member sidebar (240px, toggle) -->
      {#if showMembers}
        <MemberSidebar members={sidebarMembers} />
      {/if}
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
    currentUser={currentUserProp}
    onLogout={() => { logout() }}
  />
{/if}

<Toast />
