<script lang="ts">
  import Avatar from '../ui/Avatar.svelte'
  import Tooltip from '../ui/Tooltip.svelte'

  interface Server {
    id: string
    name: string
    iconUrl?: string
    hasNotification?: boolean
  }

  interface CurrentUser {
    username: string
    display_name: string
    avatar_url: string
  }

  let { servers, activeServerId, onSelectServer, onAddServer, currentUser = null }: {
    servers: Server[]
    activeServerId: string   // 'home' | server.id
    onSelectServer: (id: string) => void
    onAddServer?: () => void
    currentUser?: CurrentUser | null
  } = $props()

  const displayName = $derived(currentUser?.display_name || currentUser?.username || 'You')
  const isHome = $derived(activeServerId === 'home')
</script>

<aside class="flex h-full w-[72px] flex-col items-center gap-2 bg-void-bg-primary py-3 overflow-y-auto overflow-x-hidden scrollbar-none">
  <!-- Home / DMs button -->
  <Tooltip text="Mensagens Diretas" position="right">
    <button
      aria-label="Mensagens Diretas"
      class="relative flex h-12 w-12 items-center justify-center transition-all duration-200 cursor-pointer
        {isHome
          ? 'rounded-xl bg-void-accent text-white'
          : 'rounded-2xl bg-void-bg-tertiary text-void-text-primary hover:rounded-xl hover:bg-void-accent hover:text-white'}"
      onclick={() => onSelectServer('home')}
    >
      <!-- Concord dove logo (mini) -->
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" class="h-7 w-7">
        <defs>
          <linearGradient id="wl-ss" x1="10" y1="30" x2="60" y2="90" gradientUnits="userSpaceOnUse">
            <stop offset="0%" stop-color={isHome ? '#86efac' : '#4ade80'}/>
            <stop offset="100%" stop-color={isHome ? '#22c55e' : '#15803d'}/>
          </linearGradient>
          <linearGradient id="wr-ss" x1="68" y1="30" x2="118" y2="90" gradientUnits="userSpaceOnUse">
            <stop offset="0%" stop-color={isHome ? '#86efac' : '#22c55e'}/>
            <stop offset="100%" stop-color={isHome ? '#16a34a' : '#166534'}/>
          </linearGradient>
        </defs>
        <path d="M58 58 C50 42 28 20 8 28 C18 38 30 52 42 60 Z" fill="url(#wl-ss)" opacity="0.7"/>
        <path d="M56 64 C44 52 18 38 4 50 C16 56 36 62 50 66 Z" fill="url(#wl-ss)" opacity="0.85"/>
        <path d="M54 70 C40 62 14 56 4 68 C16 70 38 72 50 72 Z" fill="url(#wl-ss)"/>
        <path d="M70 58 C78 42 100 20 120 28 C110 38 98 52 86 60 Z" fill="url(#wr-ss)" opacity="0.7"/>
        <path d="M72 64 C84 52 110 38 124 50 C112 56 92 62 78 66 Z" fill="url(#wr-ss)" opacity="0.85"/>
        <path d="M74 70 C88 62 114 56 124 68 C112 70 90 72 78 72 Z" fill="url(#wr-ss)"/>
        <ellipse cx="64" cy="68" rx="14" ry="16" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <circle cx="64" cy="48" r="11" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <path d="M56 54 Q64 62 72 54" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <rect x="56" y="52" width="16" height="14" rx="4" fill={isHome ? '#fff' : '#f0fdf4'}/>
      </svg>

      {#if isHome}
        <span class="absolute -left-1 top-1/2 h-10 w-1 -translate-y-1/2 rounded-r-full bg-white"></span>
      {/if}
    </button>
  </Tooltip>

  <div class="mx-3 h-px w-8 bg-void-border"></div>

  <!-- Server list -->
  {#each servers as server}
    <Tooltip text={server.name} position="right">
      <button
        class="relative flex h-12 w-12 items-center justify-center rounded-2xl transition-all duration-200 hover:rounded-xl cursor-pointer
          {server.id === activeServerId
            ? 'rounded-xl bg-void-accent text-white'
            : 'bg-void-bg-tertiary text-void-text-primary hover:bg-void-accent-hover hover:text-white'}"
        onclick={() => onSelectServer(server.id)}
      >
        {#if server.iconUrl}
          <img src={server.iconUrl} alt={server.name} class="h-full w-full rounded-[inherit] object-cover" />
        {:else}
          <span class="text-sm font-bold">{server.name.slice(0, 2).toUpperCase()}</span>
        {/if}

        {#if server.id === activeServerId}
          <span class="absolute -left-1 top-1/2 h-10 w-1 -translate-y-1/2 rounded-r-full bg-white"></span>
        {:else if server.hasNotification}
          <span class="absolute -left-1 top-1/2 h-2 w-1 -translate-y-1/2 rounded-r-full bg-white"></span>
        {/if}
      </button>
    </Tooltip>
  {/each}

  <!-- Add server button -->
  <Tooltip text="Adicionar Servidor" position="right">
    <button
      aria-label="Adicionar Servidor"
      class="flex h-12 w-12 items-center justify-center rounded-2xl bg-void-bg-tertiary text-void-online transition-all duration-200 hover:rounded-xl hover:bg-void-online hover:text-white cursor-pointer"
      onclick={() => onAddServer?.()}
    >
      <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
    </button>
  </Tooltip>

  <div class="flex-1 shrink-0"></div>

  <!-- User avatar -->
  <div class="pb-1 shrink-0">
    {#if currentUser?.avatar_url}
      <div class="relative">
        <img src={currentUser.avatar_url} alt={displayName} class="h-10 w-10 rounded-full object-cover" />
        <span class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-void-bg-primary bg-void-online"></span>
      </div>
    {:else}
      <Avatar name={displayName} size="sm" status="online" />
    {/if}
  </div>
</aside>
