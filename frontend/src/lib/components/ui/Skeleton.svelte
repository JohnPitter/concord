<script lang="ts">
  let { className = '' }: { className?: string } = $props()
</script>

<div class={`skeleton ${className}`} aria-hidden="true"></div>
