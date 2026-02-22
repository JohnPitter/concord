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
          <linearGradient id="wl-ss" x1="0" y1="20" x2="55" y2="85" gradientUnits="userSpaceOnUse">
            <stop offset="0%" stop-color={isHome ? '#86efac' : '#a3e635'}/>
            <stop offset="50%" stop-color={isHome ? '#22c55e' : '#22c55e'}/>
            <stop offset="100%" stop-color={isHome ? '#16a34a' : '#15803d'}/>
          </linearGradient>
          <linearGradient id="wr-ss" x1="73" y1="20" x2="128" y2="85" gradientUnits="userSpaceOnUse">
            <stop offset="0%" stop-color={isHome ? '#86efac' : '#4ade80'}/>
            <stop offset="50%" stop-color={isHome ? '#22c55e' : '#16a34a'}/>
            <stop offset="100%" stop-color={isHome ? '#16a34a' : '#166534'}/>
          </linearGradient>
        </defs>
        <path d="M56 50 C44 30 20 10 4 22 C16 34 36 50 48 58 Z" fill="url(#wl-ss)" opacity="0.72"/>
        <path d="M54 60 C38 44 12 30 0 46 C14 52 38 58 50 64 Z" fill="url(#wl-ss)" opacity="0.88"/>
        <path d="M52 70 C34 58 8 48 0 64 C14 68 40 72 50 74 Z" fill="url(#wl-ss)"/>
        <path d="M72 50 C84 30 108 10 124 22 C112 34 92 50 80 58 Z" fill="url(#wr-ss)" opacity="0.72"/>
        <path d="M74 60 C90 44 116 30 128 46 C114 52 90 58 78 64 Z" fill="url(#wr-ss)" opacity="0.88"/>
        <path d="M76 70 C94 58 120 48 128 64 C114 68 88 72 78 74 Z" fill="url(#wr-ss)"/>
        <ellipse cx="64" cy="68" rx="14" ry="17" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <circle cx="64" cy="46" r="11" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <path d="M55 52 Q64 63 73 52" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <rect x="55" y="50" width="18" height="16" rx="5" fill={isHome ? '#fff' : '#f0fdf4'}/>
        <circle cx="64" cy="44" r="2.5" fill={isHome ? '#1a1a2e' : '#1a2e1a'}/>
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
