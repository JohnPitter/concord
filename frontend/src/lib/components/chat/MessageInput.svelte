<script lang="ts">
  let {
    channelName = 'general',
    disabled = false,
    onSend,
  }: {
    channelName?: string
    disabled?: boolean
    onSend: (content: string) => void
  } = $props()

  let content = $state('')

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  function handleSend() {
    const trimmed = content.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    content = ''
  }
</script>

<div class="border-t border-void-border px-4 py-4">
  <div class="flex items-end gap-2 rounded-lg bg-void-bg-tertiary px-4 py-2">
    <!-- Attach button -->
    <button
      class="shrink-0 rounded p-1 text-void-text-muted transition-colors hover:text-void-text-primary"
      aria-label="Attach file"
    >
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
      </svg>
    </button>

    <!-- Text input -->
    <textarea
      class="max-h-32 min-h-[24px] flex-1 resize-none bg-transparent text-sm text-void-text-primary placeholder-void-text-muted outline-none"
      placeholder="Message #{channelName}"
      rows="1"
      bind:value={content}
      onkeydown={handleKeydown}
      {disabled}
    ></textarea>

    <!-- Emoji button -->
    <button
      class="shrink-0 rounded p-1 text-void-text-muted transition-colors hover:text-void-text-primary"
      aria-label="Emoji"
    >
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    </button>

    <!-- Send button (visible when content exists) -->
    {#if content.trim()}
      <button
        class="shrink-0 rounded-lg bg-void-accent p-1.5 text-white transition-colors hover:bg-void-accent-hover disabled:opacity-50"
        onclick={handleSend}
        {disabled}
        aria-label="Send message"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 5l7 7-7 7M5 12h14" />
        </svg>
      </button>
    {/if}
  </div>
</div>
