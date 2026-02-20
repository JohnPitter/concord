<script lang="ts">
  interface Props {
    checked?: boolean
    disabled?: boolean
    label?: string
  }

  let { checked = $bindable(false), disabled = false, label }: Props = $props()

  function toggle() {
    if (!disabled) checked = !checked
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      toggle()
    }
  }
</script>

<label class="inline-flex items-center gap-3 {disabled ? 'opacity-50 pointer-events-none' : 'cursor-pointer'}">
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    aria-label={label ?? 'Toggle'}
    {disabled}
    class="relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors duration-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-void-accent cursor-pointer
      {checked ? 'bg-void-accent' : 'bg-void-bg-hover'}"
    onclick={toggle}
    onkeydown={handleKeydown}
  >
    <span
      class="pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200
        {checked ? 'translate-x-4.5' : 'translate-x-0.5'} mt-0.5"
    ></span>
  </button>
  {#if label}
    <span class="text-sm text-void-text-secondary select-none">{label}</span>
  {/if}
</label>
