<script lang="ts">
  import { getToasts, removeToast } from '../../stores/toast.svelte'

  const toasts = getToasts()

  const iconMap = {
    success: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
    error: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z',
    warning: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z',
    info: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
  }

  const colorMap = {
    success: 'border-void-online text-void-online',
    error: 'border-void-danger text-void-danger',
    warning: 'border-yellow-500 text-yellow-500',
    info: 'border-void-accent text-void-accent',
  }
</script>

{#if toasts.list.length > 0}
  <div class="fixed bottom-4 right-4 z-[9999] flex flex-col gap-2 pointer-events-none">
    {#each toasts.list as toast (toast.id)}
      <div
        class="pointer-events-auto flex items-start gap-3 rounded-lg border bg-void-bg-secondary px-4 py-3 shadow-lg animate-[toast-in_200ms_ease-out] min-w-[300px] max-w-[420px] {colorMap[toast.type]}"
        role="alert"
      >
        <svg class="h-5 w-5 shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d={iconMap[toast.type]} />
        </svg>
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-void-text-primary">{toast.title}</p>
          {#if toast.message}
            <p class="mt-0.5 text-xs text-void-text-muted">{toast.message}</p>
          {/if}
        </div>
        <button
          class="shrink-0 rounded p-0.5 text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
          onclick={() => removeToast(toast.id)}
          aria-label="Dismiss"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}

<style>
  @keyframes toast-in {
    from {
      opacity: 0;
      transform: translateX(100%);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }
</style>
