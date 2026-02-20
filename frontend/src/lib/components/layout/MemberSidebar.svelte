<script lang="ts">
  import Avatar from '../ui/Avatar.svelte'

  interface Member {
    id: string
    name: string
    status: 'online' | 'idle' | 'dnd' | 'offline'
    role?: string
  }

  let { members }: { members: Member[] } = $props()

  const onlineMembers = $derived(members.filter(m => m.status !== 'offline'))
  const offlineMembers = $derived(members.filter(m => m.status === 'offline'))
</script>

<aside class="flex h-full w-60 flex-col bg-void-bg-secondary overflow-y-auto">
  <div class="p-4">
    <!-- Online members -->
    {#if onlineMembers.length > 0}
      <h3 class="mb-2 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
        Online — {onlineMembers.length}
      </h3>
      <div class="space-y-0.5">
        {#each onlineMembers as member}
          <button class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 transition-colors hover:bg-void-bg-hover cursor-pointer">
            <Avatar name={member.name} size="sm" status={member.status} />
            <div class="flex-1 min-w-0 text-left">
              <p class="text-sm font-medium text-void-text-primary truncate">{member.name}</p>
              {#if member.role}
                <p class="text-[11px] text-void-text-muted">{member.role}</p>
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}

    <!-- Offline members -->
    {#if offlineMembers.length > 0}
      <h3 class="mb-2 mt-4 text-[11px] font-bold uppercase tracking-wide text-void-text-muted">
        Offline — {offlineMembers.length}
      </h3>
      <div class="space-y-0.5">
        {#each offlineMembers as member}
          <button class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 opacity-50 transition-colors hover:bg-void-bg-hover hover:opacity-70 cursor-pointer">
            <Avatar name={member.name} size="sm" status="offline" />
            <div class="flex-1 min-w-0 text-left">
              <p class="text-sm font-medium text-void-text-primary truncate">{member.name}</p>
              {#if member.role}
                <p class="text-[11px] text-void-text-muted">{member.role}</p>
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</aside>
