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
      <!-- Concord logo mark (mini) -->
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" class="h-7 w-7">
        <defs>
          <linearGradient id="g-ss" x1="0" y1="0" x2="128" y2="128" gradientUnits="userSpaceOnUse">
            <stop offset="0%" stop-color={isHome ? '#fff' : '#4ade80'}/>
            <stop offset="100%" stop-color={isHome ? '#d4fae8' : '#16a34a'}/>
          </linearGradient>
        </defs>
        <path fill="url(#g-ss)" d="M 58,6 C 89,6 114,22 120,48 C 121,52 118,56 114,56 C 110,56 107,53 106,50 C 102,30 82,18 58,18 C 28,18 10,38 10,59 C 10,80 28,100 58,100 C 82,100 102,88 106,68 C 107,65 110,62 114,62 C 118,62 121,66 120,70 C 114,96 89,112 58,112 L 42,112 L 16,126 L 22,108 C 8,100 -2,82 -2,59 C -2,28 22,6 58,6 Z"/>
        <path d="M 40,47 A 17,17 0 0 1 40,71" fill="none" stroke="#fff" stroke-width="8" stroke-linecap="round" opacity="0.95"/>
        <path d="M 54,39 A 26,26 0 0 1 54,79" fill="none" stroke="#fff" stroke-width="7" stroke-linecap="round" opacity="0.6"/>
        <path d="M 68,32 A 34,34 0 0 1 68,86" fill="none" stroke="#fff" stroke-width="6" stroke-linecap="round" opacity="0.3"/>
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
