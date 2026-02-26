<script lang="ts">
  import MessageBubble from './MessageBubble.svelte'
  import Skeleton from '../ui/Skeleton.svelte'
  import { translations, t } from '../../i18n'
  import type { MessageData, AttachmentData } from '../../stores/chat.svelte'

  let {
    messages,
    currentUserId,
    channelName = 'general',
    loading = false,
    hasMore = false,
    attachmentsByMessage = {},
    onLoadMore,
    onEdit,
    onDelete,
    canDeleteOthers = false,
    onDownloadFile,
    onDeleteFile,
  }: {
    messages: MessageData[]
    currentUserId: string
    channelName?: string
    loading?: boolean
    hasMore?: boolean
    attachmentsByMessage?: Record<string, AttachmentData[]>
    onLoadMore?: () => void
    onEdit?: (id: string) => void
    onDelete?: (id: string) => void
    canDeleteOthers?: boolean
    onDownloadFile?: (id: string) => void
    onDeleteFile?: (id: string) => void
  } = $props()

  const trans = $derived($translations)
  let scrollContainer: HTMLDivElement | undefined = $state()
  let wasAtBottom = $state(true)

  // Check if a message should show its avatar (first in a group from same author)
  function shouldShowAvatar(msg: MessageData, index: number): boolean {
    if (index === 0) return true
    const prev = messages[index - 1]
    if (prev.author_id !== msg.author_id) return true
    // Show avatar if more than 5 minutes between messages
    const prevTime = new Date(prev.created_at).getTime()
    const currTime = new Date(msg.created_at).getTime()
    return (currTime - prevTime) > 5 * 60 * 1000
  }

  // Auto-scroll to bottom when new messages arrive
  $effect(() => {
    if (messages.length > 0 && wasAtBottom && scrollContainer) {
      scrollContainer.scrollTop = scrollContainer.scrollHeight
    }
  })

  function handleScroll() {
    if (!scrollContainer) return
    const { scrollTop, scrollHeight, clientHeight } = scrollContainer
    wasAtBottom = scrollHeight - scrollTop - clientHeight < 50

    // Load more when scrolled to top
    if (scrollTop < 100 && hasMore && !loading && onLoadMore) {
      onLoadMore()
    }
  }
</script>

<div
  class="flex-1 overflow-y-auto"
  bind:this={scrollContainer}
  onscroll={handleScroll}
>
  {#if hasMore}
    <div class="flex justify-center py-4">
      {#if loading}
        <div class="h-5 w-5 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
      {:else}
        <button
          class="text-xs text-void-text-muted transition-colors hover:text-void-accent"
          onclick={() => onLoadMore?.()}
        >
          {t(trans, 'chat.loadOlder')}
        </button>
      {/if}
    </div>
  {/if}

  {#if loading && messages.length === 0}
    <div class="px-4 py-4">
      <div class="space-y-4">
        {#each Array.from({ length: 7 }) as _, i (`msg-sk-${i}`)}
          <div class="flex gap-3 {i % 3 === 2 ? 'justify-end' : ''}">
            {#if i % 3 !== 2}
              <Skeleton className="h-10 w-10 rounded-full" />
            {/if}
            <div class="max-w-[70%] min-w-[220px]">
              <Skeleton className="mb-1 h-3.5 w-24 rounded-md" />
              <Skeleton className="mb-1 h-3.5 w-full rounded-md" />
              <Skeleton className="h-3.5 w-3/4 rounded-md" />
            </div>
          </div>
        {/each}
      </div>
    </div>
  {:else if messages.length === 0 && !loading}
    <!-- Welcome message -->
    <div class="flex flex-col items-center justify-center px-4 py-16">
      <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-void-bg-tertiary">
        <svg class="h-8 w-8 text-void-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.076-4.076a1.526 1.526 0 011.037-.443 48.282 48.282 0 005.68-.494c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z" />
        </svg>
      </div>
      <h3 class="text-lg font-bold text-void-text-primary">{t(trans, 'chat.welcomeChannel', { channel: channelName })}</h3>
      <p class="mt-1 text-sm text-void-text-muted">{t(trans, 'chat.welcomeChannelHint')}</p>
    </div>
  {:else}
    <div class="py-2">
      {#each messages as message, index (message.id)}
        <MessageBubble
          {message}
          isOwn={message.author_id === currentUserId}
          showAvatar={shouldShowAvatar(message, index)}
          {canDeleteOthers}
          attachments={attachmentsByMessage[message.id] ?? []}
          {onEdit}
          {onDelete}
          {onDownloadFile}
          {onDeleteFile}
        />
      {/each}
    </div>
  {/if}
</div>
