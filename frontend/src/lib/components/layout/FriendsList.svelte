<script lang="ts">
  import type { Friend, FriendStatus } from '../../stores/friends.svelte'

  type Tab = 'online' | 'all' | 'pending' | 'blocked'

  let {
    friends,
    tab,
    onTabChange,
    onAddFriend,
  }: {
    friends: Friend[]
    tab: Tab
    onTabChange: (tab: Tab) => void
    onAddFriend?: () => void
  } = $props()

  const displayed = $derived(
    tab === 'online'
      ? friends.filter(f => f.status !== 'offline')
      : tab === 'all'
        ? friends
        : []
  )

  const onlineCount = $derived(friends.filter(f => f.status !== 'offline').length)

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

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: 'online',  label: 'Online' },
    { key: 'all',     label: 'Todos' },
    { key: 'pending', label: 'Pendente', count: 0 },
    { key: 'blocked', label: 'Bloqueado' },
  ]
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
        {#if t.count}
          <span class="flex h-4 min-w-4 items-center justify-center rounded-full bg-void-danger px-1 text-[10px] font-bold text-white">
            {t.count}
          </span>
        {/if}
      </button>
    {/each}

    <!-- Add friend button -->
    <button
      onclick={onAddFriend}
      class="ml-3 rounded-md bg-void-online/20 px-3 py-1 text-sm font-medium text-void-online hover:bg-void-online/30 transition-colors cursor-pointer"
    >
      Adicionar amigo
    </button>

    <!-- Right side: search + help icons -->
    <div class="ml-auto flex items-center gap-2">
      <button aria-label="Pesquisar" class="rounded-md p-1.5 text-void-text-secondary hover:text-void-text-primary transition-colors cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
      </button>
    </div>
  </header>

  <!-- Add friend bar (shown when tab = pending or explicit action) -->
  {#if tab === 'pending'}
    <div class="border-b border-void-border px-6 py-4 shrink-0">
      <p class="mb-2 text-sm font-bold text-void-text-primary">ADICIONAR AMIGO</p>
      <p class="mb-3 text-sm text-void-text-secondary">Você pode adicionar amigos usando o nome de usuário deles.</p>
      <div class="flex items-center gap-2 rounded-lg border border-void-border bg-void-bg-primary px-3 py-2">
        <input
          type="text"
          placeholder="Você pode adicionar amigos usando o nome de usuário deles."
          class="flex-1 bg-transparent text-sm text-void-text-primary placeholder:text-void-text-muted focus:outline-none"
        />
        <button class="rounded-md bg-void-accent px-3 py-1 text-xs font-semibold text-white hover:bg-void-accent-hover transition-colors cursor-pointer">
          Enviar pedido de amizade
        </button>
      </div>
    </div>
  {/if}

  <!-- Friends list -->
  <div class="flex-1 overflow-y-auto px-4 py-2">
    {#if tab === 'blocked'}
      <div class="flex flex-col items-center justify-center h-full gap-3 text-center">
        <svg class="h-16 w-16 text-void-text-muted opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/>
        </svg>
        <p class="text-void-text-muted text-sm">Ninguém bloqueado.</p>
      </div>
    {:else if displayed.length === 0}
      <div class="flex flex-col items-center justify-center h-full gap-3 text-center">
        <svg class="h-16 w-16 text-void-text-muted opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
          <circle cx="9" cy="7" r="4"/>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
        </svg>
        <p class="text-void-text-muted text-sm">Nenhum amigo {tab === 'online' ? 'online' : ''} por enquanto.</p>
      </div>
    {:else}
      <!-- Section header -->
      <p class="mb-2 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
        {tab === 'online' ? 'Online' : 'Todos os amigos'} — {displayed.length}
      </p>

      <!-- Friend rows -->
      {#each displayed as friend}
        <div class="group flex items-center gap-3 rounded-lg px-2 py-2.5 hover:bg-void-bg-secondary transition-colors cursor-pointer">
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
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </button>
            <button
              aria-label="Mais opções"
              class="flex h-8 w-8 items-center justify-center rounded-full bg-void-bg-hover text-void-text-secondary hover:text-void-text-primary transition-colors cursor-pointer"
              title="Mais opções"
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
