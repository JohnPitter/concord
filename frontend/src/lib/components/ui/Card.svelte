<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    interactive?: boolean
    children?: Snippet
  }

  let { interactive = false, children }: Props = $props()
</script>

<div
  class="rounded-lg border border-void-border bg-void-bg-tertiary p-4 shadow-sm transition-all duration-150
    {interactive ? 'cursor-pointer hover:border-void-accent hover:shadow-glow' : ''}"
>
  {#if children}
    {@render children()}
  {/if}
</div>
