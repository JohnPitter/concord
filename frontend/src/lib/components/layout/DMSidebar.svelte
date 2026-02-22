<script lang="ts">
  import type { DMConversation } from '../../stores/friends.svelte'
  import type { SpeakerData } from '../../stores/voice.svelte'
  import VoiceControls from '../voice/VoiceControls.svelte'
  import Tooltip from '../ui/Tooltip.svelte'
  import { translations, t } from '../../i18n'

  interface CurrentUser {
    username: string
    display_name: string
    avatar_url: string
  }

  let {
    dms,
    activeDMId = null,
    onSelectDM,
    onOpenFriends,
    currentUser = null,
    voiceConnected = false,
    voiceChannelName = '',
    voiceMuted = false,
    voiceDeafened = false,
    voiceSpeakers = [],
    voiceNoiseSuppression = true,
    voiceScreenSharing = false,
    onToggleMute,
    onToggleDeafen,
    onToggleNoiseSuppression,
    onToggleScreenShare,
    onLeaveVoice,
    onOpenSettings,
  }: {
    dms: DMConversation[]
    activeDMId?: string | null
    onSelectDM: (id: string | null) => void
    onOpenFriends: () => void
    currentUser?: CurrentUser | null
    voiceConnected?: boolean
    voiceChannelName?: string
    voiceMuted?: boolean
    voiceDeafened?: boolean
    voiceSpeakers?: SpeakerData[]
    voiceNoiseSuppression?: boolean
    voiceScreenSharing?: boolean
    onToggleMute?: () => void
    onToggleDeafen?: () => void
    onToggleNoiseSuppression?: () => void
    onToggleScreenShare?: () => void
    onLeaveVoice?: () => void
    onOpenSettings?: () => void
  } = $props()

  const displayName = $derived(currentUser?.display_name || currentUser?.username || 'You')
  const initials = $derived(displayName.slice(0, 2).toUpperCase())
  const trans = $derived($translations)

  const statusColor: Record<string, string> = {
    online: 'bg-void-online',
    idle:   'bg-void-idle',
    dnd:    'bg-void-danger',
    offline: 'bg-void-text-muted',
  }

  let searchQuery = $state('')
  let searching = $state(false)

  const filteredDms = $derived(
    searchQuery.trim()
      ? dms.filter(dm => dm.display_name.toLowerCase().includes(searchQuery.toLowerCase()))
      : dms
  )
</script>

<aside class="flex h-full w-60 flex-col bg-void-bg-secondary">
  <!-- Search bar -->
  <div class="px-2 pt-3 pb-2 shrink-0">
    {#if searching}
      <div class="flex items-center gap-2 rounded-md bg-void-bg-primary px-2 py-1.5">
        <svg class="h-3.5 w-3.5 shrink-0 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input
          type="text"
          bind:value={searchQuery}
          placeholder={t(trans, 'nav.searchPlaceholder')}
          class="flex-1 bg-transparent text-xs text-void-text-primary placeholder:text-void-text-muted outline-none"
          onkeydown={(e) => { if (e.key === 'Escape') { searching = false; searchQuery = '' } }}
        />
        {#if searchQuery}
          <button
            class="text-void-text-muted hover:text-void-text-primary cursor-pointer"
            onclick={() => { searchQuery = '' }}
          >
            <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        {/if}
      </div>
    {:else}
      <button
        class="flex w-full items-center gap-2 rounded-md bg-void-bg-primary px-2 py-1.5 text-xs text-void-text-muted cursor-text hover:bg-void-bg-primary/80 transition-colors"
        onclick={() => searching = true}
        aria-label={t(trans, 'nav.search')}
      >
        <svg class="h-3.5 w-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <span>{t(trans, 'nav.search')}</span>
      </button>
    {/if}
  </div>

  <div class="flex-1 overflow-y-auto">
    <!-- Friends button -->
    <div class="px-2 mb-1">
      <button
        class="flex w-full items-center gap-3 rounded-md px-2 py-2 text-sm font-medium transition-colors cursor-pointer
          {activeDMId === null
            ? 'bg-void-bg-hover text-void-text-primary'
            : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
        onclick={onOpenFriends}
      >
        <svg class="h-5 w-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
          <circle cx="9" cy="7" r="4"/>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
        </svg>
        {t(trans, 'nav.friends')}
      </button>
    </div>

    <!-- DM section label -->
    <div class="flex items-center justify-between px-4 pt-3 pb-1">
      <span class="text-[11px] font-bold uppercase tracking-wide text-void-text-muted">{t(trans, 'nav.directMessages')}</span>
      <Tooltip text={t(trans, 'nav.newDM')} position="top">
        <button aria-label={t(trans, 'nav.newDM')} class="text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer">
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </button>
      </Tooltip>
    </div>

    <!-- DM list -->
    {#each filteredDms as dm}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div class="px-2">
        <div
          class="group flex w-full items-center gap-2 rounded-md px-2 py-1.5 transition-colors cursor-pointer
            {activeDMId === dm.id
              ? 'bg-void-bg-hover text-void-text-primary'
              : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
          onclick={() => onSelectDM(dm.id)}
        >
          <!-- Avatar with status -->
          <div class="relative shrink-0">
            {#if dm.avatar_url}
              <img src={dm.avatar_url} alt={dm.display_name} class="h-8 w-8 rounded-full object-cover" />
            {:else}
              <div class="h-8 w-8 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
                {dm.display_name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            <span class="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-void-bg-secondary {statusColor[dm.status] ?? 'bg-void-text-muted'}"></span>
          </div>

          <div class="flex-1 min-w-0 text-left">
            <p class="truncate text-sm font-medium leading-tight">{dm.display_name}</p>
            {#if dm.lastMessage}
              <p class="truncate text-[11px] text-void-text-muted leading-tight">{dm.lastMessage}</p>
            {/if}
          </div>

          {#if dm.unread}
            <span class="flex h-4 min-w-4 items-center justify-center rounded-full bg-void-danger px-1 text-[10px] font-bold text-white">
              {dm.unread}
            </span>
          {/if}

          <!-- Close button on hover -->
          <button
            class="hidden group-hover:flex h-4 w-4 shrink-0 items-center justify-center rounded text-void-text-muted hover:text-void-text-primary transition-colors"
            onclick={(e) => { e.stopPropagation(); }}
            aria-label={t(trans, 'nav.closeDM')}
          >
            <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
      </div>
    {/each}
  </div>

  <!-- Voice controls (when in voice) -->
  <VoiceControls
    connected={voiceConnected}
    channelName={voiceChannelName}
    muted={voiceMuted}
    deafened={voiceDeafened}
    noiseSuppression={voiceNoiseSuppression}
    screenSharing={voiceScreenSharing}
    speakers={voiceSpeakers}
    onToggleMute={() => onToggleMute?.()}
    onToggleDeafen={() => onToggleDeafen?.()}
    onToggleNoiseSuppression={() => onToggleNoiseSuppression?.()}
    onToggleScreenShare={() => onToggleScreenShare?.()}
    onDisconnect={() => onLeaveVoice?.()}
  />

  <!-- User panel -->
  <div class="border-t border-void-border bg-void-bg-primary p-2 shrink-0">
    <div class="flex items-center gap-2 rounded-md px-2 py-1.5">
      {#if currentUser?.avatar_url}
        <img src={currentUser.avatar_url} alt={displayName} class="h-8 w-8 shrink-0 rounded-full object-cover" />
      {:else}
        <div class="h-8 w-8 shrink-0 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
          {initials}
        </div>
      {/if}
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-void-text-primary truncate">{displayName}</p>
        <p class="text-[11px] text-void-online">{t(trans, 'common.online')}</p>
      </div>
      <Tooltip text={t(trans, 'nav.settings')} position="top">
        <button
          aria-label={t(trans, 'nav.settings')}
          class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
          onclick={() => onOpenSettings?.()}
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
        </button>
      </Tooltip>
    </div>
  </div>
</aside>
