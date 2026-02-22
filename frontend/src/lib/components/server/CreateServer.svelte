<script lang="ts">
  import Button from '../ui/Button.svelte'
  import Input from '../ui/Input.svelte'
  import Modal from '../ui/Modal.svelte'
  import { translations, t } from '../../i18n'

  let {
    open = $bindable(false),
    onCreate,
    onJoin,
  }: {
    open: boolean
    onCreate: (name: string) => void
    onJoin: (code: string) => Promise<string | null>
  } = $props()

  let mode = $state<'choose' | 'create' | 'join'>('choose')
  let serverName = $state('')
  let inviteCode = $state('')
  let error = $state('')
  let joining = $state(false)
  const trans = $derived($translations)

  function resetState() {
    mode = 'choose'
    serverName = ''
    inviteCode = ''
    error = ''
  }

  function handleCreate() {
    const trimmed = serverName.trim()
    if (!trimmed) {
      error = t(trans, 'server.nameRequired')
      return
    }
    if (trimmed.length > 100) {
      error = t(trans, 'server.nameMaxLength')
      return
    }
    error = ''
    onCreate(trimmed)
    resetState()
    open = false
  }

  async function handleJoin() {
    const trimmed = inviteCode.trim()
    if (!trimmed) {
      error = t(trans, 'server.inviteRequired')
      return
    }
    error = ''
    joining = true
    try {
      const result = await onJoin(trimmed)
      if (result) {
        // result is an error message
        error = result
      } else {
        // success — close modal
        resetState()
        open = false
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to join server'
    } finally {
      joining = false
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      if (mode === 'create') handleCreate()
      else if (mode === 'join') handleJoin()
    }
  }

  // Reset mode when modal closes
  $effect(() => {
    if (!open) resetState()
  })

  const modalTitle = $derived(
    mode === 'create' ? t(trans, 'server.createTitle')
    : mode === 'join' ? t(trans, 'server.joinTitle')
    : t(trans, 'server.addServerTitle')
  )
</script>

<Modal bind:open title={modalTitle}>
  {#if mode === 'choose'}
    <!-- Choose: Create or Join -->
    <div class="space-y-3">
      <button
        onclick={() => { mode = 'create'; error = '' }}
        class="w-full flex items-center gap-4 rounded-lg border border-void-border bg-void-bg-secondary p-4 transition-colors hover:bg-void-bg-hover hover:border-void-accent/50 cursor-pointer group"
      >
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-void-accent/10 text-void-accent group-hover:bg-void-accent/20 transition-colors">
          <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </div>
        <div class="text-left">
          <p class="font-semibold text-void-text-primary text-sm">{t(trans, 'server.createOption')}</p>
          <p class="text-xs text-void-text-muted mt-0.5">{t(trans, 'server.createOptionDesc')}</p>
        </div>
        <svg class="ml-auto h-4 w-4 text-void-text-muted group-hover:text-void-text-secondary transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>

      <button
        onclick={() => { mode = 'join'; error = '' }}
        class="w-full flex items-center gap-4 rounded-lg border border-void-border bg-void-bg-secondary p-4 transition-colors hover:bg-void-bg-hover hover:border-void-accent/50 cursor-pointer group"
      >
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-void-accent/10 text-void-accent group-hover:bg-void-accent/20 transition-colors">
          <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
            <polyline points="10 17 15 12 10 7" />
            <line x1="15" y1="12" x2="3" y2="12" />
          </svg>
        </div>
        <div class="text-left">
          <p class="font-semibold text-void-text-primary text-sm">{t(trans, 'server.joinOption')}</p>
          <p class="text-xs text-void-text-muted mt-0.5">{t(trans, 'server.joinOptionDesc')}</p>
        </div>
        <svg class="ml-auto h-4 w-4 text-void-text-muted group-hover:text-void-text-secondary transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>
    </div>

  {:else if mode === 'create'}
    <!-- Create Server form -->
    <div class="space-y-4">
      <p class="text-sm text-void-text-secondary">
        {t(trans, 'server.createDesc')}
      </p>

      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div onkeydown={handleKeydown}>
        <Input
          placeholder={t(trans, 'server.namePlaceholder')}
          bind:value={serverName}
          error={error || undefined}
        />
      </div>

      <div class="flex justify-between">
        <Button variant="ghost" onclick={() => { mode = 'choose'; error = '' }}>
          {t(trans, 'server.back')}
        </Button>
        <div class="flex gap-2">
          <Button variant="ghost" onclick={() => { open = false }}>
            {t(trans, 'common.cancel')}
          </Button>
          <Button variant="solid" onclick={handleCreate} disabled={!serverName.trim()}>
            {t(trans, 'server.createButton')}
          </Button>
        </div>
      </div>
    </div>

  {:else}
    <!-- Join Server form -->
    <div class="space-y-4">
      <p class="text-sm text-void-text-secondary">
        {t(trans, 'server.joinDesc')}
      </p>

      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div onkeydown={handleKeydown}>
        <Input
          placeholder={t(trans, 'server.invitePlaceholder')}
          bind:value={inviteCode}
          error={error || undefined}
        />
      </div>

      <div class="flex justify-between">
        <Button variant="ghost" onclick={() => { mode = 'choose'; error = '' }}>
          {t(trans, 'server.back')}
        </Button>
        <div class="flex gap-2">
          <Button variant="ghost" onclick={() => { open = false }}>
            {t(trans, 'common.cancel')}
          </Button>
          <Button variant="solid" onclick={handleJoin} disabled={!inviteCode.trim() || joining}>
            {#if joining}
              <div class="flex items-center gap-2">
                <div class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
                {t(trans, 'server.joinButton')}
              </div>
            {:else}
              {t(trans, 'server.joinButton')}
            {/if}
          </Button>
        </div>
      </div>
    </div>
  {/if}
</Modal>
