<script lang="ts">
  import Tooltip from '../ui/Tooltip.svelte'
  import VoiceControls from '../voice/VoiceControls.svelte'
  import type { SpeakerData } from '../../stores/voice.svelte'

  interface Channel {
    id: string
    name: string
    type: 'text' | 'voice'
    unreadCount?: number
  }

  let {
    serverName,
    channels,
    activeChannelId,
    onSelectChannel,
    voiceConnected = false,
    voiceChannelName = '',
    voiceMuted = false,
    voiceDeafened = false,
    voiceSpeakers = [],
    voiceChannelId = null,
    onJoinVoice,
    onLeaveVoice,
    onToggleMute,
    onToggleDeafen,
    onOpenSettings,
  }: {
    serverName: string
    channels: Channel[]
    activeChannelId: string
    onSelectChannel: (id: string) => void
    voiceConnected?: boolean
    voiceChannelName?: string
    voiceMuted?: boolean
    voiceDeafened?: boolean
    voiceSpeakers?: SpeakerData[]
    voiceChannelId?: string | null
    onJoinVoice?: (channelId: string) => void
    onLeaveVoice?: () => void
    onToggleMute?: () => void
    onToggleDeafen?: () => void
    onOpenSettings?: () => void
  } = $props()

  const textChannels = $derived(channels.filter(c => c.type === 'text'))
  const voiceChannels = $derived(channels.filter(c => c.type === 'voice'))
</script>

<aside class="flex h-full w-60 flex-col bg-void-bg-secondary">
  <!-- Server name header -->
  <button class="flex h-12 items-center justify-between border-b border-void-border px-4 transition-colors hover:bg-void-bg-hover cursor-pointer">
    <span class="text-sm font-bold text-void-text-primary truncate">{serverName}</span>
    <svg class="h-4 w-4 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polyline points="6 9 12 15 18 9" />
    </svg>
  </button>

  <!-- Channel list -->
  <div class="flex-1 overflow-y-auto px-2 pt-4">
    <!-- Text channels -->
    {#if textChannels.length > 0}
      <div class="mb-1 flex items-center gap-1 px-1">
        <svg class="h-3 w-3 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
        <span class="text-[11px] font-bold uppercase tracking-wide text-void-text-muted">Text Channels</span>
      </div>
      {#each textChannels as channel}
        <button
          class="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm transition-colors cursor-pointer
            {channel.id === activeChannelId
              ? 'bg-void-bg-hover text-void-text-primary'
              : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
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
      {/each}
    {/if}

    <!-- Voice channels -->
    {#if voiceChannels.length > 0}
      <div class="mb-1 mt-4 flex items-center gap-1 px-1">
        <svg class="h-3 w-3 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
        <span class="text-[11px] font-bold uppercase tracking-wide text-void-text-muted">Voice Channels</span>
      </div>
      {#each voiceChannels as channel}
        <button
          class="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm transition-colors cursor-pointer
            {voiceChannelId === channel.id
              ? 'bg-void-online/10 text-void-online'
              : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
          onclick={() => onJoinVoice?.(channel.id)}
        >
          <svg class="h-4 w-4 shrink-0 opacity-60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
            <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
            <line x1="12" y1="19" x2="12" y2="23" />
            <line x1="8" y1="23" x2="16" y2="23" />
          </svg>
          <span class="truncate">{channel.name}</span>
          {#if voiceChannelId === channel.id}
            <div class="ml-auto h-2 w-2 rounded-full bg-void-online animate-pulse"></div>
          {/if}
        </button>
      {/each}
    {/if}
  </div>

  <!-- Voice controls panel (shows when connected to voice) -->
  <VoiceControls
    connected={voiceConnected}
    channelName={voiceChannelName}
    muted={voiceMuted}
    deafened={voiceDeafened}
    speakers={voiceSpeakers}
    onToggleMute={() => onToggleMute?.()}
    onToggleDeafen={() => onToggleDeafen?.()}
    onDisconnect={() => onLeaveVoice?.()}
  />

  <!-- User panel -->
  <div class="border-t border-void-border bg-void-bg-primary p-2">
    <div class="flex items-center gap-2 rounded-md px-2 py-1.5">
      <div class="h-8 w-8 shrink-0 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
        YO
      </div>
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-void-text-primary truncate">You</p>
        <p class="text-[11px] text-void-online">Online</p>
      </div>
      <Tooltip text="Settings" position="top">
        <button aria-label="Settings" class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
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
