<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    variant?: 'default' | 'success' | 'warning' | 'danger'
    children?: Snippet
  }

  let { variant = 'default', children }: Props = $props()

  const variantClasses: Record<string, string> = {
    default: 'bg-void-bg-hover text-void-text-secondary',
    success: 'bg-void-success/15 text-void-success',
    warning: 'bg-void-warning/15 text-void-warning',
    danger: 'bg-void-danger/15 text-void-danger',
  }
</script>

<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {variantClasses[variant]}">
  {#if children}
    {@render children()}
  {/if}
</span>
