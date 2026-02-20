<script lang="ts">
  interface Props {
    src?: string
    name: string
    size?: 'sm' | 'md' | 'lg'
    status?: 'online' | 'idle' | 'dnd' | 'offline'
  }

  let { src, name, size = 'md', status }: Props = $props()

  const initials = $derived(
    name
      .split(' ')
      .map((w) => w[0])
      .join('')
      .slice(0, 2)
      .toUpperCase()
  )

  const sizeClasses: Record<string, string> = {
    sm: 'h-8 w-8 text-xs',
    md: 'h-10 w-10 text-sm',
    lg: 'h-12 w-12 text-base',
  }

  const statusDotSize: Record<string, string> = {
    sm: 'h-2.5 w-2.5',
    md: 'h-3 w-3',
    lg: 'h-3.5 w-3.5',
  }

  const statusColors: Record<string, string> = {
    online: 'bg-void-online',
    idle: 'bg-void-idle',
    dnd: 'bg-void-dnd',
    offline: 'bg-void-offline',
  }

  let imgError = $state(false)
</script>

<div class="relative inline-flex shrink-0">
  {#if src && !imgError}
    <img
      {src}
      alt={name}
      class="rounded-full object-cover {sizeClasses[size]}"
      onerror={() => imgError = true}
    />
  {:else}
    <div
      class="flex items-center justify-center rounded-full bg-void-accent font-medium text-white {sizeClasses[size]}"
      aria-label={name}
    >
      {initials}
    </div>
  {/if}

  {#if status}
    <span
      class="absolute bottom-0 right-0 rounded-full ring-2 ring-void-bg-primary {statusDotSize[size]} {statusColors[status]}"
      aria-label={status}
    ></span>
  {/if}
</div>
