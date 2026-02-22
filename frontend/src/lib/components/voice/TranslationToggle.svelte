<script lang="ts">
  import Toggle from '../ui/Toggle.svelte'
  import Dropdown from '../ui/Dropdown.svelte'
  import Tooltip from '../ui/Tooltip.svelte'
  import { translations, t } from '../../i18n'
  import { getSettings, setTranslationLangs } from '../../stores/settings.svelte'
  import * as App from '../../../wailsjs/go/main/App'

  let { voiceConnected }: { voiceConnected: boolean } = $props()

  const settings = getSettings()
  const trans = $derived($translations)

  let translationEnabled = $state(false)
  let status = $state<'idle' | 'active' | 'loading' | 'error'>('idle')
  let errorMessage = $state<string | null>(null)

  let srcLang = $state(settings.translationSourceLang)
  let tgtLang = $state(settings.translationTargetLang)

  // Sync from settings store when settings change
  $effect(() => {
    srcLang = settings.translationSourceLang
    tgtLang = settings.translationTargetLang
  })

  // Persist language changes back to settings
  $effect(() => {
    setTranslationLangs(srcLang, tgtLang)
  })

  // Disable translation when voice disconnects
  $effect(() => {
    if (!voiceConnected && translationEnabled) {
      disableTranslation()
    }
  })

  const languageOptions = [
    { label: 'EN', value: 'en' },
    { label: 'PT', value: 'pt' },
    { label: 'ES', value: 'es' },
    { label: 'FR', value: 'fr' },
    { label: 'DE', value: 'de' },
    { label: 'IT', value: 'it' },
    { label: 'JA', value: 'ja' },
    { label: 'KO', value: 'ko' },
    { label: 'ZH', value: 'zh' },
    { label: 'RU', value: 'ru' },
  ]

  async function enableTranslation() {
    status = 'loading'
    errorMessage = null
    try {
      await App.EnableTranslation(srcLang, tgtLang)
      translationEnabled = true
      status = 'active'
    } catch (e) {
      status = 'error'
      errorMessage = e instanceof Error ? e.message : t(trans, 'translation.error')
      translationEnabled = false
    }
  }

  async function disableTranslation() {
    try {
      await App.DisableTranslation()
    } catch (e) {
      console.error('Failed to disable translation:', e)
    } finally {
      translationEnabled = false
      status = 'idle'
      errorMessage = null
    }
  }

  // Watch for toggle changes and enable/disable accordingly
  let prevEnabled = $state(false)
  $effect(() => {
    if (translationEnabled !== prevEnabled) {
      const wasEnabled = prevEnabled
      prevEnabled = translationEnabled
      if (translationEnabled && !wasEnabled) {
        enableTranslation()
      } else if (!translationEnabled && wasEnabled) {
        disableTranslation()
      }
    }
  })
</script>

{#if voiceConnected}
  <div class="border-t border-void-border bg-void-bg-primary px-3 py-2">
    <!-- Toggle row -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <!-- Status indicator -->
        {#if status === 'active'}
          <Tooltip text={t(trans, 'translation.active')} position="top">
            <div class="h-2 w-2 shrink-0 rounded-full bg-void-online"></div>
          </Tooltip>
        {:else if status === 'loading'}
          <div class="h-2 w-2 shrink-0">
            <svg class="h-2 w-2 animate-spin text-void-accent" viewBox="0 0 24 24" fill="none">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
        {:else if status === 'error'}
          <Tooltip text={errorMessage ?? t(trans, 'translation.error')} position="top">
            <div class="h-2 w-2 shrink-0 rounded-full bg-void-danger"></div>
          </Tooltip>
        {:else}
          <div class="h-2 w-2 shrink-0 rounded-full bg-void-text-muted"></div>
        {/if}

        <svg class="h-3.5 w-3.5 text-void-text-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="2" y1="12" x2="22" y2="12" />
          <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
        </svg>
        <span class="text-[11px] text-void-text-secondary">{t(trans, 'translation.translate')}</span>
      </div>
      <Toggle
        bind:checked={translationEnabled}
        label=""
        disabled={status === 'loading'}
      />
    </div>

    <!-- Language selectors (compact, only when enabled or idle) -->
    {#if translationEnabled || status === 'idle'}
      <div class="mt-1.5 flex items-center gap-1.5">
        <div class="flex-1">
          <Dropdown
            items={languageOptions}
            bind:selected={srcLang}
            placeholder={t(trans, 'translation.from')}
          />
        </div>
        <svg class="h-3 w-3 shrink-0 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="5" y1="12" x2="19" y2="12" />
          <polyline points="12 5 19 12 12 19" />
        </svg>
        <div class="flex-1">
          <Dropdown
            items={languageOptions}
            bind:selected={tgtLang}
            placeholder={t(trans, 'translation.to')}
          />
        </div>
      </div>
    {/if}
  </div>
{/if}
