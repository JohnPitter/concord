<script lang="ts">
  import Avatar from '../ui/Avatar.svelte'
  import Tooltip from '../ui/Tooltip.svelte'

  interface Member {
    id: string
    name: string
    avatarUrl?: string
    status: 'online' | 'idle' | 'dnd' | 'offline'
    role?: string
  }

  let {
    members,
    currentUserId = '',
    currentUserRole = 'member',
    onUpdateRole,
    onKickMember,
  }: {
    members: Member[]
    currentUserId?: string
    currentUserRole?: string
    onUpdateRole?: (memberId: string, role: string) => void
    onKickMember?: (memberId: string) => void
  } = $props()

  const canManage = $derived(
    currentUserRole === 'owner' || currentUserRole === 'admin'
  )

  const onlineMembers = $derived(members.filter(m => m.status !== 'offline'))
  const offlineMembers = $derived(members.filter(m => m.status === 'offline'))

  let roleMenuOpen = $state<string | null>(null)

  function toggleRoleMenu(memberId: string) {
    roleMenuOpen = roleMenuOpen === memberId ? null : memberId
  }

  const availableRoles = ['admin', 'moderator', 'member'] as const
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
          <div class="group relative">
            <button
              class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 transition-colors hover:bg-void-bg-hover cursor-pointer"
              onclick={() => canManage && member.id !== currentUserId ? toggleRoleMenu(member.id) : null}
            >
              {#if member.avatarUrl}
                <div class="relative shrink-0">
                  <img src={member.avatarUrl} alt={member.name} class="h-8 w-8 rounded-full object-cover" />
                  <span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-void-bg-secondary bg-void-online"></span>
                </div>
              {:else}
                <Avatar name={member.name} size="sm" status={member.status} />
              {/if}
              <div class="flex-1 min-w-0 text-left">
                <p class="text-sm font-medium text-void-text-primary truncate">{member.name}</p>
                {#if member.role}
                  <p class="text-[11px] text-void-text-muted">{member.role}</p>
                {/if}
              </div>
              {#if canManage && member.id !== currentUserId && member.role !== 'Owner'}
                <svg class="h-3.5 w-3.5 text-void-text-muted opacity-0 group-hover:opacity-100 shrink-0 transition-opacity" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>
                </svg>
              {/if}
            </button>
            <!-- Role management popover -->
            {#if roleMenuOpen === member.id && canManage && member.role !== 'Owner'}
              <div class="absolute right-0 top-full mt-1 z-30 w-44 rounded-lg border border-void-border bg-void-bg-primary shadow-md p-1 animate-fade-in-down">
                <p class="px-2 py-1 text-[10px] font-bold uppercase tracking-wide text-void-text-muted">Alterar cargo</p>
                {#each availableRoles as role}
                  <button
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors cursor-pointer
                      {member.role === role.charAt(0).toUpperCase() + role.slice(1)
                        ? 'bg-void-accent/10 text-void-accent'
                        : 'text-void-text-secondary hover:bg-void-bg-hover hover:text-void-text-primary'}"
                    onclick={() => { onUpdateRole?.(member.id, role); roleMenuOpen = null }}
                  >
                    {role.charAt(0).toUpperCase() + role.slice(1)}
                  </button>
                {/each}
                <div class="border-t border-void-border my-1"></div>
                <button
                  class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-xs text-void-danger hover:bg-void-danger/10 transition-colors cursor-pointer"
                  onclick={() => { onKickMember?.(member.id); roleMenuOpen = null }}
                >
                  Expulsar
                </button>
              </div>
            {/if}
          </div>
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
            {#if member.avatarUrl}
              <div class="relative shrink-0">
                <img src={member.avatarUrl} alt={member.name} class="h-8 w-8 rounded-full object-cover opacity-50" />
              </div>
            {:else}
              <Avatar name={member.name} size="sm" status="offline" />
            {/if}
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
