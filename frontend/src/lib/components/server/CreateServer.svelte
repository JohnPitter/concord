<script lang="ts">
  import Button from '../ui/Button.svelte'
  import Input from '../ui/Input.svelte'
  import Modal from '../ui/Modal.svelte'

  let {
    open = $bindable(false),
    onCreate,
  }: {
    open: boolean
    onCreate: (name: string) => void
  } = $props()

  let serverName = $state('')
  let error = $state('')

  function handleCreate() {
    const trimmed = serverName.trim()
    if (!trimmed) {
      error = 'Server name is required'
      return
    }
    if (trimmed.length > 100) {
      error = 'Server name cannot exceed 100 characters'
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

<Modal bind:open title="Create a Server">
  <div class="space-y-4">
    <p class="text-sm text-void-text-secondary">
      Give your new server a name. You can always change it later.
    </p>

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div onkeydown={handleKeydown}>
      <Input
        placeholder="My Awesome Server"
        bind:value={serverName}
        error={error || undefined}
      />
    </div>

    <div class="flex justify-end gap-2">
      <Button variant="ghost" onclick={() => { open = false; serverName = ''; error = '' }}>
        Cancel
      </Button>
      <Button variant="solid" onclick={handleCreate} disabled={!serverName.trim()}>
        Create Server
      </Button>
    </div>
  </div>
</Modal>
