<script lang="ts">
  import Button from '../ui/Button.svelte'
  import Input from '../ui/Input.svelte'
  import Modal from '../ui/Modal.svelte'
  import { translations, t } from '../../i18n'

  let {
    open = $bindable(false),
    onJoin,
  }: {
    open: boolean
    onJoin: (code: string) => void
  } = $props()

  let inviteCode = $state('')
  let error = $state('')
  const trans = $derived($translations)

  function handleJoin() {
    const trimmed = inviteCode.trim()
    if (!trimmed) {
      error = t(trans, 'server.inviteRequired')
      return
    }
    error = ''
    onJoin(trimmed)
    inviteCode = ''
    open = false
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleJoin()
    }
  }
</script>

<Modal bind:open title={t(trans, 'server.joinTitle')}>
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

    <div class="flex justify-end gap-2">
      <Button variant="ghost" onclick={() => { open = false; inviteCode = ''; error = '' }}>
        {t(trans, 'common.cancel')}
      </Button>
      <Button variant="solid" onclick={handleJoin} disabled={!inviteCode.trim()}>
        {t(trans, 'server.joinButton')}
      </Button>
    </div>
  </div>
</Modal>
