<script lang="ts">
  import { getAuth, startLogin, pollForCompletion, clearError, cancelLogin } from '../../stores/auth.svelte'
  import { translations, t } from '../../i18n'
  import Button from '../ui/Button.svelte'
  import logoPng from '../../../assets/logo.png'

  const auth = getAuth()
  const trans = $derived($translations)

  async function handleLogin() {
    await startLogin()
  }

  function handleCopyCode() {
    if (auth.deviceCode?.user_code) {
      navigator.clipboard.writeText(auth.deviceCode.user_code)
    }
  }

  function handleOpenGitHub() {
    if (auth.deviceCode?.verification_uri) {
      // @ts-ignore - Wails runtime binding
      window.runtime?.BrowserOpenURL(auth.deviceCode.verification_uri)
    }
  }
</script>

<div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
  <div class="w-full max-w-md space-y-8 px-6">
    <!-- Logo / Branding -->
    <div class="text-center">
      <div class="mx-auto mb-4 h-16 w-16">
        <img src={logoPng} alt="Concord" class="h-16 w-16" />
      </div>
      <h1 class="text-2xl font-bold text-void-text-primary">{t(trans, 'auth.welcome')}</h1>
      <p class="mt-2 text-sm text-void-text-muted">{t(trans, 'auth.subtitle')}</p>
    </div>

    <!-- Error display -->
    {#if auth.error}
      <div class="rounded-lg border border-void-danger/30 bg-void-danger/10 px-4 py-3">
        <p class="text-sm text-void-danger">{auth.error}</p>
        <button
          class="mt-1 text-xs text-void-text-muted underline hover:text-void-text-secondary"
          onclick={clearError}
        >
          {t(trans, 'auth.dismiss')}
        </button>
      </div>
    {/if}

    <!-- Device Code Display -->
    {#if auth.deviceCode && !auth.polling}
      <div class="space-y-4 rounded-xl border border-void-border bg-void-bg-secondary p-6">
        <p class="text-center text-sm text-void-text-secondary">
          {t(trans, 'auth.enterCode')}
        </p>
        <div class="flex items-center justify-center gap-2">
          <code class="rounded-lg bg-void-bg-tertiary px-6 py-3 font-mono text-2xl font-bold tracking-widest text-void-accent">
            {auth.deviceCode.user_code}
          </code>
          <!-- svelte-ignore a11y_consider_explicit_label -->
          <button
            class="rounded-lg p-2 text-void-text-muted transition-colors hover:bg-void-bg-hover hover:text-void-text-primary"
            onclick={handleCopyCode}
            aria-label={t(trans, 'auth.copyCode')}
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
        </div>
        <div class="flex flex-col gap-2">
          <Button variant="solid" onclick={handleOpenGitHub}>
            {t(trans, 'auth.openGithub')}
          </Button>
          <Button variant="ghost" onclick={() => pollForCompletion()}>
            {t(trans, 'auth.enteredCode')}
          </Button>
        </div>
        <button
          class="w-full text-center text-xs text-void-text-muted hover:text-void-text-secondary"
          onclick={cancelLogin}
        >
          {t(trans, 'auth.cancel')}
        </button>
      </div>
    {:else if auth.polling}
      <!-- Polling state -->
      <div class="space-y-4 rounded-xl border border-void-border bg-void-bg-secondary p-6 text-center">
        <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
        <p class="text-sm text-void-text-secondary">{t(trans, 'auth.waitingAuth')}</p>
        <p class="text-xs text-void-text-muted">{t(trans, 'auth.completeAuth')}</p>
      </div>
    {:else}
      <!-- Login button -->
      <div class="space-y-4">
        <Button variant="solid" size="lg" onclick={handleLogin} class="w-full">
          <svg class="mr-2 h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
          </svg>
          {t(trans, 'auth.signInGithub')}
        </Button>
        <p class="text-center text-xs text-void-text-muted">
          {t(trans, 'auth.deviceFlow')}
        </p>
      </div>
    {/if}
  </div>
</div>
