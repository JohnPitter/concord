<script lang="ts">
  let {
    onCreateServer,
    onJoinServer,
  }: {
    onCreateServer: () => void
    onJoinServer: () => void
  } = $props()
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-6 bg-void-bg-tertiary select-none">
  <!-- Logo mark -->
  <div class="flex h-20 w-20 items-center justify-center rounded-3xl bg-void-bg-secondary shadow-lg">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="52" height="52">
      <defs>
        <linearGradient id="g-ns" x1="0" y1="0" x2="128" y2="128" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#4ade80"/>
          <stop offset="100%" stop-color="#16a34a"/>
        </linearGradient>
      </defs>
      <path fill="url(#g-ns)" d="M 58,6 C 89,6 114,22 120,48 C 121,52 118,56 114,56 C 110,56 107,53 106,50 C 102,30 82,18 58,18 C 28,18 10,38 10,59 C 10,80 28,100 58,100 C 82,100 102,88 106,68 C 107,65 110,62 114,62 C 118,62 121,66 120,70 C 114,96 89,112 58,112 L 42,112 L 16,126 L 22,108 C 8,100 -2,82 -2,59 C -2,28 22,6 58,6 Z"/>
      <path d="M 40,47 A 17,17 0 0 1 40,71" fill="none" stroke="#fff" stroke-width="6" stroke-linecap="round" opacity="0.95"/>
      <path d="M 54,39 A 26,26 0 0 1 54,79" fill="none" stroke="#fff" stroke-width="5.5" stroke-linecap="round" opacity="0.6"/>
      <path d="M 68,32 A 34,34 0 0 1 68,86" fill="none" stroke="#fff" stroke-width="5" stroke-linecap="round" opacity="0.3"/>
    </svg>
  </div>

  <div class="text-center">
    <h2 class="mb-1 text-xl font-bold text-void-text-primary">Bem-vindo ao Concord</h2>
    <p class="text-sm text-void-text-muted">Crie um servidor ou entre num existente para começar.</p>
  </div>

  <div class="flex flex-col gap-3 w-56">
    <button
      onclick={onCreateServer}
      class="flex items-center justify-center gap-2 rounded-lg bg-void-accent px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-void-accent-hover cursor-pointer"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      Criar um servidor
    </button>
    <button
      onclick={onJoinServer}
      class="flex items-center justify-center gap-2 rounded-lg border border-void-border bg-void-bg-secondary px-4 py-2.5 text-sm font-semibold text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
        <polyline points="10 17 15 12 10 7" />
        <line x1="15" y1="12" x2="3" y2="12" />
      </svg>
      Entrar com convite
    </button>
  </div>
</div>
