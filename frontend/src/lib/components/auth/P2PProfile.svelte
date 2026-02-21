<script lang="ts">
  import { SelectAvatarFile } from '../../../wailsjs/go/main/App'
  import Button from '../ui/Button.svelte'

  let {
    onConfirm,
  }: {
    onConfirm: (profile: { displayName: string; avatarDataUrl?: string }) => void
  } = $props()

  let displayName = $state('')
  let avatarDataUrl = $state<string | undefined>(undefined)
  let error = $state('')
  let loadingAvatar = $state(false)

  async function handleSelectAvatar() {
    loadingAvatar = true
    try {
      const result = await SelectAvatarFile()
      if (result) avatarDataUrl = result
    } catch {
      // usuário cancelou o diálogo — não é erro
    } finally {
      loadingAvatar = false
    }
  }

  function handleConfirm() {
    const name = displayName.trim()
    if (name.length < 2) { error = 'Nome deve ter pelo menos 2 caracteres'; return }
    if (name.length > 32) { error = 'Nome deve ter no máximo 32 caracteres'; return }
    error = ''
    onConfirm({ displayName: name, avatarDataUrl })
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleConfirm()
  }
</script>

<div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
  <div class="w-full max-w-sm space-y-6 px-6">
    <div class="text-center">
      <h1 class="text-xl font-bold text-void-text-primary">Criar seu perfil P2P</h1>
      <p class="mt-1 text-sm text-void-text-muted">Sua identidade é armazenada apenas localmente.</p>
    </div>

    <!-- Avatar picker -->
    <div class="flex flex-col items-center gap-3">
      <button
        onclick={handleSelectAvatar}
        disabled={loadingAvatar}
        class="relative h-20 w-20 rounded-full overflow-hidden border-2 border-dashed border-void-border hover:border-void-accent transition-colors cursor-pointer group"
      >
        {#if avatarDataUrl}
          <img src={avatarDataUrl} alt="Avatar" class="h-full w-full object-cover" />
          <div class="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity">
            <svg class="h-6 w-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
          </div>
        {:else}
          <div class="flex h-full w-full flex-col items-center justify-center gap-1 bg-void-bg-secondary">
            {#if loadingAvatar}
              <div class="h-5 w-5 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
            {:else}
              <svg class="h-7 w-7 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
              <span class="text-[10px] text-void-text-muted">Foto</span>
            {/if}
          </div>
        {/if}
      </button>
      <p class="text-xs text-void-text-muted">Clique para escolher uma foto</p>
    </div>

    <!-- Name input -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div onkeydown={handleKeydown}>
      <label class="mb-1.5 block text-sm font-medium text-void-text-secondary">
        Nome de exibição
      </label>
      <input
        type="text"
        bind:value={displayName}
        placeholder="Seu nome..."
        maxlength="32"
        class="w-full rounded-md border border-void-border bg-void-bg-secondary px-3 py-2 text-sm text-void-text-primary placeholder:text-void-text-muted focus:border-void-accent focus:outline-none focus:ring-2 focus:ring-void-accent {error ? 'border-void-danger focus:ring-void-danger' : ''}"
      />
      {#if error}
        <p class="mt-1 text-xs text-void-danger">{error}</p>
      {/if}
    </div>

    <Button variant="solid" size="lg" onclick={handleConfirm} disabled={!displayName.trim()}>
      Entrar no Concord P2P
    </Button>
  </div>
</div>
