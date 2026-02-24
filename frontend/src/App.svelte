<script lang="ts">
  import { onMount } from 'svelte'
  import { getAuth, initAuth, logout, recoverFromStuckLoading } from './lib/stores/auth.svelte'
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
    startParticipantsPolling, stopParticipantsPolling,
    getChannelParticipants,
  } from './lib/stores/voice.svelte'
  import { getSettings, loadSettings, setNetworkMode, setP2PProfile, resetMode, markWelcomeSeen } from './lib/stores/settings.svelte'
  import {
    getFriends, loadFriends, setFriendsTab, openDM,
    sendFriendRequest, acceptFriendRequest, rejectFriendRequest,
    removeFriend, blockUser, unblockUser, sendDMMessage,
  } from './lib/stores/friends.svelte'

  import Login from './lib/components/auth/Login.svelte'
  import ModeSelector from './lib/components/auth/ModeSelector.svelte'
  import P2PProfile from './lib/components/auth/P2PProfile.svelte'
  import P2PApp from './lib/components/p2p/P2PApp.svelte'
  import CreateServerModal from './lib/components/server/CreateServer.svelte'
  import ServerSidebar from './lib/components/layout/ServerSidebar.svelte'
  import ChannelSidebar from './lib/components/layout/ChannelSidebar.svelte'
  import DMSidebar from './lib/components/layout/DMSidebar.svelte'
  import MainContent from './lib/components/layout/MainContent.svelte'
  import MemberSidebar from './lib/components/layout/MemberSidebar.svelte'
  import FriendsList from './lib/components/layout/FriendsList.svelte'
  import ActiveNow from './lib/components/layout/ActiveNow.svelte'
  import SettingsPanel from './lib/components/settings/SettingsPanel.svelte'
  import ServerInfoModal from './lib/components/server/ServerInfoModal.svelte'
  import Skeleton from './lib/components/ui/Skeleton.svelte'
  import Toast from './lib/components/ui/Toast.svelte'
  import MessageInput from './lib/components/chat/MessageInput.svelte'
  import WelcomeModal from './lib/components/onboarding/WelcomeModal.svelte'
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
  let showSettings = $state(false)
  let showMembers = $state(false)
  let showServerInfo = $state(false)
  let showWelcome = $state(false)
  let activeChannelId = $state<string | null>(null)

  const isHome = $derived(activeServerId === 'home')
  const AUTH_BOOTSTRAP_WATCHDOG_MS = 20_000
  let bootstrappedUserId = $state<string | null>(null)

  onMount(() => {
    loadSettings()
    void initAuth()

    const watchdog = setTimeout(() => {
      recoverFromStuckLoading('app bootstrap watchdog')
    }, AUTH_BOOTSTRAP_WATCHDOG_MS)

    return () => clearTimeout(watchdog)
  })

  // Load data once authenticated
  $effect(() => {
    if (auth.authenticated && auth.user) {
      if (bootstrappedUserId === auth.user.id) {
        return
      }
      bootstrappedUserId = auth.user.id
      void loadUserServers(auth.user.id)
      void loadFriends()
      setLocalUsername(auth.user.username, auth.user.id)
      if (!settings.hasSeenWelcome) {
        showWelcome = true
      }
    } else {
      bootstrappedUserId = null
    }
  })

  // Auto-select first server when in server mode and servers load
  $effect(() => {
    if (!isHome && srv.list.length > 0 && !srv.activeId) {
      selectServer(srv.list[0].id)
    }
  })

  // Poll voice channel participants when viewing a server
  $effect(() => {
    if (!isHome && srv.activeId) {
      const voiceChIDs = srv.channels.filter(c => c.type === 'voice').map(c => c.id)
      if (voiceChIDs.length > 0) {
        startParticipantsPolling(srv.activeId, voiceChIDs)
      }
      return () => { stopParticipantsPolling() }
    } else {
      stopParticipantsPolling()
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

  async function handleJoinServer(code: string): Promise<string | null> {
    if (!auth.user) return 'Not authenticated'
    try {
      const joined = await redeemInvite(code, auth.user.id)
      if (joined) {
        await loadUserServers(auth.user.id)
        activeServerId = joined.id
        await selectServer(joined.id)
        showCreateServer = false
        return null // success
      }
      return 'Invalid invite code or server not found'
    } catch (e) {
      return e instanceof Error ? e.message : 'Failed to join server'
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
      await joinVoice(
        activeServerId,
        channelId,
        auth.user?.id ?? '',
        auth.user?.username ?? '',
        auth.user?.avatar_url ?? '',
      )
    }
  }

  const currentUserProp = $derived(
    auth.user
      ? { id: auth.user.id, username: auth.user.username, display_name: auth.user.display_name, avatar_url: auth.user.avatar_url }
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
    <div class="grid h-full w-full grid-cols-[72px_240px_1fr_240px] gap-0">
      <div class="border-r border-void-border bg-void-bg-primary p-3">
        <div class="space-y-2">
          <Skeleton className="h-12 w-12 rounded-xl" />
          <Skeleton className="h-12 w-12 rounded-xl" />
          <Skeleton className="h-12 w-12 rounded-xl" />
          <Skeleton className="h-12 w-12 rounded-xl" />
        </div>
      </div>
      <div class="border-r border-void-border bg-void-bg-secondary p-4">
        <Skeleton className="mb-4 h-8 w-full rounded-lg" />
        <div class="space-y-2">
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
        </div>
      </div>
      <div class="bg-void-bg-tertiary p-4">
        <Skeleton className="mb-4 h-8 w-1/3 rounded-md" />
        <div class="space-y-4">
          <div class="flex gap-3">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div class="min-w-0 flex-1">
              <Skeleton className="mb-1 h-3.5 w-24 rounded-md" />
              <Skeleton className="h-3 w-2/3 rounded-md" />
            </div>
          </div>
          <div class="flex gap-3">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div class="min-w-0 flex-1">
              <Skeleton className="mb-1 h-3.5 w-28 rounded-md" />
              <Skeleton className="h-3 w-3/4 rounded-md" />
            </div>
          </div>
          <div class="flex gap-3">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div class="min-w-0 flex-1">
              <Skeleton className="mb-1 h-3.5 w-20 rounded-md" />
              <Skeleton className="h-3 w-1/2 rounded-md" />
            </div>
          </div>
        </div>
        <div class="mt-6">
          <Skeleton className="h-12 w-full rounded-lg" />
        </div>
      </div>
      <div class="bg-void-bg-secondary p-4">
        <Skeleton className="mb-3 h-4 w-20 rounded-md" />
        <div class="space-y-2">
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
          <Skeleton className="h-8 w-full rounded-md" />
        </div>
      </div>
    </div>
  </div>

{:else if !auth.authenticated}
  <Login onBack={() => resetMode()} />

{:else}
  <div class="flex h-screen w-screen overflow-hidden">

    <!-- Col 1: Server rail (72px) -->
    <ServerSidebar
      servers={sidebarServers}
      activeServerId={activeServerId}
      loading={srv.loading}
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
        loading={friends.loading}
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
        onNewDM={() => openDM(null)}
      />

      {#if friends.activeDMId && friends.activeDM}
        <!-- DM conversation view -->
        <div class="flex flex-1 flex-col bg-void-bg-tertiary overflow-hidden animate-fade-in">
          <header class="flex h-12 items-center gap-2 border-b border-void-border px-4 shrink-0">
            {#if friends.loading}
              <Skeleton className="h-6 w-6 rounded-full" />
            {:else if friends.activeDM.avatar_url}
              <img src={friends.activeDM.avatar_url} alt={friends.activeDM.display_name} class="h-6 w-6 rounded-full object-cover" />
            {:else}
              <div class="h-6 w-6 rounded-full bg-void-accent flex items-center justify-center text-[10px] font-bold text-white">
                {friends.activeDM.display_name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            {#if friends.loading}
              <Skeleton className="h-4 w-28 rounded-md" />
            {:else}
              <span class="font-bold text-void-text-primary text-sm">{friends.activeDM.display_name}</span>
            {/if}
          </header>
          <div class="flex-1 overflow-y-auto px-4 py-4">
            {#if friends.loading}
              <div class="space-y-4">
                {#each Array.from({ length: 6 }) as _, i (`dm-msg-sk-${i}`)}
                  <div class="flex gap-3 {i % 2 === 0 ? '' : 'justify-end'}">
                    {#if i % 2 === 0}
                      <Skeleton className="h-8 w-8 rounded-full" />
                    {/if}
                    <div class="min-w-[180px] max-w-[70%]">
                      <Skeleton className="mb-1 h-3.5 w-20 rounded-md" />
                      <Skeleton className="mb-1 h-3.5 w-full rounded-md" />
                      <Skeleton className="h-3.5 w-2/3 rounded-md" />
                    </div>
                  </div>
                {/each}
              </div>
            {:else if friends.activeDMMessages.length === 0}
              <div class="flex flex-col items-center justify-center h-full">
                <div class="text-center px-6">
                  <div class="mx-auto mb-4 h-16 w-16 rounded-full bg-void-accent/20 flex items-center justify-center">
                    <span class="text-xl font-bold text-void-accent">{friends.activeDM.display_name.slice(0, 2).toUpperCase()}</span>
                  </div>
                  <h3 class="text-lg font-bold text-void-text-primary mb-1">{friends.activeDM.display_name}</h3>
                  <p class="text-sm text-void-text-muted">{t(trans, 'app.dmStart', { name: friends.activeDM.display_name })}</p>
                </div>
              </div>
            {:else}
              <div class="space-y-3">
                {#each friends.activeDMMessages as msg}
                  <div class="flex gap-2 {msg.senderId === auth.user?.id ? 'justify-end' : ''}">
                    <div class="max-w-[70%] rounded-lg px-3 py-2 {msg.senderId === auth.user?.id ? 'bg-void-accent text-white' : 'bg-void-bg-secondary text-void-text-primary'}">
                      <p class="text-sm break-words">{msg.content}</p>
                      <p class="text-[10px] mt-0.5 {msg.senderId === auth.user?.id ? 'text-white/60' : 'text-void-text-muted'}">{new Date(msg.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</p>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
          {#if friends.loading}
            <div class="border-t border-void-border px-4 py-4">
              <div class="flex items-end gap-2 rounded-lg bg-void-bg-secondary px-4 py-3">
                <Skeleton className="h-5 w-5 rounded-sm" />
                <Skeleton className="h-5 flex-1 rounded-md" />
                <Skeleton className="h-5 w-5 rounded-sm" />
              </div>
            </div>
          {:else}
            <MessageInput
              channelName={friends.activeDM.display_name}
              onSend={(content) => {
                if (friends.activeDMId && auth.user) {
                  sendDMMessage(friends.activeDMId, auth.user.id, content)
                }
              }}
            />
          {/if}
        </div>
      {:else}
        <!-- Col 3: Friends list (flex-1) -->
        <FriendsList
          friends={friends.friends}
          pendingRequests={friends.pendingRequests}
          loading={friends.loading}
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
          loading={friends.loading}
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
        loading={srv.channelsLoading}
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
        {getChannelParticipants}
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
        loading={chat.loading || srv.channelsLoading}
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
          loading={srv.membersLoading}
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

{#if showWelcome}
  <WelcomeModal
    onClose={() => { showWelcome = false; markWelcomeSeen() }}
  />
{/if}

<Toast />
