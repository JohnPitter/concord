<script lang="ts">
  import Avatar from '../ui/Avatar.svelte'
  import FileAttachment from './FileAttachment.svelte'
  import type { MessageData, AttachmentData } from '../../stores/chat.svelte'

  let {
    message,
    isOwn = false,
    showAvatar = true,
    attachments = [],
    onEdit,
    onDelete,
    onDownloadFile,
    onDeleteFile,
  }: {
    message: MessageData
    isOwn?: boolean
    showAvatar?: boolean
    attachments?: AttachmentData[]
    onEdit?: (id: string) => void
    onDelete?: (id: string) => void
    onDownloadFile?: (id: string) => void
    onDeleteFile?: (id: string) => void
  } = $props()

  let showActions = $state(false)

  function formatTime(dateStr: string): string {
    const date = new Date(dateStr)
    const now = new Date()
    const isToday = date.toDateString() === now.toDateString()

    if (isToday) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }

    const yesterday = new Date(now)
    yesterday.setDate(now.getDate() - 1)
    if (date.toDateString() === yesterday.toDateString()) {
      return `Yesterday ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
    }

    return date.toLocaleDateString([], { month: 'short', day: 'numeric' }) +
      ` ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="group flex gap-4 px-4 py-1 transition-colors hover:bg-void-bg-hover/30"
  onmouseenter={() => showActions = true}
  onmouseleave={() => showActions = false}
>
  <!-- Avatar column -->
  <div class="w-10 shrink-0 pt-0.5">
    {#if showAvatar}
      <Avatar name={message.author_name} src={message.author_avatar || undefined} size="sm" />
    {/if}
  </div>

  <!-- Content column -->
  <div class="min-w-0 flex-1">
    {#if showAvatar}
      <div class="flex items-baseline gap-2">
        <span class="text-sm font-medium text-void-text-primary">{message.author_name}</span>
        <span class="text-xs text-void-text-muted">{formatTime(message.created_at)}</span>
        {#if message.edited_at}
          <span class="text-xs text-void-text-muted">(edited)</span>
        {/if}
      </div>
    {/if}

    <p class="text-sm leading-relaxed text-void-text-secondary break-words whitespace-pre-wrap">{message.content}</p>

    {#if attachments.length > 0}
      <div class="mt-1 flex flex-wrap gap-1">
        {#each attachments as att (att.id)}
          <FileAttachment
            attachment={att}
            onDownload={onDownloadFile}
            onDelete={onDeleteFile}
            canDelete={isOwn}
          />
        {/each}
      </div>
    {/if}
  </div>

  <!-- Action buttons -->
  {#if showActions && (isOwn || onDelete)}
    <div class="flex shrink-0 items-start gap-1 pt-0.5">
      {#if isOwn && onEdit}
        <button
          class="rounded p-1 text-void-text-muted transition-colors hover:bg-void-bg-tertiary hover:text-void-text-primary"
          onclick={() => onEdit?.(message.id)}
          aria-label="Edit message"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
        </button>
      {/if}
      {#if onDelete}
        <button
          class="rounded p-1 text-void-text-muted transition-colors hover:bg-void-danger/20 hover:text-void-danger"
          onclick={() => onDelete?.(message.id)}
          aria-label="Delete message"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      {/if}
    </div>
  {/if}
</div>
