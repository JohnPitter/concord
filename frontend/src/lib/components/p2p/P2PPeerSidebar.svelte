<script lang="ts">
  interface P2PPeer {
    id: string
    displayName: string
    avatarDataUrl?: string
    connected: boolean
    source: 'lan' | 'room'
  }

  let {
    peers,
    activePeerID,
    roomCode,
    profile,
    onSelectPeer,
    onJoinRoom,
    onOpenSettings,
  }: {
    peers: P2PPeer[]
    activePeerID: string | null
    roomCode: string
    profile: { displayName: string; avatarDataUrl?: string } | null
    onSelectPeer: (id: string) => void
    onJoinRoom: (code: string) => void
    onOpenSettings: () => void
  } = $props()

  let joinCode = $state('')
  let copied = $state(false)

  function copyRoomCode() {
    navigator.clipboard.writeText(roomCode)
    copied = true
    setTimeout(() => copied = false, 2000)
  }

  function handleJoin() {
    const code = joinCode.trim()
    if (!code) return
    onJoinRoom(code)
    joinCode = ''
  }

  const profileDisplayName = $derived(profile?.displayName ?? 'P2P')
  const profileInitials = $derived(profileDisplayName.slice(0, 2).toUpperCase())

  const lanPeers = $derived(peers.filter(p => p.source === 'lan'))
  const roomPeers = $derived(peers.filter(p => p.source === 'room'))

  /** Generate a consistent hue from a peer ID for avatar colors */
  function peerHue(id: string): number {
    let hash = 0
    for (let i = 0; i < id.length; i++) {
      hash = id.charCodeAt(i) + ((hash << 5) - hash)
    }
    return Math.abs(hash) % 360
  }

  function peerInitials(peer: P2PPeer): string {
    if (peer.displayName) return peer.displayName.slice(0, 2).toUpperCase()
    return peer.id.slice(0, 2).toUpperCase()
  }

  function peerLabel(peer: P2PPeer): string {
    return peer.displayName || (peer.id.length > 12 ? peer.id.slice(0, 12) + '...' : peer.id)
  }
</script>

<aside class="flex h-full w-60 flex-col bg-void-bg-secondary border-r border-void-border">
  <!-- Profile header -->
  <div class="flex items-center gap-2 border-b border-void-border p-3 shrink-0">
    {#if profile?.avatarDataUrl}
      <img src={profile.avatarDataUrl} alt={profileDisplayName} class="h-8 w-8 shrink-0 rounded-full object-cover" />
    {:else}
      <div class="h-8 w-8 shrink-0 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
        {profileInitials}
      </div>
    {/if}
    <span class="text-sm font-bold text-void-text-primary truncate">{profileDisplayName}</span>
  </div>

  <!-- Room code card -->
  <div class="mx-2 my-3 rounded-lg bg-void-bg-tertiary p-3">
    <div class="flex items-center justify-between mb-2">
      <span class="text-[10px] font-bold uppercase tracking-wider text-void-text-muted">Sala</span>
      <button
        class="rounded p-1 text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
        onclick={copyRoomCode}
        aria-label="Copiar codigo da sala"
      >
        {#if copied}
          <svg class="h-3.5 w-3.5 text-void-online" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        {:else}
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
        {/if}
      </button>
    </div>
    <p class="font-mono text-sm text-void-accent select-all mb-3">{roomCode}</p>

    <div class="flex gap-1.5">
      <input
        type="text"
        bind:value={joinCode}
        placeholder="Codigo da sala"
        class="flex-1 min-w-0 rounded-md bg-void-bg-primary px-2 py-1.5 text-xs text-void-text-primary placeholder:text-void-text-muted outline-none focus:ring-1 focus:ring-void-accent"
        onkeydown={(e) => { if (e.key === 'Enter') handleJoin() }}
      />
      <button
        class="shrink-0 rounded-md bg-void-accent px-2.5 py-1.5 text-xs font-medium text-white hover:bg-void-accent-hover transition-colors cursor-pointer disabled:opacity-50 disabled:pointer-events-none"
        onclick={handleJoin}
        disabled={!joinCode.trim()}
      >
        Entrar
      </button>
    </div>
  </div>

  <!-- Peer list -->
  <div class="flex-1 overflow-y-auto px-2">
    {#if peers.length === 0}
      <p class="px-2 py-4 text-center text-xs text-void-text-muted">Nenhum peer encontrado</p>
    {:else}
      {#if lanPeers.length > 0}
        <div class="flex items-center gap-1 px-1 pt-2 pb-1">
          <span class="text-[10px] font-bold uppercase tracking-wider text-void-text-muted">Na Rede Local</span>
        </div>
        {#each lanPeers as peer (peer.id)}
          <button
            class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors cursor-pointer
              {peer.id === activePeerID
                ? 'bg-void-bg-hover text-void-text-primary'
                : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
            onclick={() => onSelectPeer(peer.id)}
          >
            {#if peer.avatarDataUrl}
              <img src={peer.avatarDataUrl} alt={peerLabel(peer)} class="h-8 w-8 shrink-0 rounded-full object-cover" />
            {:else}
              <div
                class="h-8 w-8 shrink-0 rounded-full flex items-center justify-center text-xs font-bold text-white"
                style="background-color: hsl({peerHue(peer.id)}, 60%, 45%)"
              >
                {peerInitials(peer)}
              </div>
            {/if}
            <span class="flex-1 min-w-0 truncate text-left">{peerLabel(peer)}</span>
            <span class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold bg-void-online/20 text-void-online">LAN</span>
          </button>
        {/each}
      {/if}

      {#if roomPeers.length > 0}
        <div class="flex items-center gap-1 px-1 pt-3 pb-1">
          <span class="text-[10px] font-bold uppercase tracking-wider text-void-text-muted">Na Sala</span>
        </div>
        {#each roomPeers as peer (peer.id)}
          <button
            class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors cursor-pointer
              {peer.id === activePeerID
                ? 'bg-void-bg-hover text-void-text-primary'
                : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
            onclick={() => onSelectPeer(peer.id)}
          >
            {#if peer.avatarDataUrl}
              <img src={peer.avatarDataUrl} alt={peerLabel(peer)} class="h-8 w-8 shrink-0 rounded-full object-cover" />
            {:else}
              <div
                class="h-8 w-8 shrink-0 rounded-full flex items-center justify-center text-xs font-bold text-white"
                style="background-color: hsl({peerHue(peer.id)}, 60%, 45%)"
              >
                {peerInitials(peer)}
              </div>
            {/if}
            <span class="flex-1 min-w-0 truncate text-left">{peerLabel(peer)}</span>
            <span class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold bg-blue-500/20 text-blue-400">WAN</span>
          </button>
        {/each}
      {/if}
    {/if}
  </div>

  <!-- Settings footer -->
  <div class="border-t border-void-border p-2 shrink-0">
    <button
      class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary transition-colors cursor-pointer"
      onclick={onOpenSettings}
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
      <span>Configuracoes</span>
    </button>
  </div>
</aside>
