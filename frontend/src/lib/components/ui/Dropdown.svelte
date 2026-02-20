<script lang="ts">
  interface DropdownItem {
    label: string
    value: string
  }

  interface Props {
    items: DropdownItem[]
    selected?: string
    placeholder?: string
  }

  let { items, selected = $bindable(''), placeholder = 'Select...' }: Props = $props()

  let open = $state(false)
  let containerEl: HTMLDivElement | undefined = $state()
  let highlightedIndex = $state(-1)

  const displayLabel = $derived(
    items.find((i) => i.value === selected)?.label ?? placeholder
  )

  function select(item: DropdownItem) {
    selected = item.value
    open = false
    highlightedIndex = -1
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!open && (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown')) {
      e.preventDefault()
      open = true
      highlightedIndex = 0
      return
    }

    if (!open) return

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        highlightedIndex = (highlightedIndex + 1) % items.length
        break
      case 'ArrowUp':
        e.preventDefault()
        highlightedIndex = (highlightedIndex - 1 + items.length) % items.length
        break
      case 'Enter':
      case ' ':
        e.preventDefault()
        if (highlightedIndex >= 0) select(items[highlightedIndex])
        break
      case 'Escape':
        open = false
        highlightedIndex = -1
        break
    }
  }

  function handleClickOutside(e: MouseEvent) {
    if (containerEl && !containerEl.contains(e.target as Node)) {
      open = false
      highlightedIndex = -1
    }
  }
</script>

<svelte:document onclick={handleClickOutside} />

<div class="relative" bind:this={containerEl}>
  <button
    type="button"
    class="flex w-full items-center justify-between rounded-md border border-void-border bg-void-bg-secondary px-3 py-2 text-sm transition-colors hover:border-void-text-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-void-accent cursor-pointer
      {selected ? 'text-void-text-primary' : 'text-void-text-muted'}"
    aria-haspopup="listbox"
    aria-expanded={open}
    onclick={() => { open = !open; highlightedIndex = -1 }}
    onkeydown={handleKeydown}
  >
    <span>{displayLabel}</span>
    <svg class="h-4 w-4 text-void-text-muted transition-transform duration-150 {open ? 'rotate-180' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polyline points="6 9 12 15 18 9" />
    </svg>
  </button>

  {#if open}
    <ul
      class="absolute z-50 mt-1 w-full overflow-hidden rounded-md border border-void-border bg-void-bg-secondary py-1 shadow-md animate-[fade-in_100ms_ease-out]"
      role="listbox"
    >
      {#each items as item, i}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <li
          role="option"
          aria-selected={item.value === selected}
          class="cursor-pointer px-3 py-2 text-sm transition-colors
            {item.value === selected ? 'text-void-accent' : 'text-void-text-primary'}
            {i === highlightedIndex ? 'bg-void-bg-hover' : 'hover:bg-void-bg-hover'}"
          onclick={() => select(item)}
          onmouseenter={() => highlightedIndex = i}
        >
          {item.label}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  @keyframes fade-in {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
