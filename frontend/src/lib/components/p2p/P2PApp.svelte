<script lang="ts">
  import P2PPeerSidebar from './P2PPeerSidebar.svelte'
  import P2PChatArea from './P2PChatArea.svelte'
  import SettingsPanel from '../settings/SettingsPanel.svelte'
  import {
    getP2P, initP2PStore, setActivePeer, sendMessage, joinRoom, stopP2PStore,
    type P2PPeer, type P2PMessage,
  } from '../../stores/p2p.svelte'

  let {
    profile,
  }: {
    profile: { displayName: string; avatarDataUrl?: string } | null
  } = $props()

  const p2p = getP2P()
  let showSettings = $state(false)

  $effect(() => {
    initP2PStore(profile)
    return () => stopP2PStore()
  })

  const activePeer = $derived(p2p.peers.find(p => p.id === p2p.activePeerID) ?? null)
  const peerMessages = $derived(p2p.activePeerID ? (p2p.messages[p2p.activePeerID] ?? []) : [])
</script>

<div class="flex h-screen w-screen overflow-hidden">
  <P2PPeerSidebar
    peers={p2p.peers}
    activePeerID={p2p.activePeerID}
    roomCode={p2p.roomCode}
    {profile}
    onSelectPeer={(id) => setActivePeer(id)}
    onJoinRoom={(code) => joinRoom(code)}
    onOpenSettings={() => showSettings = true}
  />
  <P2PChatArea
    peer={activePeer}
    messages={peerMessages}
    sending={p2p.sending}
    onSend={(content) => p2p.activePeerID && sendMessage(p2p.activePeerID, content)}
  />
</div>

<SettingsPanel
  bind:open={showSettings}
  currentUser={profile ? { username: profile.displayName, display_name: profile.displayName, avatar_url: profile.avatarDataUrl } : null}
  onLogout={() => {}}
/>
