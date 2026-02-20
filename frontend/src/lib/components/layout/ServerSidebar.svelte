<script lang="ts">
  import Avatar from '../ui/Avatar.svelte'
  import Tooltip from '../ui/Tooltip.svelte'

  interface Server {
    id: string
    name: string
    icon?: string
    hasNotification?: boolean
  }

  let { servers, activeServerId, onSelectServer }: {
    servers: Server[]
    activeServerId: string
    onSelectServer: (id: string) => void
  } = $props()
</script>

<aside class="flex h-full w-[72px] flex-col items-center gap-2 bg-void-bg-primary py-3 overflow-y-auto">
  <!-- Home button -->
  <Tooltip text="Direct Messages" position="right">
    <button
      aria-label="Direct Messages"
      class="flex h-12 w-12 items-center justify-center rounded-2xl bg-void-bg-tertiary text-void-text-primary transition-all duration-200 hover:rounded-xl hover:bg-void-accent cursor-pointer"
      onclick={() => onSelectServer('home')}
    >
      <svg class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor">
        <path d="M21.47 4.35l-1.34-.56-.02-.01-8.58 3.56L3 3.78l-1.34.56 1.62 4.87-.38 13.34L5.47 24l5.06-3.19 5.06 3.19 2.57-1.45-.38-13.34 1.62-4.87.07.02zM12 15.5l-4.5-3.5L12 8.5l4.5 3.5-4.5 3.5z"/>
      </svg>
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
        {#if server.icon}
          <img src={server.icon} alt={server.name} class="h-full w-full rounded-[inherit] object-cover" />
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
  <Tooltip text="Add Server" position="right">
    <button aria-label="Add Server" class="flex h-12 w-12 items-center justify-center rounded-2xl bg-void-bg-tertiary text-void-online transition-all duration-200 hover:rounded-xl hover:bg-void-online hover:text-white cursor-pointer">
      <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
    </button>
  </Tooltip>

  <div class="flex-1"></div>

  <!-- User avatar -->
  <Tooltip text="Settings" position="right">
    <button class="cursor-pointer">
      <Avatar name="You" size="sm" status="online" />
    </button>
  </Tooltip>
</aside>
