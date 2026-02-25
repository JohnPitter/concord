<script lang="ts">
  let {
    stream,
    username,
    local = false,
  }: {
    stream: MediaStream
    username: string
    local?: boolean
  } = $props()

  let videoEl = $state<HTMLVideoElement | null>(null)

  $effect(() => {
    if (!videoEl) return
    if (videoEl.srcObject !== stream) {
      videoEl.srcObject = stream
    }
    videoEl.muted = true
  })
</script>

<div class="rounded-lg border border-void-border bg-void-bg-primary p-1.5">
  <div class="mb-1 flex items-center justify-between">
    <span class="truncate text-[10px] font-semibold text-void-text-primary">{username}</span>
    {#if local}
      <span class="rounded bg-void-accent/20 px-1 py-0.5 text-[9px] font-semibold text-void-accent">you</span>
    {/if}
  </div>
  <video
    bind:this={videoEl}
    autoplay
    playsinline
    muted
    class="h-24 w-full rounded-md bg-black object-cover"
  ></video>
</div>
