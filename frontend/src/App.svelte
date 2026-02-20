<script lang="ts">
  import Button from './lib/components/ui/Button.svelte'
  import Input from './lib/components/ui/Input.svelte'
  import Modal from './lib/components/ui/Modal.svelte'
  import Badge from './lib/components/ui/Badge.svelte'
  import Avatar from './lib/components/ui/Avatar.svelte'
  import Tooltip from './lib/components/ui/Tooltip.svelte'
  import Toggle from './lib/components/ui/Toggle.svelte'
  import Dropdown from './lib/components/ui/Dropdown.svelte'
  import Card from './lib/components/ui/Card.svelte'

  let modalOpen = $state(false)
  let toggleChecked = $state(false)
  let selectedItem = $state('Option 1')

  const dropdownItems = [
    { label: 'Option 1', value: 'Option 1' },
    { label: 'Option 2', value: 'Option 2' },
    { label: 'Option 3', value: 'Option 3' },
  ]
</script>

<main class="min-h-screen bg-void-bg-primary p-8">
  <div class="mx-auto max-w-4xl space-y-10">
    <header>
      <h1 class="text-3xl font-bold text-void-text-primary">Void Design System</h1>
      <p class="mt-1 text-void-text-secondary">Concord — Phase 1.2</p>
    </header>

    <!-- Buttons -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Button</h2>
      <div class="flex flex-wrap items-center gap-3">
        <Button>Solid</Button>
        <Button variant="outline">Outline</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="danger">Danger</Button>
        <Button size="sm">Small</Button>
        <Button size="lg">Large</Button>
        <Button loading>Loading</Button>
        <Button disabled>Disabled</Button>
      </div>
    </section>

    <!-- Input -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Input</h2>
      <div class="flex max-w-sm flex-col gap-3">
        <Input placeholder="Default input" />
        <Input type="password" placeholder="Password" />
        <Input placeholder="With error" error="This field is required" />
        <Input placeholder="Disabled" disabled />
      </div>
    </section>

    <!-- Modal -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Modal</h2>
      <Button onclick={() => modalOpen = true}>Open Modal</Button>
      <Modal bind:open={modalOpen} title="Example Modal">
        <p class="text-void-text-secondary">This is a modal dialog with focus trap and Escape key support.</p>
        <div class="mt-4 flex justify-end gap-2">
          <Button variant="ghost" onclick={() => modalOpen = false}>Cancel</Button>
          <Button onclick={() => modalOpen = false}>Confirm</Button>
        </div>
      </Modal>
    </section>

    <!-- Badge -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Badge</h2>
      <div class="flex flex-wrap gap-2">
        <Badge>Default</Badge>
        <Badge variant="success">Success</Badge>
        <Badge variant="warning">Warning</Badge>
        <Badge variant="danger">Danger</Badge>
      </div>
    </section>

    <!-- Avatar -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Avatar</h2>
      <div class="flex items-center gap-3">
        <Avatar name="John" size="sm" status="online" />
        <Avatar name="Alice" status="idle" />
        <Avatar name="Bob" size="lg" status="dnd" />
        <Avatar name="Eve" size="lg" status="offline" />
      </div>
    </section>

    <!-- Tooltip -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Tooltip</h2>
      <div class="flex gap-4">
        <Tooltip text="Top tooltip">
          <Button variant="outline">Top</Button>
        </Tooltip>
        <Tooltip text="Bottom tooltip" position="bottom">
          <Button variant="outline">Bottom</Button>
        </Tooltip>
        <Tooltip text="Left tooltip" position="left">
          <Button variant="outline">Left</Button>
        </Tooltip>
        <Tooltip text="Right tooltip" position="right">
          <Button variant="outline">Right</Button>
        </Tooltip>
      </div>
    </section>

    <!-- Toggle -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Toggle</h2>
      <div class="flex flex-col gap-3">
        <Toggle bind:checked={toggleChecked} label="Notifications" />
        <Toggle checked={true} label="Always on" disabled />
      </div>
    </section>

    <!-- Dropdown -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Dropdown</h2>
      <div class="max-w-xs">
        <Dropdown items={dropdownItems} bind:selected={selectedItem} />
      </div>
    </section>

    <!-- Card -->
    <section class="space-y-3">
      <h2 class="text-lg font-bold text-void-text-primary">Card</h2>
      <div class="grid grid-cols-2 gap-4">
        <Card>
          <h3 class="font-bold text-void-text-primary">Static Card</h3>
          <p class="mt-1 text-sm text-void-text-secondary">Basic card component.</p>
        </Card>
        <Card interactive>
          <h3 class="font-bold text-void-text-primary">Interactive Card</h3>
          <p class="mt-1 text-sm text-void-text-secondary">Hover to see the effect.</p>
        </Card>
      </div>
    </section>
  </div>
</main>
