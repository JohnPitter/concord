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
  let triggerEl: HTMLDivElement | undefined = $state()
  let tooltipX = $state(0)
  let tooltipY = $state(0)

  function show() {
    timeout = setTimeout(() => {
      if (triggerEl) {
        const rect = triggerEl.getBoundingClientRect()
        if (position === 'right') {
          tooltipX = rect.right + 8
          tooltipY = rect.top + rect.height / 2
        } else if (position === 'left') {
          tooltipX = rect.left - 8
          tooltipY = rect.top + rect.height / 2
        } else if (position === 'bottom') {
          tooltipX = rect.left + rect.width / 2
          tooltipY = rect.bottom + 6
        } else {
          tooltipX = rect.left + rect.width / 2
          tooltipY = rect.top - 6
        }
      }
      visible = true
    }, delay)
  }

  function hide() {
    clearTimeout(timeout)
    visible = false
  }

  const transformMap: Record<string, string> = {
    top: 'translate(-50%, -100%)',
    bottom: 'translate(-50%, 0)',
    left: 'translate(-100%, -50%)',
    right: 'translate(0, -50%)',
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  bind:this={triggerEl}
  class="inline-flex"
  onmouseenter={show}
  onmouseleave={hide}
  onfocusin={show}
  onfocusout={hide}
>
  {#if children}
    {@render children()}
  {/if}
</div>

{#if visible}
  <div
    class="pointer-events-none fixed z-[100] whitespace-nowrap rounded-md bg-void-bg-tertiary px-2.5 py-1.5 text-xs text-void-text-primary shadow-md border border-void-border"
    role="tooltip"
    style="left: {tooltipX}px; top: {tooltipY}px; transform: {transformMap[position]}; animation: tooltip-fade 100ms ease-out"
  >
    {text}
  </div>
{/if}

<style>
  @keyframes tooltip-fade {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
