<script lang="ts">
  import { getAuth, initAuth, logout } from './lib/stores/auth.svelte'
  import {
    getServers, loadUserServers, selectServer,
    createServer, redeemInvite, generateInvite,
    createChannel, deleteChannel, updateServer, deleteServer,
    updateMemberRole, kickMember,
  } from './lib/stores/servers.svelte'
  import {
    getChat, loadMessages, loadOlderMessages,
    sendMessage, editMessage, deleteMessage, resetChat,
    uploadFile, downloadFile, deleteAttachment, loadAttachments,
    searchMessages, clearSearch,
  } from './lib/stores/chat.svelte'
  import {
    getVoice, joinVoice, leaveVoice,
    toggleMute, toggleDeafen, resetVoice,
    toggleNoiseSuppression, toggleScreenSharing,
    setLocalUsername,
  } from './lib/stores/voice.svelte'
  import { getSettings, loadSettings, setNetworkMode, setP2PProfile, resetMode } from './lib/stores/settings.svelte'
  import {
    getFriends, loadFriends, setFriendsTab, openDM,
    sendFriendRequest, acceptFriendRequest, rejectFriendRequest,
    removeFriend, blockUser, unblockUser,
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
  import ServerInfoModal from './lib/components/server/ServerInfoModal.svelte'
  import Toast from './lib/components/ui/Toast.svelte'
  import { translations, t } from './lib/i18n'

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
  let showServerInfo = $state(false)
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
      setLocalUsername(auth.user.username)
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
      role: m.role === 'owner' ? 'Owner' : m.role === 'admin' ? 'Admin' : m.role === 'moderator' ? 'Moderator' : 'Member',
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

  const currentUserRole = $derived(
    auth.user ? (srv.members.find(m => m.user_id === auth.user!.id)?.role ?? 'member') : 'member'
  )

  const trans = $derived($translations)
</script>

{#if needsModeSelection}
  <ModeSelector onSelectMode={handleModeSelect} />

{:else if needsP2PProfile}
  <P2PProfile onConfirm={handleP2PProfileConfirm} />

{:else if isP2PMode}
  <P2PApp
    profile={settings.p2pProfile}
    onLogout={async () => { await leaveVoice(); resetMode() }}
    onSwitchMode={async () => { await leaveVoice(); resetMode() }}
  />

{:else if auth.loading}
  <div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
    <div class="text-center">
      <div class="mx-auto mb-4 h-10 w-10 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
      <p class="text-sm text-void-text-muted">{t(trans, 'app.loading')}</p>
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
        voiceNoiseSuppression={vc.noiseSuppression}
        voiceScreenSharing={vc.screenSharing}
        onToggleMute={toggleMute}
        onToggleDeafen={toggleDeafen}
        onToggleNoiseSuppression={toggleNoiseSuppression}
        onToggleScreenShare={toggleScreenSharing}
        onLeaveVoice={leaveVoice}
        onOpenSettings={() => showSettings = true}
      />

      {#if friends.activeDMId && friends.activeDM}
        <!-- DM conversation view -->
        <div class="flex flex-1 flex-col bg-void-bg-tertiary overflow-hidden animate-fade-in">
          <header class="flex h-12 items-center gap-2 border-b border-void-border px-4 shrink-0">
            <div class="h-6 w-6 rounded-full bg-void-accent flex items-center justify-center text-[10px] font-bold text-white">
              {friends.activeDM.display_name.slice(0, 2).toUpperCase()}
            </div>
            <span class="font-bold text-void-text-primary text-sm">{friends.activeDM.display_name}</span>
          </header>
          <div class="flex-1 flex items-center justify-center">
            <div class="text-center px-6">
              <div class="mx-auto mb-4 h-16 w-16 rounded-full bg-void-accent/20 flex items-center justify-center">
                <span class="text-xl font-bold text-void-accent">{friends.activeDM.display_name.slice(0, 2).toUpperCase()}</span>
              </div>
              <h3 class="text-lg font-bold text-void-text-primary mb-1">{friends.activeDM.display_name}</h3>
              <p class="text-sm text-void-text-muted">{t(trans, 'app.dmStart', { name: friends.activeDM.display_name })}</p>
            </div>
          </div>
          <div class="border-t border-void-border p-4 shrink-0">
            <div class="flex items-center gap-2 rounded-lg bg-void-bg-primary px-3 py-2.5">
              <input
                type="text"
                placeholder={t(trans, 'app.sendMessageTo', { name: friends.activeDM.display_name })}
                class="flex-1 bg-transparent text-sm text-void-text-primary placeholder:text-void-text-muted outline-none"
              />
            </div>
          </div>
        </div>
      {:else}
        <!-- Col 3: Friends list (flex-1) -->
        <FriendsList
          friends={friends.friends}
          pendingRequests={friends.pendingRequests}
          tab={friends.tab}
          addFriendError={friends.addFriendError}
          addFriendSuccess={friends.addFriendSuccess}
          onTabChange={(t) => setFriendsTab(t)}
          onAddFriend={(username) => sendFriendRequest(username)}
          onAcceptRequest={(id) => acceptFriendRequest(id)}
          onRejectRequest={(id) => rejectFriendRequest(id)}
          onRemoveFriend={(id) => removeFriend(id)}
          onBlockUser={(id) => blockUser(id)}
          onUnblockUser={(username) => unblockUser(username)}
          onMessage={(friendId) => openDM(`dm-${friendId}`)}
        />

        <!-- Col 4: Active Now (340px) -->
        <ActiveNow
          friends={friends.friends}
          voiceSpeakers={vc.speakers}
          voiceConnected={vc.connected}
          currentServerName={srv.active?.name ?? ''}
        />
      {/if}

    {:else}
      <!-- ══ SERVER VIEW (Channels + Chat + Members) ═════════════════════ -->

      <!-- Col 2: Channel sidebar (240px) -->
      <ChannelSidebar
        serverName={srv.active?.name ?? 'Servidor'}
        channels={sidebarChannels}
        activeChannelId={activeChannelId ?? ''}
        onSelectChannel={(id) => activeChannelId = id}
        onCreateChannel={(name, type) => srv.activeId && auth.user && createChannel(srv.activeId, auth.user.id, name, type)}
        onDeleteChannel={(channelId) => srv.activeId && auth.user && deleteChannel(srv.activeId, auth.user.id, channelId)}
        onServerInfo={() => showServerInfo = true}
        currentUser={currentUserProp}
        serverMembers={srv.members}
        currentUserRole={currentUserRole}
        voiceConnected={vc.connected}
        voiceChannelName={voiceChannelName}
        voiceMuted={vc.muted}
        voiceDeafened={vc.deafened}
        voiceSpeakers={vc.speakers}
        voiceChannelId={vc.channelId}
        voiceElapsed={vc.elapsed}
        voiceNoiseSuppression={vc.noiseSuppression}
        voiceScreenSharing={vc.screenSharing}
        voiceLocalSpeaking={vc.localSpeaking}
        onJoinVoice={handleJoinVoice}
        onLeaveVoice={leaveVoice}
        onToggleMute={toggleMute}
        onToggleDeafen={toggleDeafen}
        onToggleNoiseSuppression={toggleNoiseSuppression}
        onToggleScreenShare={toggleScreenSharing}
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
        searchResults={chat.searchResults}
        searchQuery={chat.searchQuery}
        onSend={handleSendMessage}
        onLoadMore={loadOlderMessages}
        onDelete={handleDeleteMessage}
        onFileSelect={handleFileSelect}
        onDownloadFile={handleDownloadFile}
        onDeleteFile={handleDeleteFile}
        onToggleMembers={() => showMembers = !showMembers}
        onSearch={(q) => activeChannelId && searchMessages(activeChannelId, q)}
        onClearSearch={clearSearch}
      />

      <!-- Col 4: Member sidebar (240px, toggle) -->
      {#if showMembers}
        <MemberSidebar
          members={sidebarMembers}
          currentUserId={auth.user?.id ?? ''}
          currentUserRole={currentUserRole}
          onUpdateRole={(memberId, role) => srv.activeId && auth.user && updateMemberRole(srv.activeId, auth.user.id, memberId, role)}
          onKickMember={(memberId) => srv.activeId && auth.user && kickMember(srv.activeId, auth.user.id, memberId)}
        />
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
    onLogout={async () => {
      await leaveVoice()
      resetChat()
      resetVoice()
      activeServerId = 'home'
      activeChannelId = null
      showSettings = false
      logout()
    }}
    onSwitchMode={async () => {
      await leaveVoice()
      resetChat()
      resetVoice()
      activeServerId = 'home'
      activeChannelId = null
      showSettings = false
      showMembers = false
      showCreateServer = false
      showJoinServer = false
      showServerInfo = false
      resetMode()
    }}
  />

  <ServerInfoModal
    bind:open={showServerInfo}
    serverName={srv.active?.name ?? ''}
    memberCount={srv.members.length}
    inviteCode={srv.active?.invite_code ?? ''}
    isOwner={srv.active?.owner_id === auth.user?.id}
    onClose={() => showServerInfo = false}
    onGenerateInvite={async () => {
      if (srv.activeId && auth.user) await generateInvite(srv.activeId, auth.user.id)
    }}
    onDeleteServer={async () => {
      if (srv.activeId && auth.user) {
        await deleteServer(srv.activeId, auth.user.id)
        activeServerId = 'home'
      }
    }}
  />
{/if}

<Toast />
