<script lang="ts">
  import logoPng from '../../../assets/logo.png'
  import { translations, t } from '../../i18n'

  let {
    onAddServer,
  }: {
    onAddServer: () => void
  } = $props()

  const trans = $derived($translations)
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-6 bg-void-bg-tertiary select-none">
  <!-- Logo mark -->
  <div class="flex h-20 w-20 items-center justify-center rounded-3xl bg-void-bg-secondary shadow-lg">
    <img src={logoPng} alt="Concord" width="52" height="52" />
  </div>

  <div class="text-center">
    <h2 class="mb-1 text-xl font-bold text-void-text-primary">{t(trans, 'noServers.welcome')}</h2>
    <p class="text-sm text-void-text-muted">{t(trans, 'noServers.subtitle')}</p>
  </div>

  <div class="flex flex-col gap-3 w-56">
    <button
      onclick={onAddServer}
      class="flex items-center justify-center gap-2 rounded-lg bg-void-accent px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-void-accent-hover cursor-pointer"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {t(trans, 'noServers.create')}
    </button>
    <button
      onclick={onAddServer}
      class="flex items-center justify-center gap-2 rounded-lg border border-void-border bg-void-bg-secondary px-4 py-2.5 text-sm font-semibold text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
        <polyline points="10 17 15 12 10 7" />
        <line x1="15" y1="12" x2="3" y2="12" />
      </svg>
      {t(trans, 'noServers.join')}
    </button>
  </div>
</div>
