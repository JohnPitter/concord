<script lang="ts">
  let { channelName }: { channelName: string } = $props()
</script>

<main class="flex h-full flex-1 flex-col bg-void-bg-tertiary">
  <!-- Channel header -->
  <header class="flex h-12 items-center gap-2 border-b border-void-border px-4 shrink-0">
    <svg class="h-5 w-5 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <line x1="4" y1="9" x2="20" y2="9" />
      <line x1="4" y1="15" x2="20" y2="15" />
      <line x1="10" y1="3" x2="8" y2="21" />
      <line x1="16" y1="3" x2="14" y2="21" />
    </svg>
    <span class="font-bold text-void-text-primary">{channelName}</span>
    <div class="mx-2 h-6 w-px bg-void-border"></div>
    <span class="text-sm text-void-text-muted">Welcome to #{channelName}</span>

    <!-- Header actions -->
    <div class="ml-auto flex items-center gap-2">
      <button aria-label="Search" class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:text-void-text-primary cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
      </button>
      <button aria-label="Members" class="rounded-md p-1.5 text-void-text-secondary transition-colors hover:text-void-text-primary cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
          <circle cx="9" cy="7" r="4" />
          <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
          <path d="M16 3.13a4 4 0 0 1 0 7.75" />
        </svg>
      </button>
    </div>
  </header>

  <!-- Messages area (placeholder) -->
  <div class="flex flex-1 flex-col overflow-y-auto p-4">
    <div class="flex-1"></div>

    <!-- Welcome message -->
    <div class="mb-6">
      <div class="mb-2 flex h-16 w-16 items-center justify-center rounded-full bg-void-bg-hover">
        <svg class="h-8 w-8 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="4" y1="9" x2="20" y2="9" />
          <line x1="4" y1="15" x2="20" y2="15" />
          <line x1="10" y1="3" x2="8" y2="21" />
          <line x1="16" y1="3" x2="14" y2="21" />
        </svg>
      </div>
      <h2 class="text-3xl font-bold text-void-text-primary">Welcome to #{channelName}</h2>
      <p class="mt-1 text-void-text-muted">This is the start of the #{channelName} channel.</p>
    </div>

    <!-- Mock messages -->
    <div class="space-y-4">
      <div class="group flex gap-3 rounded-md px-2 py-1 hover:bg-void-bg-hover/50">
        <div class="h-10 w-10 shrink-0 rounded-full bg-void-accent flex items-center justify-center text-sm font-bold text-white">AL</div>
        <div>
          <div class="flex items-baseline gap-2">
            <span class="font-medium text-void-accent-hover">Alice</span>
            <span class="text-[11px] text-void-text-muted">Today at 14:30</span>
          </div>
          <p class="text-sm text-void-text-primary">Hey everyone! Ready for tonight's raid?</p>
        </div>
      </div>
      <div class="group flex gap-3 rounded-md px-2 py-1 hover:bg-void-bg-hover/50">
        <div class="h-10 w-10 shrink-0 rounded-full bg-void-online flex items-center justify-center text-sm font-bold text-white">BO</div>
        <div>
          <div class="flex items-baseline gap-2">
            <span class="font-medium text-void-online">Bob</span>
            <span class="text-[11px] text-void-text-muted">Today at 14:32</span>
          </div>
          <p class="text-sm text-void-text-primary">Let's go! I'll be on voice in 5 min</p>
        </div>
      </div>
      <div class="group flex gap-3 rounded-md px-2 py-1 hover:bg-void-bg-hover/50">
        <div class="h-10 w-10 shrink-0 rounded-full bg-void-warning flex items-center justify-center text-sm font-bold text-white">EV</div>
        <div>
          <div class="flex items-baseline gap-2">
            <span class="font-medium text-void-warning">Eve</span>
            <span class="text-[11px] text-void-text-muted">Today at 14:35</span>
          </div>
          <p class="text-sm text-void-text-primary">Count me in. What comp are we running?</p>
        </div>
      </div>
    </div>
  </div>

  <!-- Message input -->
  <div class="shrink-0 px-4 pb-6">
    <div class="flex items-center gap-2 rounded-lg bg-void-bg-hover px-4 py-2.5">
      <button aria-label="Attach file" class="text-void-text-muted transition-colors hover:text-void-text-primary cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="16" />
          <line x1="8" y1="12" x2="16" y2="12" />
        </svg>
      </button>
      <input
        type="text"
        class="flex-1 bg-transparent text-sm text-void-text-primary placeholder:text-void-text-muted focus:outline-none"
        placeholder="Message #{channelName}"
      />
      <button aria-label="Emoji" class="text-void-text-muted transition-colors hover:text-void-text-primary cursor-pointer">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <path d="M8 14s1.5 2 4 2 4-2 4-2" />
          <line x1="9" y1="9" x2="9.01" y2="9" />
          <line x1="15" y1="9" x2="15.01" y2="9" />
        </svg>
      </button>
    </div>
  </div>
</main>
