<script lang="ts">
  import MessageList from '../chat/MessageList.svelte'
  import MessageInput from '../chat/MessageInput.svelte'
  import type { MessageData, AttachmentData } from '../../stores/chat.svelte'

  let {
    channelName = 'general',
    messages = [],
    currentUserId = '',
    loading = false,
    hasMore = false,
    sending = false,
    attachmentsByMessage = {},
    onSend,
    onLoadMore,
    onEdit,
    onDelete,
    onFileSelect,
    onDownloadFile,
    onDeleteFile,
  }: {
    channelName?: string
    messages?: MessageData[]
    currentUserId?: string
    loading?: boolean
    hasMore?: boolean
    sending?: boolean
    attachmentsByMessage?: Record<string, AttachmentData[]>
    onSend: (content: string) => void
    onLoadMore?: () => void
    onEdit?: (id: string) => void
    onDelete?: (id: string) => void
    onFileSelect?: (file: { name: string; data: number[] }) => void
    onDownloadFile?: (id: string) => void
    onDeleteFile?: (id: string) => void
  } = $props()
</script>

<main class="flex h-full flex-1 flex-col bg-void-bg-tertiary">
  <!-- Channel header -->
  <header class="flex h-12 items-center gap-2 border-b border-void-border px-4 shrink-0">
    <svg class="h-5 w-5 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <line x1="4" y1="9" x2="20" y2="9" />
      <line x1="4" y1="15" x2="20" y2="15" />
      <line x1="10" y1="3" x2="8" y2="21" />
      <line x1="16" y1="3" x2="14" y2="21" />
    </svg>
    <span class="font-bold text-void-text-primary">{channelName}</span>
    <div class="mx-2 h-6 w-px bg-void-border"></div>
    <span class="text-sm text-void-text-muted">Welcome to #{channelName}</span>

    <!-- Header actions -->
    <div class="ml-auto flex items-center gap-2">
      <button aria-label="Search" class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:text-void-text-primary cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
      </button>
      <button aria-label="Members" class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:text-void-text-primary cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
          <circle cx="9" cy="7" r="4" />
          <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
          <path d="M16 3.13a4 4 0 0 1 0 7.75" />
        </svg>
      </button>
    </div>
  </header>

  <!-- Messages area -->
  <MessageList
    {messages}
    {currentUserId}
    {channelName}
    {loading}
    {hasMore}
    {attachmentsByMessage}
    {onLoadMore}
    {onEdit}
    {onDelete}
    {onDownloadFile}
    {onDeleteFile}
  />

  <!-- Message input -->
  <MessageInput
    {channelName}
    disabled={sending}
    {onSend}
    {onFileSelect}
  />
</main>
