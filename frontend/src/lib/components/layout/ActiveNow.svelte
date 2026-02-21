<script lang="ts">
  import type { Friend } from '../../stores/friends.svelte'

  let {
    friends,
  }: {
    friends: Friend[]
  } = $props()

  // Only show friends with activities or streaming
  const active = $derived(
    friends.filter(f => f.status !== 'offline' && (f.game || f.streaming || f.activity))
  )
</script>

<aside class="flex h-full w-[340px] shrink-0 flex-col border-l border-void-border bg-void-bg-secondary overflow-y-auto">
  <div class="px-4 pt-5 pb-2 shrink-0">
    <h3 class="text-xs font-bold uppercase tracking-wide text-void-text-primary">Ativo agora</h3>
  </div>

  {#if active.length === 0}
    <div class="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <div class="flex h-16 w-16 items-center justify-center rounded-full bg-void-bg-hover">
        <svg class="h-8 w-8 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
          <circle cx="9" cy="7" r="4"/>
        </svg>
      </div>
      <p class="text-sm font-semibold text-void-text-primary">Está meio quieto por aqui</p>
      <p class="text-xs text-void-text-muted">Quando um amigo iniciar uma atividade, como jogar um game ou assistir algo, você verá aqui!</p>
    </div>
  {:else}
    <div class="flex flex-col gap-3 px-3 pb-4">
      {#each active as friend}
        <div class="rounded-xl overflow-hidden bg-void-bg-tertiary">
          {#if friend.streaming}
            <!-- Streaming card with preview placeholder -->
            <div class="relative h-36 bg-void-bg-primary flex items-center justify-center overflow-hidden">
              <!-- Stream preview placeholder (dark gradient) -->
              <div class="absolute inset-0 bg-gradient-to-br from-void-bg-primary via-void-bg-hover to-void-bg-secondary"></div>
              <!-- Game/stream art placeholder -->
              <div class="relative z-10 flex flex-col items-center gap-2 text-void-text-muted">
                <svg class="h-10 w-10 opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <polygon points="23 7 16 12 23 17 23 7"/>
                  <rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
                </svg>
              </div>
              <!-- Live badge -->
              <div class="absolute top-2 left-2 flex items-center gap-1 rounded-md bg-red-600 px-1.5 py-0.5">
                <span class="h-1.5 w-1.5 rounded-full bg-white animate-pulse"></span>
                <span class="text-[10px] font-bold text-white uppercase tracking-wide">Ao vivo</span>
              </div>
              <!-- User count badge -->
              <div class="absolute top-2 right-2 flex items-center gap-1 rounded-md bg-black/60 px-1.5 py-0.5">
                <svg class="h-3 w-3 text-white" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                </svg>
                <span class="text-[10px] font-bold text-white">
                  {Math.floor(Math.random() * 50) + 2}
                </span>
              </div>
            </div>
            <div class="p-3">
              <div class="flex items-center gap-2 mb-1">
                <div class="relative shrink-0">
                  {#if friend.avatar_url}
                    <img src={friend.avatar_url} alt={friend.display_name} class="h-6 w-6 rounded-full object-cover" />
                  {:else}
                    <div class="h-6 w-6 rounded-full bg-void-accent flex items-center justify-center text-[10px] font-bold text-white">
                      {friend.display_name.slice(0, 2).toUpperCase()}
                    </div>
                  {/if}
                  <span class="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border border-void-bg-tertiary bg-void-online"></span>
                </div>
                <span class="text-xs font-semibold text-void-text-primary truncate">{friend.display_name}</span>
              </div>
              {#if friend.streamTitle}
                <p class="text-xs text-void-text-secondary truncate font-medium">{friend.streamTitle}</p>
              {/if}
              <p class="text-[11px] text-void-text-muted mt-0.5">{friend.activity}</p>
            </div>
          {:else if friend.game}
            <!-- Game card -->
            <div class="p-3">
              <div class="flex items-center gap-2 mb-2">
                <div class="relative shrink-0">
                  {#if friend.avatar_url}
                    <img src={friend.avatar_url} alt={friend.display_name} class="h-8 w-8 rounded-full object-cover" />
                  {:else}
                    <div class="h-8 w-8 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
                      {friend.display_name.slice(0, 2).toUpperCase()}
                    </div>
                  {/if}
                  <span class="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border border-void-bg-tertiary bg-void-online"></span>
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-semibold text-void-text-primary truncate">{friend.display_name}</p>
                  <p class="text-[11px] text-void-text-muted truncate">{friend.activity}</p>
                </div>
              </div>

              <!-- Game art placeholder -->
              <div class="flex gap-2 items-start">
                <div class="h-14 w-11 shrink-0 rounded bg-void-bg-hover flex items-center justify-center">
                  <svg class="h-6 w-6 text-void-text-muted opacity-40" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M21 6H3a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h18a1 1 0 0 0 1-1V7a1 1 0 0 0-1-1zm-10 8H8v-2H6v-2h2V8h2v2h2v2h-2v2zm4.5 1c-.83 0-1.5-.67-1.5-1.5S14.67 12 15.5 12s1.5.67 1.5 1.5S16.33 15 15.5 15zm3-3c-.83 0-1.5-.67-1.5-1.5S17.67 9 18.5 9s1.5.67 1.5 1.5S19.33 12 18.5 12z"/>
                  </svg>
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-bold text-void-text-primary truncate">{friend.game}</p>
                  {#if friend.gameSince}
                    <p class="text-[11px] text-void-text-muted mt-0.5">{friend.gameSince}</p>
                  {/if}
                  <button class="mt-1.5 rounded bg-void-accent/20 px-2 py-0.5 text-[11px] font-medium text-void-accent hover:bg-void-accent/30 transition-colors cursor-pointer">
                    Jogar junto
                  </button>
                </div>
              </div>
            </div>
          {:else}
            <!-- Generic activity -->
            <div class="flex items-center gap-2 p-3">
              <div class="relative shrink-0">
                {#if friend.avatar_url}
                  <img src={friend.avatar_url} alt={friend.display_name} class="h-8 w-8 rounded-full object-cover" />
                {:else}
                  <div class="h-8 w-8 rounded-full bg-void-accent flex items-center justify-center text-xs font-bold text-white">
                    {friend.display_name.slice(0, 2).toUpperCase()}
                  </div>
                {/if}
                <span class="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border border-void-bg-tertiary bg-void-online"></span>
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-xs font-semibold text-void-text-primary truncate">{friend.display_name}</p>
                <p class="text-[11px] text-void-text-muted truncate">{friend.activity}</p>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</aside>
