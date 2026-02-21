<script lang="ts">
  import type { Snippet } from 'svelte'
  import type { HTMLButtonAttributes } from 'svelte/elements'

  interface Props extends HTMLButtonAttributes {
    variant?: 'solid' | 'outline' | 'ghost' | 'danger'
    size?: 'sm' | 'md' | 'lg'
    loading?: boolean
    children?: Snippet
  }

  let {
    variant = 'solid',
    size = 'md',
    loading = false,
    disabled = false,
    class: extraClass = '',
    children,
    ...rest
  }: Props & { class?: string } = $props()

  const sizeClasses: Record<string, string> = {
    sm: 'px-2.5 py-1 text-xs',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-2.5 text-base',
  }

  const variantClasses: Record<string, string> = {
    solid: 'bg-void-accent text-white hover:bg-void-accent-hover shadow-sm active:scale-[0.97]',
    outline: 'border border-void-border text-void-text-primary hover:border-void-accent hover:text-void-accent',
    ghost: 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary',
    danger: 'bg-void-danger text-white hover:bg-void-danger-hover active:scale-[0.97]',
  }
</script>

<button
  class="inline-flex cursor-pointer items-center justify-center gap-2 rounded-md font-medium transition-all duration-150 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-void-accent disabled:pointer-events-none disabled:opacity-50 {sizeClasses[size]} {variantClasses[variant]} {extraClass}"
  disabled={disabled || loading}
  aria-busy={loading}
  {...rest}
>
  {#if loading}
    <svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
    </svg>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</button>
