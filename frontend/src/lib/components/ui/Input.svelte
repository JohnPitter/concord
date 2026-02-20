<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements'

  interface Props extends HTMLInputAttributes {
    error?: string
    label?: string
  }

  let {
    type = 'text',
    error,
    label,
    disabled = false,
    ...rest
  }: Props = $props()

  let showPassword = $state(false)
  const inputType = $derived(type === 'password' && showPassword ? 'text' : type)
</script>

<div class="flex flex-col gap-1.5">
  {#if label}
    <!-- svelte-ignore a11y_label_has_associated_control -->
    <label class="text-sm font-medium text-void-text-secondary">{label}</label>
  {/if}
  <div class="relative">
    <input
      type={inputType}
      class="w-full rounded-md border bg-void-bg-secondary px-3 py-2 text-sm text-void-text-primary placeholder:text-void-text-muted transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-void-accent disabled:pointer-events-none disabled:opacity-50
        {error ? 'border-void-danger focus:ring-void-danger' : 'border-void-border hover:border-void-text-muted focus:border-void-accent'}"
      {disabled}
      aria-invalid={!!error}
      aria-describedby={error ? 'input-error' : undefined}
      {...rest}
    />
    {#if type === 'password'}
      <button
        type="button"
        class="absolute right-2.5 top-1/2 -translate-y-1/2 text-void-text-muted hover:text-void-text-secondary transition-colors cursor-pointer"
        onclick={() => showPassword = !showPassword}
        aria-label={showPassword ? 'Hide password' : 'Show password'}
      >
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          {#if showPassword}
            <path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19m-6.72-1.07a3 3 0 11-4.24-4.24" />
            <line x1="1" y1="1" x2="23" y2="23" />
          {:else}
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
            <circle cx="12" cy="12" r="3" />
          {/if}
        </svg>
      </button>
    {/if}
  </div>
  {#if error}
    <p id="input-error" class="text-xs text-void-danger" role="alert">{error}</p>
  {/if}
</div>
