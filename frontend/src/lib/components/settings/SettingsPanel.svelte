<script lang="ts">
  import Avatar from '../ui/Avatar.svelte'
  import Button from '../ui/Button.svelte'
  import Toggle from '../ui/Toggle.svelte'
  import Dropdown from '../ui/Dropdown.svelte'
  import {
    getSettings, setAudioInput, setAudioOutput,
    setNotifications, setNotificationSounds,
    setTranslationLangs, setTheme,
  } from '../../stores/settings.svelte'

  interface Props {
    open: boolean
    currentUser: { username: string; display_name: string; avatar_url: string } | null
    onLogout: () => void
    onSwitchMode?: () => void
  }

  let { open = $bindable(false), currentUser, onLogout, onSwitchMode }: Props = $props()

  const settings = getSettings()

  type Category = 'account' | 'audio' | 'appearance' | 'notifications' | 'language'
  let activeCategory = $state<Category>('account')

  // Audio device lists
  let audioInputDevices = $state<{ label: string; value: string }[]>([])
  let audioOutputDevices = $state<{ label: string; value: string }[]>([])

  // Load audio devices when panel opens
  $effect(() => {
    if (open) {
      enumerateAudioDevices()
    }
  })

  async function enumerateAudioDevices() {
    try {
      // Request permission first so labels are available
      await navigator.mediaDevices.getUserMedia({ audio: true })
        .then(stream => stream.getTracks().forEach(t => t.stop()))
        .catch(() => { /* permission denied is fine, we still list devices */ })

      const devices = await navigator.mediaDevices.enumerateDevices()

      audioInputDevices = devices
        .filter(d => d.kind === 'audioinput')
        .map(d => ({
          label: d.label || `Microphone ${d.deviceId.slice(0, 8)}`,
          value: d.deviceId,
        }))

      audioOutputDevices = devices
        .filter(d => d.kind === 'audiooutput')
        .map(d => ({
          label: d.label || `Speaker ${d.deviceId.slice(0, 8)}`,
          value: d.deviceId,
        }))
    } catch (e) {
      console.error('Failed to enumerate audio devices:', e)
    }
  }

  // Bindable proxies for dropdowns
  let selectedInputDevice = $state(settings.audioInputDevice)
  let selectedOutputDevice = $state(settings.audioOutputDevice)
  let notifEnabled = $state(settings.notificationsEnabled)
  let notifSounds = $state(settings.notificationSounds)
  let srcLang = $state(settings.translationSourceLang)
  let tgtLang = $state(settings.translationTargetLang)

  // Sync from store when panel opens
  $effect(() => {
    if (open) {
      selectedInputDevice = settings.audioInputDevice
      selectedOutputDevice = settings.audioOutputDevice
      notifEnabled = settings.notificationsEnabled
      notifSounds = settings.notificationSounds
      srcLang = settings.translationSourceLang
      tgtLang = settings.translationTargetLang
    }
  })

  // Sync changes to store
  $effect(() => { setAudioInput(selectedInputDevice) })
  $effect(() => { setAudioOutput(selectedOutputDevice) })
  $effect(() => { setNotifications(notifEnabled) })
  $effect(() => { setNotificationSounds(notifSounds) })
  $effect(() => { setTranslationLangs(srcLang, tgtLang) })

  const categories: { id: Category; label: string; icon: string }[] = [
    { id: 'account', label: 'Account', icon: 'user' },
    { id: 'audio', label: 'Audio', icon: 'mic' },
    { id: 'appearance', label: 'Appearance', icon: 'palette' },
    { id: 'notifications', label: 'Notifications', icon: 'bell' },
    { id: 'language', label: 'Language', icon: 'globe' },
  ]

  const languageOptions = [
    { label: 'English', value: 'en' },
    { label: 'Portuguese', value: 'pt' },
    { label: 'Spanish', value: 'es' },
    { label: 'French', value: 'fr' },
    { label: 'German', value: 'de' },
    { label: 'Italian', value: 'it' },
    { label: 'Japanese', value: 'ja' },
    { label: 'Korean', value: 'ko' },
    { label: 'Chinese', value: 'zh' },
    { label: 'Russian', value: 'ru' },
    { label: 'Arabic', value: 'ar' },
  ]

  function close() {
    open = false
    activeCategory = 'account'
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      close()
    }
  }
</script>

{#if open}
  <!-- Full-screen overlay -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    role="dialog"
    tabindex="-1"
    aria-modal="true"
    aria-label="Settings"
    onkeydown={handleKeydown}
  >
    <div class="flex h-[80vh] w-[90vw] max-w-4xl overflow-hidden rounded-lg border border-void-border bg-void-bg-primary shadow-md animate-[settings-in_150ms_ease-out]">
      <!-- Category sidebar -->
      <nav class="flex w-56 shrink-0 flex-col border-r border-void-border bg-void-bg-secondary">
        <div class="px-4 pt-4 pb-2">
          <h2 class="text-xs font-bold uppercase tracking-wider text-void-text-muted">Settings</h2>
        </div>
        <div class="flex-1 overflow-y-auto px-2 py-1">
          {#each categories as cat}
            <button
              class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors cursor-pointer mb-0.5
                {activeCategory === cat.id
                  ? 'bg-void-bg-hover text-void-text-primary'
                  : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
              onclick={() => activeCategory = cat.id}
            >
              {#if cat.icon === 'user'}
                <svg class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                  <circle cx="12" cy="7" r="4" />
                </svg>
              {:else if cat.icon === 'mic'}
                <svg class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
                  <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                  <line x1="12" y1="19" x2="12" y2="23" />
                  <line x1="8" y1="23" x2="16" y2="23" />
                </svg>
              {:else if cat.icon === 'palette'}
                <svg class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="13.5" cy="6.5" r="0.5" fill="currentColor" />
                  <circle cx="17.5" cy="10.5" r="0.5" fill="currentColor" />
                  <circle cx="8.5" cy="7.5" r="0.5" fill="currentColor" />
                  <circle cx="6.5" cy="12.5" r="0.5" fill="currentColor" />
                  <path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.926 0 1.648-.746 1.648-1.688 0-.437-.18-.835-.437-1.125-.29-.289-.438-.652-.438-1.125a1.64 1.64 0 0 1 1.668-1.668h1.996c3.051 0 5.555-2.503 5.555-5.554C21.965 6.012 17.461 2 12 2z" />
                </svg>
              {:else if cat.icon === 'bell'}
                <svg class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
                  <path d="M13.73 21a2 2 0 0 1-3.46 0" />
                </svg>
              {:else if cat.icon === 'globe'}
                <svg class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="2" y1="12" x2="22" y2="12" />
                  <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                </svg>
              {/if}
              <span>{cat.label}</span>
            </button>
          {/each}
        </div>
        <!-- Separator + version info -->
        <div class="border-t border-void-border px-4 py-3">
          <p class="text-[11px] text-void-text-muted">Concord Desktop</p>
        </div>
      </nav>

      <!-- Content area -->
      <div class="flex flex-1 flex-col overflow-hidden">
        <!-- Header with close button -->
        <div class="flex items-center justify-between border-b border-void-border px-6 py-3">
          <h3 class="text-base font-bold text-void-text-primary">
            {categories.find(c => c.id === activeCategory)?.label ?? 'Settings'}
          </h3>
          <button
            aria-label="Close settings"
            class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
            onclick={close}
          >
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <!-- Scrollable content -->
        <div class="flex-1 overflow-y-auto px-6 py-5">
          <!-- Account -->
          {#if activeCategory === 'account'}
            <div class="space-y-6">
              {#if currentUser}
                <div class="flex items-center gap-4 rounded-lg border border-void-border bg-void-bg-secondary p-4">
                  <Avatar
                    src={currentUser.avatar_url}
                    name={currentUser.display_name || currentUser.username}
                    size="lg"
                  />
                  <div class="min-w-0 flex-1">
                    <p class="text-base font-bold text-void-text-primary truncate">
                      {currentUser.display_name || currentUser.username}
                    </p>
                    <p class="text-sm text-void-text-secondary truncate">
                      @{currentUser.username}
                    </p>
                  </div>
                </div>
              {:else}
                <div class="rounded-lg border border-void-border bg-void-bg-secondary p-4">
                  <p class="text-sm text-void-text-muted">No user logged in.</p>
                </div>
              {/if}

              <div class="border-t border-void-border pt-4">
                <h4 class="mb-3 text-sm font-semibold text-void-text-primary">Account Actions</h4>
                <Button
                  variant="danger"
                  size="sm"
                  onclick={() => { onLogout(); close() }}
                >
                  Log Out
                </Button>
              </div>

              {#if onSwitchMode}
                <div class="border-t border-void-border pt-4">
                  <h4 class="mb-3 text-sm font-semibold text-void-text-primary">Connection Mode</h4>
                  <div class="flex items-center gap-3 mb-3">
                    <span class="text-sm text-void-text-secondary">Modo atual:</span>
                    {#if settings.networkMode === 'p2p'}
                      <span class="rounded-full bg-void-online/20 px-2.5 py-0.5 text-xs font-bold text-void-online">P2P</span>
                    {:else}
                      <span class="rounded-full bg-blue-500/20 px-2.5 py-0.5 text-xs font-bold text-blue-400">Servidor</span>
                    {/if}
                  </div>
                  <p class="mb-3 text-xs text-void-text-muted">Voce sera redirecionado para a selecao de modo.</p>
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => { onSwitchMode(); close() }}
                  >
                    Trocar modo
                  </Button>
                </div>
              {/if}
            </div>
          {/if}

          <!-- Audio -->
          {#if activeCategory === 'audio'}
            <div class="space-y-6">
              <div>
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Input Device</h4>
                <p class="mb-3 text-xs text-void-text-muted">Select the microphone to use for voice chat.</p>
                {#if audioInputDevices.length > 0}
                  <Dropdown
                    items={audioInputDevices}
                    bind:selected={selectedInputDevice}
                    placeholder="Default microphone"
                  />
                {:else}
                  <p class="text-sm text-void-text-muted">No input devices found. Grant microphone permission to see devices.</p>
                {/if}
              </div>

              <div class="border-t border-void-border pt-4">
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Output Device</h4>
                <p class="mb-3 text-xs text-void-text-muted">Select the speaker or headphones to use for audio playback.</p>
                {#if audioOutputDevices.length > 0}
                  <Dropdown
                    items={audioOutputDevices}
                    bind:selected={selectedOutputDevice}
                    placeholder="Default speaker"
                  />
                {:else}
                  <p class="text-sm text-void-text-muted">No output devices found.</p>
                {/if}
              </div>
            </div>
          {/if}

          <!-- Appearance -->
          {#if activeCategory === 'appearance'}
            <div class="space-y-6">
              <div>
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Theme</h4>
                <p class="mb-3 text-xs text-void-text-muted">Choose the visual theme for Concord.</p>
                <div class="flex gap-3">
                  <!-- Dark theme -->
                  <button
                    class="flex flex-col items-center gap-2 rounded-lg border-2 bg-void-bg-secondary p-4 cursor-pointer transition-colors
                      {settings.theme === 'dark' ? 'border-void-accent' : 'border-void-border hover:border-void-text-muted'}"
                    onclick={() => setTheme('dark')}
                  >
                    <div class="flex h-10 w-16 items-center justify-center rounded-md bg-void-bg-primary">
                      <svg class="h-5 w-5 {settings.theme === 'dark' ? 'text-void-accent' : 'text-void-text-muted'}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
                      </svg>
                    </div>
                    <span class="text-xs font-medium {settings.theme === 'dark' ? 'text-void-accent' : 'text-void-text-muted'}">Dark</span>
                  </button>
                  <!-- Light theme -->
                  <button
                    class="flex flex-col items-center gap-2 rounded-lg border-2 bg-void-bg-secondary p-4 cursor-pointer transition-colors
                      {settings.theme === 'light' ? 'border-void-accent' : 'border-void-border hover:border-void-text-muted'}"
                    onclick={() => setTheme('light')}
                  >
                    <div class="flex h-10 w-16 items-center justify-center rounded-md bg-void-bg-tertiary">
                      <svg class="h-5 w-5 {settings.theme === 'light' ? 'text-void-accent' : 'text-void-text-muted'}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="5" />
                        <line x1="12" y1="1" x2="12" y2="3" />
                        <line x1="12" y1="21" x2="12" y2="23" />
                        <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                        <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                        <line x1="1" y1="12" x2="3" y2="12" />
                        <line x1="21" y1="12" x2="23" y2="12" />
                        <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                        <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
                      </svg>
                    </div>
                    <span class="text-xs font-medium {settings.theme === 'light' ? 'text-void-accent' : 'text-void-text-muted'}">Light</span>
                  </button>
                </div>
              </div>
            </div>
          {/if}

          <!-- Notifications -->
          {#if activeCategory === 'notifications'}
            <div class="space-y-6">
              <div>
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Desktop Notifications</h4>
                <p class="mb-3 text-xs text-void-text-muted">Show desktop notifications for new messages and events.</p>
                <Toggle bind:checked={notifEnabled} label="Enable notifications" />
              </div>

              <div class="border-t border-void-border pt-4">
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Notification Sounds</h4>
                <p class="mb-3 text-xs text-void-text-muted">Play a sound when a notification is received.</p>
                <Toggle bind:checked={notifSounds} label="Enable notification sounds" disabled={!notifEnabled} />
              </div>
            </div>
          {/if}

          <!-- Language -->
          {#if activeCategory === 'language'}
            <div class="space-y-6">
              <div>
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Translation</h4>
                <p class="mb-3 text-xs text-void-text-muted">
                  Configure os idiomas de tradução. Para traduzir uma mensagem, passe o mouse sobre ela
                  e clique no ícone de tradução (<svg class="inline h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.913 17H20.087M12.913 17L11 21M12.913 17L16.5 9L20.087 17M2 5H12M7 2V5M11 5C9.72 8.33 7.5 11.17 5 13.5M8 17C6.18 15.27 4.56 13.42 3.18 11.36" /></svg>).
                </p>
              </div>

              <div>
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Source Language</h4>
                <p class="mb-3 text-xs text-void-text-muted">The language you or others are speaking in.</p>
                <Dropdown
                  items={languageOptions}
                  bind:selected={srcLang}
                  placeholder="Select source language"
                />
              </div>

              <div class="border-t border-void-border pt-4">
                <h4 class="mb-1 text-sm font-semibold text-void-text-primary">Target Language</h4>
                <p class="mb-3 text-xs text-void-text-muted">The language to translate into.</p>
                <Dropdown
                  items={languageOptions}
                  bind:selected={tgtLang}
                  placeholder="Select target language"
                />
              </div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  @keyframes settings-in {
    from {
      opacity: 0;
      transform: scale(0.97);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
