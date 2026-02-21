<script lang="ts">
  interface P2PPeer {
    id: string
    displayName: string
    avatarDataUrl?: string
    connected: boolean
    source: 'lan' | 'room'
  }

  interface P2PMessage {
    id: string
    peerID: string
    direction: 'sent' | 'received'
    content: string
    sentAt: string
  }

  let {
    peer,
    messages,
    sending,
    onSend,
  }: {
    peer: P2PPeer | null
    messages: P2PMessage[]
    sending: boolean
    onSend: (content: string) => void
  } = $props()

  let inputValue = $state('')
  let messagesContainer: HTMLDivElement | undefined = $state()

  function handleSend() {
    const content = inputValue.trim()
    if (!content || sending) return
    onSend(content)
    inputValue = ''
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  function formatTime(iso: string): string {
    try {
      const d = new Date(iso)
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    } catch {
      return ''
    }
  }

  function peerHue(id: string): number {
    let hash = 0
    for (let i = 0; i < id.length; i++) {
      hash = id.charCodeAt(i) + ((hash << 5) - hash)
    }
    return Math.abs(hash) % 360
  }

  function peerInitials(p: P2PPeer): string {
    if (p.displayName) return p.displayName.slice(0, 2).toUpperCase()
    return p.id.slice(0, 2).toUpperCase()
  }

  function peerLabel(p: P2PPeer): string {
    return p.displayName || (p.id.length > 12 ? p.id.slice(0, 12) + '...' : p.id)
  }

  // Auto-scroll on new messages
  $effect(() => {
    if (messages.length && messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight
    }
  })
</script>

<main class="flex flex-1 flex-col bg-void-bg-primary">
  {#if !peer}
    <!-- Empty state -->
    <div class="flex flex-1 flex-col items-center justify-center gap-3">
      <svg class="h-16 w-16 text-void-text-muted/30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="8" cy="12" r="3"/>
        <circle cx="16" cy="12" r="3"/>
        <line x1="11" y1="12" x2="13" y2="12"/>
      </svg>
      <p class="text-void-text-muted text-sm">Selecione um peer para conversar</p>
    </div>
  {:else}
    <!-- Header -->
    <header class="flex h-12 items-center gap-2 border-b border-void-border px-4 shrink-0">
      {#if peer.avatarDataUrl}
        <img src={peer.avatarDataUrl} alt={peerLabel(peer)} class="h-6 w-6 rounded-full object-cover" />
      {:else}
        <div
          class="h-6 w-6 rounded-full flex items-center justify-center text-[10px] font-bold text-white"
          style="background-color: hsl({peerHue(peer.id)}, 60%, 45%)"
        >
          {peerInitials(peer)}
        </div>
      {/if}
      <span class="font-bold text-void-text-primary text-sm">{peerLabel(peer)}</span>
      {#if peer.source === 'lan'}
        <span class="rounded px-1.5 py-0.5 text-[10px] font-bold bg-void-online/20 text-void-online">LAN</span>
      {:else}
        <span class="rounded px-1.5 py-0.5 text-[10px] font-bold bg-blue-500/20 text-blue-400">WAN</span>
      {/if}
    </header>

    <!-- Messages -->
    <div bind:this={messagesContainer} class="flex-1 overflow-y-auto p-4 flex flex-col gap-2">
      {#if messages.length === 0}
        <div class="flex flex-1 items-center justify-center">
          <p class="text-void-text-muted text-xs">Nenhuma mensagem ainda. Diga oi!</p>
        </div>
      {:else}
        {#each messages as msg (msg.id)}
          <div class="flex {msg.direction === 'sent' ? 'justify-end' : 'justify-start'}">
            <div class="max-w-[70%]">
              <div class="px-3 py-2 text-sm break-words
                {msg.direction === 'sent'
                  ? 'bg-void-accent text-white rounded-2xl rounded-tr-sm'
                  : 'bg-void-bg-secondary text-void-text-primary rounded-2xl rounded-tl-sm'}"
              >
                {msg.content}
              </div>
              <p class="mt-0.5 text-[10px] text-void-text-muted {msg.direction === 'sent' ? 'text-right' : 'text-left'}">
                {formatTime(msg.sentAt)}
              </p>
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <!-- Input -->
    <div class="border-t border-void-border p-3 shrink-0">
      <div class="flex items-end gap-2">
        <textarea
          bind:value={inputValue}
          onkeydown={handleKeydown}
          placeholder="Enviar mensagem para {peerLabel(peer)}"
          rows="1"
          class="flex-1 resize-none rounded-lg bg-void-bg-secondary px-3 py-2 text-sm text-void-text-primary placeholder:text-void-text-muted outline-none focus:ring-1 focus:ring-void-accent"
        ></textarea>
        <button
          class="shrink-0 rounded-lg bg-void-accent p-2 text-white hover:bg-void-accent-hover transition-colors cursor-pointer disabled:opacity-50 disabled:pointer-events-none"
          onclick={handleSend}
          disabled={sending || !inputValue.trim()}
          aria-label="Enviar mensagem"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="22" y1="2" x2="11" y2="13"/>
            <polygon points="22 2 15 22 11 13 2 9 22 2"/>
          </svg>
        </button>
      </div>
    </div>
  {/if}
</main>
