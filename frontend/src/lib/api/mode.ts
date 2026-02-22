// Helper to check if we're in server mode (HTTP API) vs P2P mode (Wails bindings)

import { getSettings } from '../stores/settings.svelte'

export function isServerMode(): boolean {
  const settings = getSettings()
  return settings.networkMode === 'server'
}
