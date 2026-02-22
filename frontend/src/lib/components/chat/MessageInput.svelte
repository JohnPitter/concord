<script lang="ts">
  import { translations, t } from '../../i18n'

  let {
    channelName = 'general',
    disabled = false,
    onSend,
    onFileSelect,
  }: {
    channelName?: string
    disabled?: boolean
    onSend: (content: string) => void
    onFileSelect?: (file: { name: string; data: number[] }) => void
  } = $props()

  let content = $state('')
  let pendingFile = $state<{ name: string; data: number[] } | null>(null)
  let fileInput: HTMLInputElement | undefined = $state()
  let showEmoji = $state(false)
  const trans = $derived($translations)

  const emojiCategoryKeys = ['chat.emojiSmileys', 'chat.emojiGestures', 'chat.emojiObjects', 'chat.emojiAnimals']
  const emojiSets = [
    ['😀','😂','😍','🥺','😎','🤔','😅','😢','😡','🥳','😱','🤩','😴','🤗','😏','🙄','😬','🤯','🥴','😈'],
    ['👍','👎','👏','🙏','✌️','🤞','🤟','👌','🤙','💪','👀','🫡','🫶','🤝','👋','✋','🖐️','🤚','🫰','🫳'],
    ['❤️','🔥','⭐','💯','🎉','🎮','💀','✅','❌','⚡','💡','🚀','🏆','🎯','💎','🔔','📌','💬','🎵','🎶'],
    ['🐱','🐶','🐻','🦊','🐼','🐸','🦁','🐧','🐵','🐝','🦋','🐙','🐰','🐮','🐔','🦄','🐺','🦇','🐍','🐠'],
  ]

  function insertEmoji(emoji: string) {
    content += emoji
    showEmoji = false
  }

  function handleWindowClick(e: MouseEvent) {
    if (showEmoji) {
      const target = e.target as HTMLElement
      if (!target.closest('.emoji-picker-container')) {
        showEmoji = false
      }
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  function handleSend() {
    const trimmed = content.trim()
    if (!trimmed && !pendingFile) return
    if (disabled) return

    if (pendingFile && onFileSelect) {
      onFileSelect(pendingFile)
      pendingFile = null
    }
    if (trimmed) {
      onSend(trimmed)
    }
    content = ''
  }

  function handleAttachClick() {
    fileInput?.click()
  }

  async function handleFileChange(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files?.[0]
    if (!file) return

    const buffer = await file.arrayBuffer()
    const data = Array.from(new Uint8Array(buffer))
    pendingFile = { name: file.name, data }

    // Reset input so the same file can be selected again
    input.value = ''
  }

  function removePendingFile() {
    pendingFile = null
  }
</script>

<svelte:window onclick={handleWindowClick} />

<div class="border-t border-void-border px-4 py-4">
  <!-- Hidden file input -->
  <input
    bind:this={fileInput}
    type="file"
    class="hidden"
    onchange={handleFileChange}
  />

  {#if pendingFile}
    <div class="mb-2 flex items-center gap-2 rounded-lg bg-void-bg-secondary px-3 py-2 text-sm">
      <svg class="h-4 w-4 shrink-0 text-void-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
        <polyline points="14,2 14,8 20,8" />
      </svg>
      <span class="min-w-0 truncate text-void-text-secondary">{pendingFile.name}</span>
      <button
        class="ml-auto shrink-0 rounded p-0.5 text-void-text-muted transition-colors hover:text-void-danger"
        onclick={removePendingFile}
        aria-label={t(trans, 'chat.removeFile')}
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </button>
    </div>
  {/if}

  <div class="flex items-end gap-2 rounded-lg bg-void-bg-tertiary px-4 py-2">
    <!-- Attach button -->
    <button
      class="shrink-0 rounded p-1 text-void-text-muted transition-colors hover:text-void-text-primary"
      onclick={handleAttachClick}
      aria-label={t(trans, 'chat.attachFile')}
    >
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
      </svg>
    </button>

    <!-- Text input -->
    <textarea
      class="max-h-32 min-h-[24px] flex-1 resize-none bg-transparent text-sm text-void-text-primary placeholder-void-text-muted outline-none"
      placeholder={t(trans, 'chat.messagePlaceholder', { channel: channelName })}
      rows="1"
      bind:value={content}
      onkeydown={handleKeydown}
      {disabled}
    ></textarea>

    <!-- Emoji button -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div class="relative emoji-picker-container">
      <button
        class="shrink-0 rounded p-1 transition-colors cursor-pointer {showEmoji ? 'text-void-accent' : 'text-void-text-muted hover:text-void-text-primary'}"
        aria-label={t(trans, 'chat.emoji')}
        onclick={() => showEmoji = !showEmoji}
      >
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </button>
      {#if showEmoji}
        <div class="absolute bottom-full right-0 mb-2 w-72 rounded-lg border border-void-border bg-void-bg-primary shadow-md p-2 z-50 animate-fade-in-up">
          <div class="max-h-56 overflow-y-auto space-y-2">
            {#each emojiCategoryKeys as catKey, ci}
              <div>
                <p class="text-[10px] font-bold uppercase tracking-wide text-void-text-muted px-1 mb-1">{t(trans, catKey)}</p>
                <div class="grid grid-cols-8 gap-0.5">
                  {#each emojiSets[ci] as emoji}
                    <button
                      class="flex items-center justify-center rounded p-1 text-lg hover:bg-void-bg-hover transition-colors cursor-pointer"
                      onclick={() => insertEmoji(emoji)}
                    >
                      {emoji}
                    </button>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <!-- Send button (visible when content or file exists) -->
    {#if content.trim() || pendingFile}
      <button
        class="shrink-0 rounded-lg bg-void-accent p-1.5 text-white transition-colors hover:bg-void-accent-hover disabled:opacity-50"
        onclick={handleSend}
        {disabled}
        aria-label={t(trans, 'chat.sendMessage')}
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 5l7 7-7 7M5 12h14" />
        </svg>
      </button>
    {/if}
  </div>
</div>
