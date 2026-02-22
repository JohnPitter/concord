<script lang="ts">
  import Button from '../ui/Button.svelte'
  import Input from '../ui/Input.svelte'
  import Modal from '../ui/Modal.svelte'
  import { translations, t } from '../../i18n'

  let {
    open = $bindable(false),
    onCreate,
  }: {
    open: boolean
    onCreate: (name: string) => void
  } = $props()

  let serverName = $state('')
  let error = $state('')
  const trans = $derived($translations)

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
    serverName = ''
    open = false
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleCreate()
    }
  }
</script>

<Modal bind:open title={t(trans, 'server.createTitle')}>
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

    <div class="flex justify-end gap-2">
      <Button variant="ghost" onclick={() => { open = false; serverName = ''; error = '' }}>
        {t(trans, 'common.cancel')}
      </Button>
      <Button variant="solid" onclick={handleCreate} disabled={!serverName.trim()}>
        {t(trans, 'server.createButton')}
      </Button>
    </div>
  </div>
</Modal>
