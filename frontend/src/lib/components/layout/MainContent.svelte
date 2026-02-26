<script lang="ts">
  import MessageList from '../chat/MessageList.svelte'
  import MessageInput from '../chat/MessageInput.svelte'
  import Skeleton from '../ui/Skeleton.svelte'
  import { translations, t } from '../../i18n'
  import type { MessageData, AttachmentData, SearchResultData } from '../../stores/chat.svelte'

  let {
    channelName = 'general',
    messages = [],
    currentUserId = '',
    loading = false,
    hasMore = false,
    sending = false,
    attachmentsByMessage = {},
    membersVisible = false,
    searchResults = [],
    searchQuery = '',
    onSend,
    onLoadMore,
    onEdit,
    onDelete,
    canDeleteOthers = false,
    onFileSelect,
    onDownloadFile,
    onDeleteFile,
    onToggleMembers,
    onSearch,
    onClearSearch,
  }: {
    channelName?: string
    messages?: MessageData[]
    currentUserId?: string
    loading?: boolean
    hasMore?: boolean
    sending?: boolean
    attachmentsByMessage?: Record<string, AttachmentData[]>
    membersVisible?: boolean
    searchResults?: SearchResultData[]
    searchQuery?: string
    onSend: (content: string) => void
    onLoadMore?: () => void
    onEdit?: (id: string) => void
    onDelete?: (id: string) => void
    canDeleteOthers?: boolean
    onFileSelect?: (file: { name: string; data: number[] }) => void
    onDownloadFile?: (id: string) => void
    onDeleteFile?: (id: string) => void
    onToggleMembers?: () => void
    onSearch?: (query: string) => void
    onClearSearch?: () => void
  } = $props()

  let showSearch = $state(false)
  let searchInput = $state('')
  let searchInputEl: HTMLInputElement | undefined = $state()
  const trans = $derived($translations)

  function toggleSearch() {
    showSearch = !showSearch
    if (showSearch) {
      setTimeout(() => searchInputEl?.focus(), 50)
    } else {
      searchInput = ''
      onClearSearch?.()
    }
  }

  function handleSearch() {
    const q = searchInput.trim()
    if (!q) return
    onSearch?.(q)
  }
</script>

<main class="flex h-full flex-1 flex-col bg-void-bg-tertiary">
  <!-- Channel header -->
  <header class="flex h-12 items-center gap-2 border-b border-void-border px-4 shrink-0">
    {#if loading}
      <Skeleton className="h-5 w-5 rounded-sm" />
      <Skeleton className="h-4 w-28 rounded-md" />
      <div class="mx-2 h-6 w-px bg-void-border"></div>
      <Skeleton className="h-3.5 w-56 rounded-md" />
      <div class="ml-auto flex items-center gap-2">
        <Skeleton className="h-8 w-8 rounded-md" />
        <Skeleton className="h-8 w-8 rounded-md" />
      </div>
    {:else}
      <svg class="h-5 w-5 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="4" y1="9" x2="20" y2="9" />
        <line x1="4" y1="15" x2="20" y2="15" />
        <line x1="10" y1="3" x2="8" y2="21" />
        <line x1="16" y1="3" x2="14" y2="21" />
      </svg>
      <span class="font-bold text-void-text-primary">{channelName}</span>
      <div class="mx-2 h-6 w-px bg-void-border"></div>
      <span class="text-sm text-void-text-muted">{t(trans, 'chat.welcomeChannel', { channel: channelName })}</span>

      <!-- Header actions -->
      <div class="ml-auto flex items-center gap-2">
        <button
          aria-label="Search"
          class="rounded-md p-1.5 transition-colors cursor-pointer {showSearch ? 'text-void-text-primary bg-void-bg-hover' : 'text-void-text-secondary hover:text-void-text-primary'}"
          onclick={toggleSearch}
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
        </button>
        <button
          aria-label="Members"
          class="rounded-md p-1.5 transition-colors cursor-pointer {membersVisible ? 'text-void-text-primary bg-void-bg-hover' : 'text-void-text-secondary hover:text-void-text-primary'}"
          onclick={() => onToggleMembers?.()}
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
        </button>
      </div>
    {/if}
  </header>

  <!-- Search bar (slides down when active) -->
  {#if !loading && showSearch}
    <div class="flex items-center gap-2 border-b border-void-border px-4 py-2 bg-void-bg-secondary shrink-0 animate-fade-in-down">
      <svg class="h-4 w-4 text-void-text-muted shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
      <input
        bind:this={searchInputEl}
        bind:value={searchInput}
        type="text"
        placeholder={t(trans, 'chat.searchMessages')}
        class="flex-1 bg-transparent text-sm text-void-text-primary placeholder:text-void-text-muted outline-none"
        onkeydown={(e) => { if (e.key === 'Enter') handleSearch(); if (e.key === 'Escape') toggleSearch() }}
      />
      {#if searchInput}
        <button
          class="text-xs text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
          onclick={() => { searchInput = ''; onClearSearch?.() }}
        >
          {t(trans, 'chat.clear')}
        </button>
      {/if}
      <button
        class="rounded-md px-2.5 py-1 text-xs font-medium bg-void-accent text-white hover:bg-void-accent-hover transition-colors cursor-pointer disabled:opacity-50"
        onclick={handleSearch}
        disabled={!searchInput.trim()}
      >
        {t(trans, 'chat.search')}
      </button>
    </div>
  {/if}

  <!-- Search results overlay -->
  {#if !loading && showSearch && searchQuery}
    <div class="border-b border-void-border bg-void-bg-secondary px-4 py-2 max-h-60 overflow-y-auto shrink-0 animate-fade-in">
      {#if searchResults.length > 0}
        <p class="text-[11px] font-bold uppercase tracking-wide text-void-text-muted mb-2">
          {t(trans, 'chat.resultsFor', { count: String(searchResults.length), plural: searchResults.length !== 1 ? 's' : '', query: searchQuery })}
        </p>
        {#each searchResults as result}
          <div class="rounded-md px-3 py-2 mb-1 bg-void-bg-tertiary hover:bg-void-bg-hover transition-colors cursor-pointer">
            <div class="flex items-center gap-2 mb-1">
              <span class="text-xs font-semibold text-void-text-primary">{result.author_name}</span>
              <span class="text-[10px] text-void-text-muted">{new Date(result.created_at).toLocaleString()}</span>
            </div>
            <p class="text-sm text-void-text-secondary">{result.snippet || result.content}</p>
          </div>
        {/each}
      {:else}
        <div class="flex flex-col items-center gap-2 py-4 text-center">
          <svg class="h-10 w-10 text-void-text-muted opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="11" cy="11" r="8"/>
            <line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <p class="text-sm text-void-text-muted">{t(trans, 'chat.noResults', { query: searchQuery })}</p>
          <p class="text-xs text-void-text-muted">{t(trans, 'chat.noResultsHint')}</p>
        </div>
      {/if}
    </div>
  {/if}

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
    {canDeleteOthers}
    {onDownloadFile}
    {onDeleteFile}
  />

  <!-- Message input -->
  {#if loading && messages.length === 0}
    <div class="border-t border-void-border px-4 py-4">
      <div class="flex items-end gap-2 rounded-lg bg-void-bg-secondary px-4 py-3">
        <Skeleton className="h-5 w-5 rounded-sm" />
        <Skeleton className="h-5 flex-1 rounded-md" />
        <Skeleton className="h-5 w-5 rounded-sm" />
      </div>
    </div>
  {:else}
    <MessageInput
      {channelName}
      disabled={sending || loading}
      {onSend}
      {onFileSelect}
    />
  {/if}
</main>
