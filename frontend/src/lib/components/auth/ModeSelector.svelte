<script lang="ts">
  let {
    onSelectMode,
  }: {
    onSelectMode: (mode: 'p2p' | 'server') => void
  } = $props()
</script>

<div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
  <div class="w-full max-w-2xl space-y-8 px-6">
    <!-- Logo -->
    <div class="text-center">
      <div class="mx-auto mb-4 h-16 w-16">
        <svg viewBox="0 0 128 128" class="h-16 w-16">
          <defs>
            <linearGradient id="cg" x1="0" y1="0" x2="128" y2="128" gradientUnits="userSpaceOnUse">
              <stop offset="0%" stop-color="#4ade80"/>
              <stop offset="100%" stop-color="#16a34a"/>
            </linearGradient>
          </defs>
          <path fill="url(#cg)" d="M58,6 C89,6 114,22 120,48 C121,52 118,56 114,56 C110,56 107,53 106,50 C102,30 82,18 58,18 C28,18 10,38 10,59 C10,80 28,100 58,100 C82,100 102,88 106,68 C107,65 110,62 114,62 C118,62 121,66 120,70 C114,96 89,112 58,112 L42,112 L16,126 L22,108 C8,100 -2,82 -2,59 C-2,28 22,6 58,6 Z"/>
          <path d="M40,47 A17,17 0 0 1 40,71" fill="none" stroke="#fff" stroke-width="6" stroke-linecap="round" opacity="0.95"/>
          <path d="M54,39 A26,26 0 0 1 54,79" fill="none" stroke="#fff" stroke-width="5.5" stroke-linecap="round" opacity="0.6"/>
          <path d="M68,32 A34,34 0 0 1 68,86" fill="none" stroke="#fff" stroke-width="5" stroke-linecap="round" opacity="0.3"/>
        </svg>
      </div>
      <h1 class="mt-4 text-2xl font-bold text-void-text-primary">Bem-vindo ao Concord</h1>
      <p class="mt-2 text-sm text-void-text-muted">Como você quer se conectar?</p>
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
          <p class="text-base font-bold text-void-text-primary">P2P</p>
          <p class="mt-1 text-xs text-void-text-muted leading-relaxed">
            Sem servidor central.<br>
            Conecte-se direto com amigos<br>
            na LAN ou via código de sala.
          </p>
        </div>
        <div class="mt-auto flex flex-col gap-1 w-full">
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Sem conta necessária
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Descoberta automática na LAN
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            WAN via código de sala
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
          <p class="text-base font-bold text-void-text-primary">Servidor Oficial</p>
          <p class="mt-1 text-xs text-void-text-muted leading-relaxed">
            Infraestrutura centralizada.<br>
            Mensagens persistidas,<br>
            acesso de qualquer lugar.
          </p>
        </div>
        <div class="mt-auto flex flex-col gap-1 w-full">
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Login com GitHub
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Mensagens na nuvem
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Servidores globais
          </span>
        </div>
      </button>
    </div>

    <p class="text-center text-xs text-void-text-muted">
      Esta escolha pode ser alterada nas configurações.
    </p>
  </div>
</div>
