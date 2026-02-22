<script lang="ts">
  import { translations, t } from '../../i18n'

  let {
    attachment,
    onDownload,
    onDelete,
    canDelete = false,
  }: {
    attachment: {
      id: string
      filename: string
      size_bytes: number
      mime_type: string
    }
    onDownload?: (id: string) => void
    onDelete?: (id: string) => void
    canDelete?: boolean
  } = $props()

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  const trans = $derived($translations)
  const isImage = $derived(attachment.mime_type.startsWith('image/'))
  const isAudio = $derived(attachment.mime_type.startsWith('audio/'))
  const isVideo = $derived(attachment.mime_type.startsWith('video/'))

  function getIcon(): string {
    if (isImage) return 'image'
    if (isAudio) return 'audio'
    if (isVideo) return 'video'
    if (attachment.mime_type === 'application/pdf') return 'pdf'
    if (attachment.mime_type.startsWith('text/')) return 'text'
    if (attachment.mime_type.includes('zip') || attachment.mime_type.includes('tar') || attachment.mime_type.includes('7z') || attachment.mime_type.includes('gzip')) return 'archive'
    return 'file'
  }
</script>

<div class="group mt-1 inline-flex items-center gap-2 rounded-lg border border-void-border bg-void-bg-secondary px-3 py-2 transition-colors hover:bg-void-bg-tertiary">
  <!-- File icon -->
  <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-void-accent/10 text-void-accent">
    {#if getIcon() === 'image'}
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
        <circle cx="8.5" cy="8.5" r="1.5" />
        <polyline points="21,15 16,10 5,21" />
      </svg>
    {:else if getIcon() === 'audio'}
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path d="M9 18V5l12-2v13" />
        <circle cx="6" cy="18" r="3" />
        <circle cx="18" cy="16" r="3" />
      </svg>
    {:else if getIcon() === 'video'}
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <polygon points="23,7 16,12 23,17" />
        <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
      </svg>
    {:else if getIcon() === 'pdf'}
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
        <polyline points="14,2 14,8 20,8" />
        <line x1="16" y1="13" x2="8" y2="13" />
        <line x1="16" y1="17" x2="8" y2="17" />
      </svg>
    {:else if getIcon() === 'archive'}
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <polyline points="21,8 21,21 3,21 3,8" />
        <rect x="1" y="3" width="22" height="5" />
        <line x1="10" y1="12" x2="14" y2="12" />
      </svg>
    {:else}
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
        <polyline points="14,2 14,8 20,8" />
      </svg>
    {/if}
  </div>

  <!-- File info -->
  <div class="min-w-0 flex-1">
    <p class="truncate text-sm font-medium text-void-accent">{attachment.filename}</p>
    <p class="text-xs text-void-text-muted">{formatSize(attachment.size_bytes)}</p>
  </div>

  <!-- Action buttons -->
  <div class="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
    {#if onDownload}
      <button
        class="rounded p-1 text-void-text-muted transition-colors hover:bg-void-bg-primary hover:text-void-text-primary"
        onclick={() => onDownload?.(attachment.id)}
        aria-label={t(trans, 'chat.downloadFile')}
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7,10 12,15 17,10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
      </button>
    {/if}
    {#if canDelete && onDelete}
      <button
        class="rounded p-1 text-void-text-muted transition-colors hover:bg-void-danger/20 hover:text-void-danger"
        onclick={() => onDelete?.(attachment.id)}
        aria-label={t(trans, 'chat.deleteFile')}
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      </button>
    {/if}
  </div>
</div>
