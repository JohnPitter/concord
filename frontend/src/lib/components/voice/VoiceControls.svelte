<script lang="ts">
  import Tooltip from '../ui/Tooltip.svelte'
  import { translations, t } from '../../i18n'

  let {
    connected,
    channelName,
    muted,
    deafened,
    noiseSuppression = true,
    screenSharing = false,
    onToggleMute,
    onToggleDeafen,
    onToggleNoiseSuppression,
    onToggleScreenShare,
    onDisconnect,
  }: {
    connected: boolean
    channelName: string
    muted: boolean
    deafened: boolean
    noiseSuppression?: boolean
    screenSharing?: boolean
    onToggleMute: () => void
    onToggleDeafen: () => void
    onToggleNoiseSuppression?: () => void
    onToggleScreenShare?: () => void
    onDisconnect: () => void
  } = $props()

  const trans = $derived($translations)
</script>

{#if connected}
  <div class="border-t border-void-border bg-void-bg-primary">
    <!-- Connected indicator -->
    <div class="flex items-center justify-between px-3 py-2">
      <div class="flex items-center gap-2 min-w-0">
        <div class="h-2 w-2 shrink-0 rounded-full bg-void-online animate-pulse"></div>
        <div class="min-w-0">
          <p class="text-xs font-semibold text-void-online">{t(trans, 'voice.connected')}</p>
          <p class="text-[11px] text-void-text-muted truncate">{channelName}</p>
        </div>
      </div>
      <Tooltip text={t(trans, 'voice.disconnect')} position="top">
        <button
          aria-label={t(trans, 'voice.disconnect')}
          class="rounded-md p-1.5 text-void-danger transition-colors hover:bg-void-danger/10 cursor-pointer"
          onclick={onDisconnect}
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 2.59 3.4z" />
            <line x1="1" y1="1" x2="23" y2="23" />
          </svg>
        </button>
      </Tooltip>
    </div>

    <!-- Controls -->
    <div class="flex items-center justify-center gap-1 border-t border-void-border px-3 py-1.5">
      <Tooltip text={noiseSuppression ? t(trans, 'voice.disableNoiseSuppression') : t(trans, 'voice.enableNoiseSuppression')} position="top">
        <button
          aria-label={noiseSuppression ? t(trans, 'voice.disableNoiseSuppression') : t(trans, 'voice.enableNoiseSuppression')}
          class="rounded-md p-2 transition-colors cursor-pointer
            {noiseSuppression
              ? 'text-void-online hover:bg-void-online/10'
              : 'text-void-text-muted hover:bg-void-bg-hover hover:text-void-text-primary'}"
          onclick={() => onToggleNoiseSuppression?.()}
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M2 16s1-1 3-1 3 1 5 1 3-1 5-1 3 1 5 1 3-1 3-1"/>
            <path d="M2 12s1-1 3-1 3 1 5 1 3-1 5-1 3 1 5 1 3-1 3-1"/>
            <path d="M2 8s1-1 3-1 3 1 5 1 3-1 5-1 3 1 5 1 3-1 3-1"/>
          </svg>
        </button>
      </Tooltip>
      <Tooltip text={screenSharing ? t(trans, 'voice.stopScreenShare') : t(trans, 'voice.shareScreen')} position="top">
        <button
          aria-label={screenSharing ? t(trans, 'voice.stopScreenShare') : t(trans, 'voice.shareScreen')}
          class="rounded-md p-2 transition-colors cursor-pointer
            {screenSharing
              ? 'bg-void-online/20 text-void-online hover:bg-void-online/30'
              : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
          onclick={() => onToggleScreenShare?.()}
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
            <line x1="8" y1="21" x2="16" y2="21"/>
            <line x1="12" y1="17" x2="12" y2="21"/>
          </svg>
        </button>
      </Tooltip>
      <div class="w-px h-5 bg-void-border mx-0.5"></div>
      <Tooltip text={muted ? t(trans, 'voice.unmute') : t(trans, 'voice.mute')} position="top">
        <button
          aria-label={muted ? t(trans, 'voice.unmute') : t(trans, 'voice.mute')}
          class="rounded-md p-2 transition-colors cursor-pointer
            {muted
              ? 'bg-void-danger/20 text-void-danger hover:bg-void-danger/30'
              : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
          onclick={onToggleMute}
        >
          {#if muted}
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="1" y1="1" x2="23" y2="23" />
              <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6" />
              <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2c0 .67-.1 1.32-.27 1.93" />
              <line x1="12" y1="19" x2="12" y2="23" />
              <line x1="8" y1="23" x2="16" y2="23" />
            </svg>
          {:else}
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
              <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
              <line x1="12" y1="19" x2="12" y2="23" />
              <line x1="8" y1="23" x2="16" y2="23" />
            </svg>
          {/if}
        </button>
      </Tooltip>
      <Tooltip text={deafened ? t(trans, 'voice.undeafen') : t(trans, 'voice.deafen')} position="top">
        <button
          aria-label={deafened ? t(trans, 'voice.undeafen') : t(trans, 'voice.deafen')}
          class="rounded-md p-2 transition-colors cursor-pointer
            {deafened
              ? 'bg-void-danger/20 text-void-danger hover:bg-void-danger/30'
              : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
          onclick={onToggleDeafen}
        >
          {#if deafened}
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="1" y1="1" x2="23" y2="23" />
              <path d="M3 18v-6a9 9 0 0 1 .84-3.8" />
              <path d="M21 18v-6a9 9 0 0 0-9-9c-1.83 0-3.52.55-4.93 1.49" />
              <path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z" />
            </svg>
          {:else}
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 18v-6a9 9 0 0 1 18 0v6" />
              <path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z" />
            </svg>
          {/if}
        </button>
      </Tooltip>
    </div>
  </div>
{/if}
