<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    text: string
    position?: 'top' | 'bottom' | 'left' | 'right'
    delay?: number
    children?: Snippet
  }

  let { text, position = 'top', delay = 200, children }: Props = $props()

  let visible = $state(false)
  let timeout: ReturnType<typeof setTimeout> | undefined

  function show() {
    timeout = setTimeout(() => (visible = true), delay)
  }

  function hide() {
    clearTimeout(timeout)
    visible = false
  }

  const positionClasses: Record<string, string> = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="relative inline-flex"
  onmouseenter={show}
  onmouseleave={hide}
  onfocusin={show}
  onfocusout={hide}
>
  {#if children}
    {@render children()}
  {/if}

  {#if visible}
    <div
      class="pointer-events-none absolute z-50 whitespace-nowrap rounded-md bg-void-bg-tertiary px-2.5 py-1.5 text-xs text-void-text-primary shadow-md border border-void-border animate-[fade-in_100ms_ease-out] {positionClasses[position]}"
      role="tooltip"
    >
      {text}
    </div>
  {/if}
</div>

<style>
  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
