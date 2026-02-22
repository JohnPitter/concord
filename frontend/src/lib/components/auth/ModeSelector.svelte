<script lang="ts">
  import { translations, t } from '../../i18n'
  import logoPng from '../../../assets/logo.png'

  let {
    onSelectMode,
  }: {
    onSelectMode: (mode: 'p2p' | 'server') => void
  } = $props()

  const trans = $derived($translations)
</script>

<div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
  <div class="w-full max-w-2xl space-y-8 px-6">
    <!-- Logo -->
    <div class="text-center">
      <div class="mx-auto mb-4 h-16 w-16">
        <img src={logoPng} alt="Concord" class="h-16 w-16" />
      </div>
      <h1 class="mt-4 text-2xl font-bold text-void-text-primary">{t(trans, 'mode.welcome')}</h1>
      <p class="mt-2 text-sm text-void-text-muted">{t(trans, 'mode.howConnect')}</p>
    </div>

    <!-- Cards de escolha -->
    <div class="grid grid-cols-2 gap-4">
      <!-- P2P Card -->
      <button
        onclick={() => onSelectMode('p2p')}
        class="group flex flex-col items-center gap-4 rounded-xl border border-void-border bg-void-bg-secondary p-8 text-left transition-all hover:border-void-accent hover:bg-void-bg-hover cursor-pointer"
      >
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-void-bg-tertiary group-hover:bg-void-accent/10 transition-colors">
          <svg class="h-8 w-8 text-void-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M4 12v-1a8 8 0 0 1 8-8"/>
            <path d="M20 12v1a8 8 0 0 1-8 8"/>
            <polyline points="9 3 4 3 4 8"/>
            <polyline points="15 21 20 21 20 16"/>
          </svg>
        </div>
        <div class="text-center">
          <p class="text-base font-bold text-void-text-primary">{t(trans, 'mode.p2p')}</p>
          <p class="mt-1 text-xs text-void-text-muted leading-relaxed whitespace-pre-line">
            {t(trans, 'mode.p2pDesc')}
          </p>
        </div>
        <div class="mt-auto flex flex-col gap-1 w-full">
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            {t(trans, 'mode.p2pNoAccount')}
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            {t(trans, 'mode.p2pLanDiscovery')}
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            {t(trans, 'mode.p2pWanRoom')}
          </span>
        </div>
      </button>

      <!-- Servidor Card -->
      <button
        onclick={() => onSelectMode('server')}
        class="group flex flex-col items-center gap-4 rounded-xl border border-void-border bg-void-bg-secondary p-8 text-left transition-all hover:border-void-accent hover:bg-void-bg-hover cursor-pointer"
      >
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-void-bg-tertiary group-hover:bg-void-accent/10 transition-colors">
          <svg class="h-8 w-8 text-void-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
          </svg>
        </div>
        <div class="text-center">
          <p class="text-base font-bold text-void-text-primary">{t(trans, 'mode.server')}</p>
          <p class="mt-1 text-xs text-void-text-muted leading-relaxed whitespace-pre-line">
            {t(trans, 'mode.serverDesc')}
          </p>
        </div>
        <div class="mt-auto flex flex-col gap-1 w-full">
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            {t(trans, 'mode.serverGithub')}
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            {t(trans, 'mode.serverCloud')}
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            {t(trans, 'mode.serverGlobal')}
          </span>
        </div>
      </button>
    </div>

    <p class="text-center text-xs text-void-text-muted">
      {t(trans, 'mode.canChange')}
    </p>
  </div>
</div>
