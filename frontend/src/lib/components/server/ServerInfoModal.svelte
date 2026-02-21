<script lang="ts">
  import Button from '../ui/Button.svelte'

  interface Props {
    open: boolean
    serverName: string
    memberCount: number
    inviteCode: string
    isOwner: boolean
    onClose: () => void
    onGenerateInvite: () => void
    onDeleteServer: () => void
  }

  let {
    open = $bindable(false),
    serverName,
    memberCount,
    inviteCode,
    isOwner,
    onClose,
    onGenerateInvite,
    onDeleteServer,
  }: Props = $props()

  let copied = $state(false)

  function copyInvite() {
    if (!inviteCode) return
    navigator.clipboard.writeText(inviteCode)
    copied = true
    setTimeout(() => copied = false, 2000)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose()
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    role="dialog"
    tabindex="-1"
    aria-modal="true"
    aria-label="Server Info"
    onkeydown={handleKeydown}
    onclick={(e) => { if (e.target === e.currentTarget) onClose() }}
  >
    <div class="w-[400px] rounded-lg border border-void-border bg-void-bg-primary shadow-md animate-[settings-in_150ms_ease-out]">
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-void-border px-5 py-4">
        <h3 class="text-base font-bold text-void-text-primary">{serverName}</h3>
        <button
          aria-label="Fechar"
          class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:bg-void-bg-hover hover:text-void-text-primary cursor-pointer"
          onclick={onClose}
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>

      <!-- Content -->
      <div class="px-5 py-4 space-y-4">
        <!-- Members -->
        <div class="flex items-center gap-3 rounded-lg bg-void-bg-secondary p-3">
          <svg class="h-5 w-5 text-void-text-muted shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
          <div>
            <p class="text-sm font-semibold text-void-text-primary">{memberCount} membro{memberCount !== 1 ? 's' : ''}</p>
          </div>
        </div>

        <!-- Invite -->
        <div>
          <h4 class="mb-2 text-sm font-semibold text-void-text-primary">Convite</h4>
          {#if inviteCode}
            <div class="flex items-center gap-2 rounded-lg bg-void-bg-secondary p-3">
              <p class="flex-1 font-mono text-sm text-void-accent select-all truncate">{inviteCode}</p>
              <button
                class="shrink-0 rounded-md px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer
                  {copied ? 'bg-void-online/20 text-void-online' : 'bg-void-bg-hover text-void-text-secondary hover:text-void-text-primary'}"
                onclick={copyInvite}
              >
                {copied ? 'Copiado!' : 'Copiar'}
              </button>
            </div>
          {:else}
            <Button variant="outline" size="sm" onclick={onGenerateInvite}>
              Gerar convite
            </Button>
          {/if}
        </div>

        <!-- Owner actions -->
        {#if isOwner}
          <div class="border-t border-void-border pt-4">
            <h4 class="mb-3 text-sm font-semibold text-void-text-primary">Zona de Perigo</h4>
            <Button
              variant="danger"
              size="sm"
              onclick={() => { onDeleteServer(); onClose() }}
            >
              Excluir Servidor
            </Button>
          </div>
        {/if}
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
