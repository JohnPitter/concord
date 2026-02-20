<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    open: boolean
    title?: string
    children?: Snippet
  }

  let { open = $bindable(false), title, children }: Props = $props()

  let dialogEl: HTMLDialogElement | undefined = $state()

  $effect(() => {
    if (!dialogEl) return
    if (open) {
      dialogEl.showModal()
    } else {
      dialogEl.close()
    }
  })

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      open = false
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === dialogEl) {
      open = false
    }
  }
</script>

{#if open}
  <dialog
    bind:this={dialogEl}
    class="m-auto max-w-lg w-full rounded-lg border border-void-border bg-void-bg-secondary p-0 text-void-text-primary shadow-md backdrop:bg-black/60 backdrop:backdrop-blur-sm open:animate-[modal-in_150ms_ease-out]"
    onkeydown={handleKeydown}
    onclick={handleBackdropClick}
  >
    <div class="p-6">
      {#if title}
        <h2 class="mb-4 text-lg font-bold">{title}</h2>
      {/if}
      {#if children}
        {@render children()}
      {/if}
    </div>
  </dialog>
{/if}

<style>
  @keyframes modal-in {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
