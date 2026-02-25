<script lang="ts">
  import Skeleton from '../ui/Skeleton.svelte'
  import Tooltip from '../ui/Tooltip.svelte'
  import VoiceControls from '../voice/VoiceControls.svelte'
  import { translations, t } from '../../i18n'
  import type { SpeakerData } from '../../stores/voice.svelte'
  import type { VoiceDiagnosticsSnapshot, VoiceScreenShare } from '../../services/voiceRTC'

  interface Channel {
    id: string
    name: string
    type: 'text' | 'voice'
    unreadCount?: number
  }

  interface ServerMember {
    user_id: string
    username: string
    avatar_url: string
    role: 'owner' | 'admin' | 'moderator' | 'member'
  }

  interface CurrentUser {
    id?: string
    username: string
    display_name: string
    avatar_url: string
  }

  let {
    serverName,
    channels,
    activeChannelId,
    onSelectChannel,
    onCreateChannel,
    onDeleteChannel,
    onServerInfo,
    loading = false,
    currentUser = null,
    serverMembers = [],
    currentUserRole = 'member',
    voiceConnected = false,
    voiceChannelName = '',
    voiceMuted = false,
    voiceDeafened = false,
    voiceSpeakers = [],
    voiceChannelId = null,
    voiceElapsed = '',
    voiceNoiseSuppression = true,
    voiceScreenSharing = false,
    voiceScreenShares = [],
    voiceDiagnostics = null,
    voiceLocalSpeaking = false,
    getChannelParticipants,
    onJoinVoice,
    onLeaveVoice,
    onToggleMute,
    onToggleDeafen,
    onToggleNoiseSuppression,
    onToggleScreenShare,
    onOpenSettings,
  }: {
    serverName: string
    channels: Channel[]
    activeChannelId: string
    onSelectChannel: (id: string) => void
    onCreateChannel?: (name: string, type: 'text' | 'voice') => void
    onDeleteChannel?: (channelId: string) => void
    onServerInfo?: () => void
    loading?: boolean
    currentUser?: CurrentUser | null
    serverMembers?: ServerMember[]
    currentUserRole?: string
    voiceConnected?: boolean
    voiceChannelName?: string
    voiceMuted?: boolean
    voiceDeafened?: boolean
    voiceSpeakers?: SpeakerData[]
    voiceChannelId?: string | null
    voiceElapsed?: string
    voiceNoiseSuppression?: boolean
    voiceScreenSharing?: boolean
    voiceScreenShares?: VoiceScreenShare[]
    voiceDiagnostics?: VoiceDiagnosticsSnapshot | null
    voiceLocalSpeaking?: boolean
    getChannelParticipants?: (channelId: string) => SpeakerData[]
    onJoinVoice?: (channelId: string) => void
    onLeaveVoice?: () => void
    onToggleMute?: () => void
    onToggleDeafen?: () => void
    onToggleNoiseSuppression?: () => void
    onToggleScreenShare?: () => void
    onOpenSettings?: () => void
  } = $props()

  const canManageChannels = $derived(
    currentUserRole === 'owner' || currentUserRole === 'admin' || currentUserRole === 'moderator'
  )

  function getAvatarForSpeaker(speaker: SpeakerData): string | undefined {
    // Try to find member avatar by matching username
    const member = serverMembers.find(m => m.username === speaker.username)
    return member?.avatar_url || speaker.avatar_url
  }

  function qualityStyle(quality?: SpeakerData['quality']): string {
    if (quality === 'good') return 'bg-void-online/20 text-void-online'
    if (quality === 'fair') return 'bg-void-warning/20 text-void-warning'
    if (quality === 'poor') return 'bg-void-danger/20 text-void-danger'
    return 'bg-void-bg-hover text-void-text-muted'
  }

  function qualityLabel(quality?: SpeakerData['quality']): string {
    if (quality === 'good') return 'good'
    if (quality === 'fair') return 'fair'
    if (quality === 'poor') return 'poor'
    return 'unknown'
  }

  const displayName = $derived(currentUser?.display_name || currentUser?.username || 'You')
  const githubUsername = $derived(currentUser?.username || 'you')
  const initials = $derived(displayName.slice(0, 2).toUpperCase())

  const textChannels = $derived(channels.filter(c => c.type === 'text'))
  const voiceChannels = $derived(channels.filter(c => c.type === 'voice'))
  const trans = $derived($translations)

  let creatingText = $state(false)
  let creatingVoice = $state(false)
  let newChannelName = $state('')

  function submitChannel(type: 'text' | 'voice') {
    const name = newChannelName.trim()
    if (!name) return
    onCreateChannel?.(name, type)
    newChannelName = ''
    creatingText = false
    creatingVoice = false
  }
</script>

<aside class="flex h-full w-60 flex-col bg-void-bg-secondary">
  <!-- Server name header -->
  <button
    class="flex h-12 items-center justify-between border-b border-void-border px-4 transition-colors hover:bg-void-bg-hover cursor-pointer"
    onclick={() => onServerInfo?.()}
  >
    {#if loading}
      <Skeleton className="h-4 w-36 rounded-md" />
    {:else}
      <span class="text-sm font-bold text-void-text-primary truncate">{serverName}</span>
    {/if}
    <svg class="h-4 w-4 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polyline points="6 9 12 15 18 9" />
    </svg>
  </button>

  <!-- Channel list -->
  <div class="flex-1 overflow-y-auto px-2 pt-4">
    {#if loading}
      <div class="space-y-5 px-1">
        <div>
          <div class="mb-2 flex items-center gap-2">
            <Skeleton className="h-3 w-3 rounded-sm" />
            <Skeleton className="h-3 w-24 rounded-md" />
          </div>
          <div class="space-y-1">
            {#each Array.from({ length: 4 }) as _, i (`text-sk-${i}`)}
              <div class="flex items-center gap-2 px-2 py-1.5">
                <Skeleton className="h-4 w-4 rounded-sm" />
                <Skeleton className="h-3 w-32 rounded-md" />
              </div>
            {/each}
          </div>
        </div>

        <div>
          <div class="mb-2 flex items-center gap-2">
            <Skeleton className="h-3 w-3 rounded-sm" />
            <Skeleton className="h-3 w-24 rounded-md" />
          </div>
          <div class="space-y-1">
            {#each Array.from({ length: 3 }) as _, i (`voice-sk-${i}`)}
              <div class="flex items-center gap-2 px-2 py-1.5">
                <Skeleton className="h-4 w-4 rounded-sm" />
                <Skeleton className="h-3 w-28 rounded-md" />
              </div>
              <div class="ml-5 mb-1 space-y-1">
                <div class="flex items-center gap-2 px-2 py-1">
                  <Skeleton className="h-6 w-6 rounded-full" />
                  <Skeleton className="h-3 w-20 rounded-md" />
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {:else}
    <!-- Text channels -->
      <div class="mb-1 flex items-center gap-1 px-1">
        <svg class="h-3 w-3 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
        <span class="text-[11px] font-bold uppercase tracking-wide text-void-text-muted flex-1">{t(trans, 'channel.textChannels')}</span>
        <button
          class="rounded p-0.5 text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
          onclick={() => { creatingText = true; creatingVoice = false; newChannelName = '' }}
          aria-label={t(trans, 'channel.createText')}
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </button>
      </div>
    {#if creatingText}
      <div class="flex gap-1 px-1 mb-1 animate-fade-in-down">
        <input
          type="text"
          bind:value={newChannelName}
          placeholder={t(trans, 'channel.namePlaceholder')}
          class="flex-1 min-w-0 rounded-md bg-void-bg-primary px-2 py-1 text-xs text-void-text-primary placeholder:text-void-text-muted outline-none focus:ring-1 focus:ring-void-accent"
          onkeydown={(e) => { if (e.key === 'Enter') submitChannel('text'); if (e.key === 'Escape') creatingText = false }}
        />
        <button
          class="shrink-0 rounded-md bg-void-accent px-2 py-1 text-[10px] font-medium text-white hover:bg-void-accent-hover transition-colors cursor-pointer disabled:opacity-50"
          onclick={() => submitChannel('text')}
          disabled={!newChannelName.trim()}
        >{t(trans, 'channel.ok')}</button>
      </div>
    {/if}
    {#if textChannels.length > 0}
      {#each textChannels as channel}
        <div class="group flex items-center rounded-md transition-colors
          {channel.id === activeChannelId
            ? 'bg-void-bg-hover'
            : 'hover:bg-void-bg-hover'}">
          <button
            class="flex flex-1 items-center gap-1.5 px-2 py-1.5 text-sm cursor-pointer min-w-0
              {channel.id === activeChannelId
                ? 'text-void-text-primary'
                : 'text-void-text-secondary hover:text-void-text-primary'}"
            onclick={() => onSelectChannel(channel.id)}
          >
            <svg class="h-4 w-4 shrink-0 opacity-60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="4" y1="9" x2="20" y2="9" />
              <line x1="4" y1="15" x2="20" y2="15" />
              <line x1="10" y1="3" x2="8" y2="21" />
              <line x1="16" y1="3" x2="14" y2="21" />
            </svg>
            <span class="truncate">{channel.name}</span>
            {#if channel.unreadCount}
              <span class="ml-auto flex h-4 min-w-4 items-center justify-center rounded-full bg-void-danger px-1 text-[10px] font-bold text-white">
                {channel.unreadCount}
              </span>
            {/if}
          </button>
          {#if canManageChannels}
            <button
              class="shrink-0 p-1 mr-1 rounded text-void-text-muted opacity-0 group-hover:opacity-100 hover:text-void-danger transition-all cursor-pointer"
              onclick={() => onDeleteChannel?.(channel.id)}
              aria-label={t(trans, 'channel.deleteChannel')}
            >
              <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
            </button>
          {/if}
        </div>
      {/each}
    {/if}

    <!-- Voice channels -->
      <div class="mb-1 mt-4 flex items-center gap-1 px-1">
        <svg class="h-3 w-3 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
        <span class="text-[11px] font-bold uppercase tracking-wide text-void-text-muted flex-1">{t(trans, 'channel.voiceChannels')}</span>
        <button
          class="rounded p-0.5 text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
          onclick={() => { creatingVoice = true; creatingText = false; newChannelName = '' }}
          aria-label={t(trans, 'channel.createVoice')}
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </button>
      </div>
    {#if creatingVoice}
      <div class="flex gap-1 px-1 mb-1 animate-fade-in-down">
        <input
          type="text"
          bind:value={newChannelName}
          placeholder={t(trans, 'channel.namePlaceholder')}
          class="flex-1 min-w-0 rounded-md bg-void-bg-primary px-2 py-1 text-xs text-void-text-primary placeholder:text-void-text-muted outline-none focus:ring-1 focus:ring-void-accent"
          onkeydown={(e) => { if (e.key === 'Enter') submitChannel('voice'); if (e.key === 'Escape') creatingVoice = false }}
        />
        <button
          class="shrink-0 rounded-md bg-void-accent px-2 py-1 text-[10px] font-medium text-white hover:bg-void-accent-hover transition-colors cursor-pointer disabled:opacity-50"
          onclick={() => submitChannel('voice')}
          disabled={!newChannelName.trim()}
        >{t(trans, 'channel.ok')}</button>
      </div>
    {/if}
    {#if voiceChannels.length > 0}
      {#each voiceChannels as channel}
        <div class="group flex items-center rounded-md transition-colors
          {voiceChannelId === channel.id
            ? 'bg-void-online/10'
            : 'hover:bg-void-bg-hover'}">
          <button
            class="flex flex-1 items-center gap-1.5 px-2 py-1.5 text-sm cursor-pointer min-w-0
              {voiceChannelId === channel.id
                ? 'text-void-online'
                : 'text-void-text-secondary hover:text-void-text-primary'}"
            onclick={() => onJoinVoice?.(channel.id)}
          >
            <svg class="h-4 w-4 shrink-0 opacity-60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
              <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
              <line x1="12" y1="19" x2="12" y2="23" />
              <line x1="8" y1="23" x2="16" y2="23" />
            </svg>
            <span class="truncate">{channel.name}</span>
            {#if voiceChannelId === channel.id && voiceElapsed}
              <span class="ml-auto text-[11px] font-mono text-void-online tabular-nums">{voiceElapsed}</span>
            {:else}
              {@const spectatorParticipants = getChannelParticipants?.(channel.id) ?? []}
              {#if spectatorParticipants.length > 0}
                <span class="ml-auto flex items-center gap-1 text-[11px] text-void-text-muted">
                  <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                    <circle cx="9" cy="7" r="4" />
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                    <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                  </svg>
                  {spectatorParticipants.length}
                </span>
              {/if}
            {/if}
          </button>
          {#if canManageChannels}
            <button
              class="shrink-0 p-1 mr-1 rounded text-void-text-muted opacity-0 group-hover:opacity-100 hover:text-void-danger transition-all cursor-pointer"
              onclick={() => onDeleteChannel?.(channel.id)}
              aria-label={t(trans, 'channel.deleteChannel')}
            >
              <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
            </button>
          {/if}
        </div>
        <!-- Connected users in this voice channel -->
        {@const isMyChannel = voiceChannelId === channel.id}
        {@const channelSpeakers = isMyChannel ? voiceSpeakers : (getChannelParticipants?.(channel.id) ?? [])}
        {#if channelSpeakers.length > 0}
          <div class="ml-4 mb-1 space-y-0.5">
            {#each channelSpeakers as speaker (speaker.peer_id || speaker.user_id || speaker.username)}
              {@const avatarUrl = getAvatarForSpeaker(speaker)}
              {@const displaySpeakerName = speaker.username || speaker.user_id || speaker.peer_id || 'user'}
              {@const isLocal = isMyChannel && (
                (currentUser?.id && speaker.user_id === currentUser.id) ||
                speaker.username === currentUser?.username
              )}
              <div class="flex items-center gap-2 rounded-md py-1 px-2 hover:bg-void-bg-hover/50 transition-colors">
                <div class="relative shrink-0">
                  {#if avatarUrl}
                    <img src={avatarUrl} alt={displaySpeakerName} class="h-6 w-6 rounded-full object-cover" />
                  {:else}
                    <div class="h-6 w-6 rounded-full bg-void-accent/30 flex items-center justify-center text-[9px] font-bold text-void-accent">
                      {displaySpeakerName.slice(0, 2).toUpperCase()}
                    </div>
                  {/if}
                </div>
                <span class="text-xs text-void-text-secondary truncate">{displaySpeakerName}</span>
                {#if speaker.dominant}
                  <span class="rounded bg-void-accent/20 px-1.5 py-0.5 text-[9px] font-semibold uppercase text-void-accent">voice</span>
                {/if}
                {#if speaker.screenSharing || (isLocal && voiceScreenSharing)}
                  <span class="rounded bg-void-danger px-1.5 py-0.5 text-[9px] font-bold uppercase text-white animate-pulse">{t(trans, 'channel.live')}</span>
                {/if}
                <span class="rounded px-1.5 py-0.5 text-[9px] font-semibold uppercase {qualityStyle(speaker.quality)}">
                  {qualityLabel(speaker.quality)}
                </span>
                {#if speaker.deafened}
                  <Tooltip text={t(trans, 'voice.deafened')} position="top">
                    <svg class="h-3 w-3 shrink-0 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="1" y1="1" x2="23" y2="23" />
                      <path d="M3 18v-6a9 9 0 0 1 .84-3.8" />
                      <path d="M21 18v-6a9 9 0 0 0-9-9c-1.83 0-3.52.55-4.93 1.49" />
                      <path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z" />
                    </svg>
                  </Tooltip>
                {:else if speaker.muted}
                  <Tooltip text={t(trans, 'voice.muted')} position="top">
                    <svg class="h-3 w-3 shrink-0 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="1" y1="1" x2="23" y2="23" />
                      <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6" />
                      <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2c0 .67-.1 1.32-.27 1.93" />
                      <line x1="12" y1="19" x2="12" y2="23" />
                      <line x1="8" y1="23" x2="16" y2="23" />
                    </svg>
                  </Tooltip>
                {/if}
                <svg class="ml-auto h-3.5 w-3.5 shrink-0 {(speaker.speaking || (isLocal && voiceLocalSpeaking)) ? 'text-void-online animate-pulse' : 'text-void-text-muted'}" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M1 9l2 2c4.97-4.97 13.03-4.97 18 0l2-2C16.93 2.93 7.08 2.93 1 9zm8 8l3 3 3-3c-1.65-1.66-4.34-1.66-6 0zm-4-4l2 2c2.76-2.76 7.24-2.76 10 0l2-2C15.14 9.14 8.87 9.14 5 13z"/>
                </svg>
              </div>
            {/each}
          </div>
        {/if}
      {/each}
    {/if}
    {/if}
  </div>

  <!-- Voice controls panel (shows when connected to voice) -->
  <VoiceControls
    connected={voiceConnected}
    channelName={voiceChannelName}
    muted={voiceMuted}
    deafened={voiceDeafened}
    noiseSuppression={voiceNoiseSuppression}
    screenSharing={voiceScreenSharing}
    screenShares={voiceScreenShares}
    diagnostics={voiceDiagnostics}
    onToggleMute={() => onToggleMute?.()}
    onToggleDeafen={() => onToggleDeafen?.()}
    onToggleNoiseSuppression={() => onToggleNoiseSuppression?.()}
    onToggleScreenShare={() => onToggleScreenShare?.()}
    onDisconnect={() => onLeaveVoice?.()}
  />

  <!-- User panel -->
  <div class="border-t border-void-border bg-void-bg-primary p-2">
    <div class="flex items-center gap-2 rounded-md px-2 py-1.5">
      {#if loading}
        <Skeleton className="h-8 w-8 rounded-full" />
      {:else if currentUser?.avatar_url}
        <img src={currentUser.avatar_url} alt={displayName} class="h-8 w-8 shrink-0 rounded-full object-cover" />
      {:else}
        <div class="h-8 w-8 shrink-0 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
          {initials}
        </div>
      {/if}
      <div class="flex-1 min-w-0">
        {#if loading}
          <Skeleton className="h-3.5 w-24 rounded-md mb-1" />
          <Skeleton className="h-3 w-14 rounded-md" />
        {:else}
          <p class="text-sm font-medium text-void-text-primary truncate">{displayName}</p>
          <p class="text-[11px] text-void-online">{t(trans, 'channel.online')}</p>
        {/if}
      </div>
      <Tooltip text={t(trans, 'nav.settings')} position="top">
        <button aria-label={t(trans, 'nav.settings')} class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
          onclick={() => onOpenSettings?.()}>
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
        </button>
      </Tooltip>
    </div>
  </div>
</aside>
