<script lang="ts">
  import Button from '../ui/Button.svelte'
  import Input from '../ui/Input.svelte'
  import Modal from '../ui/Modal.svelte'

  let {
    open = $bindable(false),
    onJoin,
  }: {
    open: boolean
    onJoin: (code: string) => void
  } = $props()

  let inviteCode = $state('')
  let error = $state('')

  function handleJoin() {
    const trimmed = inviteCode.trim()
    if (!trimmed) {
      error = 'Invite code is required'
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

<Modal bind:open title="Join a Server">
  <div class="space-y-4">
    <p class="text-sm text-void-text-secondary">
      Enter an invite code to join an existing server.
    </p>

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div onkeydown={handleKeydown}>
      <Input
        placeholder="Enter invite code"
        bind:value={inviteCode}
        error={error || undefined}
      />
    </div>

    <div class="flex justify-end gap-2">
      <Button variant="ghost" onclick={() => { open = false; inviteCode = ''; error = '' }}>
        Cancel
      </Button>
      <Button variant="solid" onclick={handleJoin} disabled={!inviteCode.trim()}>
        Join Server
      </Button>
    </div>
  </div>
</Modal>
