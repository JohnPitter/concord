<script lang="ts">
  import type { Friend, FriendStatus, FriendRequest } from '../../stores/friends.svelte'

  type Tab = 'online' | 'all' | 'pending' | 'blocked'

  let {
    friends,
    pendingRequests = [],
    blockedUsers = [],
    tab,
    addFriendError = null,
    addFriendSuccess = null,
    onTabChange,
    onAddFriend,
    onAcceptRequest,
    onRejectRequest,
    onRemoveFriend,
    onBlockUser,
    onUnblockUser,
    onMessage,
  }: {
    friends: Friend[]
    pendingRequests?: FriendRequest[]
    blockedUsers?: string[]
    tab: Tab
    addFriendError?: string | null
    addFriendSuccess?: string | null
    onTabChange: (tab: Tab) => void
    onAddFriend?: (username: string) => void
    onAcceptRequest?: (requestId: string) => void
    onRejectRequest?: (requestId: string) => void
    onRemoveFriend?: (friendId: string) => void
    onBlockUser?: (friendId: string) => void
    onUnblockUser?: (username: string) => void
    onMessage?: (friendId: string) => void
  } = $props()

  let searchQuery = $state('')
  let addFriendInput = $state('')

  const displayed = $derived(() => {
    const base = tab === 'online'
      ? friends.filter(f => f.status !== 'offline')
      : tab === 'all'
        ? friends
        : []

    if (!searchQuery.trim()) return base
    return base.filter(f =>
      f.display_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.username.toLowerCase().includes(searchQuery.toLowerCase())
    )
  })

  const onlineCount = $derived(friends.filter(f => f.status !== 'offline').length)
  const incomingRequests = $derived(pendingRequests.filter(r => r.direction === 'incoming'))
  const outgoingRequests = $derived(pendingRequests.filter(r => r.direction === 'outgoing'))

  const statusLabel: Record<FriendStatus, string> = {
    online:  'Online',
    idle:    'Inativo',
    dnd:     'Não perturbe',
    offline: 'Offline',
  }

  const statusDot: Record<FriendStatus, string> = {
    online:  'bg-void-online',
    idle:    'bg-void-idle',
    dnd:     'bg-void-danger',
    offline: 'bg-void-text-muted',
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'online',  label: 'Online' },
    { key: 'all',     label: 'Todos' },
    { key: 'pending', label: 'Pendente' },
    { key: 'blocked', label: 'Bloqueado' },
  ]

  function handleAddFriend() {
    if (!addFriendInput.trim()) return
    onAddFriend?.(addFriendInput)
    addFriendInput = ''
  }

  let showSearch = $state(false)
</script>

<div class="flex flex-1 flex-col bg-void-bg-tertiary overflow-hidden">
  <!-- Header with tabs -->
  <header class="flex h-12 shrink-0 items-center gap-1 border-b border-void-border px-4">
    <!-- Friends icon + title -->
    <svg class="h-5 w-5 text-void-text-secondary shrink-0 mr-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
      <circle cx="9" cy="7" r="4"/>
      <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
      <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
    </svg>
    <span class="font-bold text-void-text-primary text-sm mr-3">Amigos</span>

    <div class="h-5 w-px bg-void-border mx-1"></div>

    <!-- Tabs -->
    {#each tabs as t}
      <button
        class="relative flex items-center gap-1.5 rounded-md px-2.5 py-1 text-sm font-medium transition-colors cursor-pointer
          {tab === t.key
            ? 'bg-void-bg-hover text-void-text-primary'
            : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
        onclick={() => onTabChange(t.key)}
      >
        {t.label}
        {#if t.key === 'pending' && incomingRequests.length > 0}
          <span class="flex h-4 min-w-4 items-center justify-center rounded-full bg-void-danger px-1 text-[10px] font-bold text-white">
            {incomingRequests.length}
          </span>
        {/if}
      </button>
    {/each}

    <!-- Add friend button -->
    <button
      onclick={() => onTabChange('pending')}
      class="ml-3 rounded-md bg-void-online/20 px-3 py-1 text-sm font-medium text-void-online hover:bg-void-online/30 transition-colors cursor-pointer"
    >
      Adicionar amigo
    </button>

    <!-- Right side: search icon -->
    <div class="ml-auto flex items-center gap-2">
      <button
        aria-label="Pesquisar"
        class="rounded-md p-1.5 transition-colors cursor-pointer {showSearch ? 'text-void-text-primary bg-void-bg-hover' : 'text-void-text-secondary hover:text-void-text-primary'}"
        onclick={() => { showSearch = !showSearch; if (!showSearch) searchQuery = '' }}
      >
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
      </button>
    </div>
  </header>

  <!-- Search bar -->
  {#if showSearch && (tab === 'online' || tab === 'all')}
    <div class="flex items-center gap-2 border-b border-void-border px-6 py-2 bg-void-bg-secondary shrink-0 animate-fade-in-down">
      <svg class="h-4 w-4 text-void-text-muted shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"/>
        <line x1="21" y1="21" x2="16.65" y2="16.65"/>
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Pesquisar amigos..."
        class="flex-1 bg-transparent text-sm text-void-text-primary placeholder:text-void-text-muted outline-none"
        onkeydown={(e) => { if (e.key === 'Escape') { showSearch = false; searchQuery = '' } }}
      />
      {#if searchQuery}
        <button
          class="text-xs text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
          onclick={() => searchQuery = ''}
        >
          Limpar
        </button>
      {/if}
    </div>
  {/if}

  <!-- Add friend bar (shown when tab = pending) -->
  {#if tab === 'pending'}
    <div class="border-b border-void-border px-6 py-4 shrink-0 animate-fade-in">
      <p class="mb-2 text-sm font-bold text-void-text-primary">ADICIONAR AMIGO</p>
      <p class="mb-3 text-sm text-void-text-secondary">Digite o nome de usuario do GitHub da pessoa que voce quer adicionar como amigo.</p>
      <div class="flex items-center gap-2 rounded-lg border border-void-border bg-void-bg-primary px-3 py-2">
        <span class="text-sm text-void-text-muted shrink-0">@</span>
        <input
          type="text"
          bind:value={addFriendInput}
          placeholder="username-do-github"
          class="flex-1 bg-transparent text-sm text-void-text-primary placeholder:text-void-text-muted focus:outline-none"
          onkeydown={(e) => { if (e.key === 'Enter') handleAddFriend() }}
        />
        <button
          class="rounded-md bg-void-accent px-3 py-1 text-xs font-semibold text-white hover:bg-void-accent-hover transition-colors cursor-pointer disabled:opacity-50"
          onclick={handleAddFriend}
          disabled={!addFriendInput.trim()}
        >
          Enviar pedido
        </button>
      </div>
      {#if addFriendError}
        <p class="mt-2 text-xs text-void-danger animate-fade-in">{addFriendError}</p>
      {/if}
      {#if addFriendSuccess}
        <p class="mt-2 text-xs text-void-online animate-fade-in">{addFriendSuccess}</p>
      {/if}

      <!-- Instructions -->
      <div class="mt-4 rounded-lg bg-void-bg-secondary border border-void-border p-3">
        <p class="text-xs font-bold text-void-text-primary mb-2">Como funciona?</p>
        <ol class="text-xs text-void-text-secondary space-y-1.5 list-decimal list-inside">
          <li>Digite o <strong class="text-void-text-primary">@username do GitHub</strong> da pessoa no campo acima</li>
          <li>Clique em <strong class="text-void-text-primary">"Enviar pedido"</strong> para enviar o convite</li>
          <li>O pedido aparecera na secao <strong class="text-void-text-primary">"Enviados"</strong> abaixo</li>
          <li>Quando a pessoa aceitar, ela aparecera na sua lista de amigos</li>
        </ol>
        <p class="mt-2 text-[11px] text-void-text-muted">Dica: ambos precisam ter uma conta no Concord logada com GitHub.</p>
      </div>
    </div>
  {/if}

  <!-- Friends list -->
  <div class="flex-1 overflow-y-auto px-4 py-2">
    {#if tab === 'blocked'}
      {#if blockedUsers.length === 0}
        <div class="flex flex-col items-center justify-center h-full gap-3 text-center animate-fade-in">
          <svg class="h-16 w-16 text-void-text-muted opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/>
            <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/>
          </svg>
          <p class="text-void-text-muted text-sm">Ninguem bloqueado.</p>
        </div>
      {:else}
        <p class="mb-2 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
          Bloqueados — {blockedUsers.length}
        </p>
        {#each blockedUsers as username}
          <div class="group flex items-center gap-3 rounded-lg px-2 py-2.5 hover:bg-void-bg-secondary transition-colors animate-fade-in-up">
            <div class="h-10 w-10 rounded-full bg-void-bg-hover flex items-center justify-center text-sm font-bold text-void-text-muted">
              {username.slice(0, 2).toUpperCase()}
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-void-text-primary truncate">{username}</p>
              <p class="text-xs text-void-text-muted">Bloqueado</p>
            </div>
            <button
              class="rounded-md px-2.5 py-1 text-xs font-medium bg-void-bg-hover text-void-text-secondary hover:text-void-text-primary transition-colors cursor-pointer opacity-0 group-hover:opacity-100"
              onclick={() => onUnblockUser?.(username)}
            >
              Desbloquear
            </button>
          </div>
          <div class="mx-2 h-px bg-void-border/50"></div>
        {/each}
      {/if}

    {:else if tab === 'pending'}
      <!-- Pending requests -->
      {#if pendingRequests.length === 0}
        <div class="flex flex-col items-center justify-center h-full gap-3 text-center animate-fade-in">
          <svg class="h-16 w-16 text-void-text-muted opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
            <circle cx="8.5" cy="7" r="4"/>
            <line x1="20" y1="8" x2="20" y2="14"/>
            <line x1="23" y1="11" x2="17" y2="11"/>
          </svg>
          <p class="text-void-text-muted text-sm">Nenhum pedido de amizade pendente.</p>
          <p class="text-xs text-void-text-muted">Envie um pedido usando o campo acima!</p>
        </div>
      {:else}
        {#if incomingRequests.length > 0}
          <p class="mb-2 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
            Recebidos — {incomingRequests.length}
          </p>
          {#each incomingRequests as request}
            <div class="group flex items-center gap-3 rounded-lg px-2 py-2.5 hover:bg-void-bg-secondary transition-colors animate-fade-in-up">
              <div class="h-10 w-10 rounded-full bg-void-accent flex items-center justify-center text-sm font-bold text-white">
                {request.display_name.slice(0, 2).toUpperCase()}
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-semibold text-void-text-primary truncate">{request.display_name}</p>
                <p class="text-xs text-void-text-muted">Pedido de amizade recebido</p>
              </div>
              <div class="flex items-center gap-1">
                <button
                  aria-label="Aceitar"
                  class="flex h-8 w-8 items-center justify-center rounded-full bg-void-bg-hover text-void-online hover:bg-void-online/20 transition-colors cursor-pointer"
                  onclick={() => onAcceptRequest?.(request.id)}
                >
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                </button>
                <button
                  aria-label="Rejeitar"
                  class="flex h-8 w-8 items-center justify-center rounded-full bg-void-bg-hover text-void-danger hover:bg-void-danger/20 transition-colors cursor-pointer"
                  onclick={() => onRejectRequest?.(request.id)}
                >
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>
            </div>
            <div class="mx-2 h-px bg-void-border/50"></div>
          {/each}
        {/if}

        {#if outgoingRequests.length > 0}
          <p class="mb-2 mt-3 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
            Enviados — {outgoingRequests.length}
          </p>
          {#each outgoingRequests as request}
            <div class="group flex items-center gap-3 rounded-lg px-2 py-2.5 hover:bg-void-bg-secondary transition-colors animate-fade-in-up">
              <div class="h-10 w-10 rounded-full bg-void-accent/30 flex items-center justify-center text-sm font-bold text-void-accent">
                {request.display_name.slice(0, 2).toUpperCase()}
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-semibold text-void-text-primary truncate">{request.display_name}</p>
                <p class="text-xs text-void-text-muted">Pedido enviado</p>
              </div>
              <button
                aria-label="Cancelar pedido"
                class="flex h-8 w-8 items-center justify-center rounded-full bg-void-bg-hover text-void-text-muted hover:text-void-danger transition-colors cursor-pointer"
                onclick={() => onRejectRequest?.(request.id)}
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="mx-2 h-px bg-void-border/50"></div>
          {/each}
        {/if}
      {/if}

    {:else if displayed().length === 0}
      <div class="flex flex-col items-center justify-center h-full gap-3 text-center animate-fade-in">
        <svg class="h-16 w-16 text-void-text-muted opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
          <circle cx="9" cy="7" r="4"/>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
        </svg>
        {#if searchQuery.trim()}
          <p class="text-void-text-muted text-sm">Nenhum amigo encontrado para "{searchQuery}"</p>
          <p class="text-xs text-void-text-muted">Tente pesquisar com outro nome.</p>
        {:else}
          <p class="text-void-text-muted text-sm">
            {#if tab === 'online'}Nenhum amigo online no momento.{:else}Voce ainda nao tem amigos adicionados.{/if}
          </p>
          <p class="text-xs text-void-text-muted">Adicione amigos usando a aba "Pendente"!</p>
        {/if}
      </div>
    {:else}
      <!-- Section header -->
      <p class="mb-2 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
        {tab === 'online' ? 'Online' : 'Todos os amigos'} — {displayed().length}
      </p>

      <!-- Friend rows -->
      {#each displayed() as friend, i}
        <div
          class="group flex items-center gap-3 rounded-lg px-2 py-2.5 hover:bg-void-bg-secondary transition-colors cursor-pointer"
          style="animation: fade-in-up 250ms ease-out {i * 30}ms both"
        >
          <!-- Avatar + status -->
          <div class="relative shrink-0">
            {#if friend.avatar_url}
              <img src={friend.avatar_url} alt={friend.display_name} class="h-10 w-10 rounded-full object-cover" />
            {:else}
              <div class="h-10 w-10 rounded-full bg-void-accent flex items-center justify-center text-sm font-bold text-white">
                {friend.display_name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            <span
              class="absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-void-bg-tertiary {statusDot[friend.status]}"
              title={statusLabel[friend.status]}
            ></span>
          </div>

          <!-- Name + activity -->
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-void-text-primary truncate">{friend.display_name}</p>
            <p class="text-xs text-void-text-muted truncate">
              {friend.activity ?? statusLabel[friend.status]}
            </p>
          </div>

          <!-- Action buttons (visible on hover) -->
          <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              aria-label="Mensagem"
              class="flex h-8 w-8 items-center justify-center rounded-full bg-void-bg-hover text-void-text-secondary hover:text-void-text-primary transition-colors cursor-pointer"
              title="Enviar mensagem"
              onclick={() => onMessage?.(friend.id)}
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </button>
            <button
              aria-label="Mais opcoes"
              class="flex h-8 w-8 items-center justify-center rounded-full bg-void-bg-hover text-void-text-secondary hover:text-void-text-primary transition-colors cursor-pointer"
              title="Mais opcoes"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                <circle cx="12" cy="5" r="1.5"/>
                <circle cx="12" cy="12" r="1.5"/>
                <circle cx="12" cy="19" r="1.5"/>
              </svg>
            </button>
          </div>
        </div>

        <div class="mx-2 h-px bg-void-border/50"></div>
      {/each}
    {/if}
  </div>
</div>
