<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    open: boolean
    title?: string
    children?: Snippet
  }

  let { open = $bindable(false), title, children }: Props = $props()

  function handleBackdropClick(e: MouseEvent) {
    if ((e.target as HTMLElement).dataset.backdrop) {
      open = false
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      open = false
    }
  }

  // Mount the modal outside the component tree so it always renders over everything
  function portal(node: HTMLElement) {
    document.body.appendChild(node)
    return {
      destroy() {
        node.parentNode?.removeChild(node)
      }
    }
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div
    use:portal
    data-backdrop="true"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
    onclick={handleBackdropClick}
    style="animation: backdrop-in 150ms ease-out;"
  >
    <div
      class="relative w-full max-w-lg rounded-lg border border-void-border bg-void-bg-secondary p-6 text-void-text-primary shadow-xl"
      style="animation: modal-in 150ms ease-out;"
    >
      {#if title}
        <h2 class="mb-4 text-lg font-bold">{title}</h2>
      {/if}
      {#if children}
        {@render children()}
      {/if}
    </div>
  </div>
{/if}

<style>
  @keyframes modal-in {
    from { opacity: 0; transform: scale(0.95); }
    to   { opacity: 1; transform: scale(1); }
  }
  @keyframes backdrop-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }
</style>
