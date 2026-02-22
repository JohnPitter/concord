<script lang="ts">
  import { translations, t } from '../../i18n'
  import logoImg from '../../../assets/logo.png'

  let { onClose }: { onClose: () => void } = $props()

  let step = $state(0)
  const totalSteps = 4
  const trans = $derived($translations)

  function next() {
    if (step < totalSteps - 1) step++
    else onClose()
  }

  function skip() {
    onClose()
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in"
  onclick={(e) => { if (e.target === e.currentTarget) skip() }}
>
  <div class="w-full max-w-md rounded-2xl bg-void-bg-secondary border border-void-border shadow-2xl overflow-hidden">
    <!-- Content area -->
    <div class="px-8 pt-8 pb-4 min-h-[320px] flex flex-col items-center justify-center text-center">
      {#if step === 0}
        <img src={logoImg} alt="Concord" class="h-20 w-20 mb-4" />
        <h2 class="text-2xl font-bold text-void-text-primary mb-2">{t(trans, 'welcome.title')}</h2>
        <p class="text-sm text-void-text-secondary leading-relaxed">{t(trans, 'welcome.subtitle')}</p>
      {:else if step === 1}
        <div class="mx-auto mb-4 h-16 w-16 rounded-2xl bg-void-accent/20 flex items-center justify-center">
          <svg class="h-8 w-8 text-void-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="3" width="20" height="14" rx="2"/>
            <line x1="8" y1="21" x2="16" y2="21"/>
            <line x1="12" y1="17" x2="12" y2="21"/>
          </svg>
        </div>
        <h2 class="text-xl font-bold text-void-text-primary mb-2">{t(trans, 'welcome.serversTitle')}</h2>
        <p class="text-sm text-void-text-secondary leading-relaxed">{t(trans, 'welcome.serversDesc')}</p>
      {:else if step === 2}
        <div class="mx-auto mb-4 h-16 w-16 rounded-2xl bg-void-online/20 flex items-center justify-center">
          <svg class="h-8 w-8 text-void-online" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
        </div>
        <h2 class="text-xl font-bold text-void-text-primary mb-2">{t(trans, 'welcome.friendsTitle')}</h2>
        <p class="text-sm text-void-text-secondary leading-relaxed">{t(trans, 'welcome.friendsDesc')}</p>
      {:else}
        <div class="mx-auto mb-4 h-16 w-16 rounded-2xl bg-void-accent/20 flex items-center justify-center">
          <svg class="h-8 w-8 text-void-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <h2 class="text-xl font-bold text-void-text-primary mb-2">{t(trans, 'welcome.readyTitle')}</h2>
        <p class="text-sm text-void-text-secondary leading-relaxed">{t(trans, 'welcome.readyDesc')}</p>
      {/if}
    </div>

    <!-- Step dots + buttons -->
    <div class="flex items-center justify-between px-8 pb-6">
      <button
        class="text-sm text-void-text-muted hover:text-void-text-primary transition-colors cursor-pointer"
        onclick={skip}
      >
        {t(trans, 'welcome.skip')}
      </button>

      <div class="flex items-center gap-1.5">
        {#each Array(totalSteps) as _, i}
          <div class="h-1.5 rounded-full transition-all duration-300 {i === step ? 'w-6 bg-void-accent' : 'w-1.5 bg-void-border'}"></div>
        {/each}
      </div>

      <button
        class="rounded-lg bg-void-accent px-4 py-2 text-sm font-semibold text-white hover:bg-void-accent-hover transition-colors cursor-pointer"
        onclick={next}
      >
        {step < totalSteps - 1 ? t(trans, 'welcome.next') : t(trans, 'welcome.finish')}
      </button>
    </div>
  </div>
</div>
