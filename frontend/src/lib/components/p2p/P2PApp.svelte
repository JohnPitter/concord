<script lang="ts">
  import P2PPeerSidebar from './P2PPeerSidebar.svelte'
  import P2PChatArea from './P2PChatArea.svelte'
  import SettingsPanel from '../settings/SettingsPanel.svelte'

  interface P2PPeer {
    id: string
    displayName: string
    avatarDataUrl?: string
    connected: boolean
    source: 'lan' | 'room'
  }

  interface P2PMessage {
    id: string
    peerID: string
    direction: 'sent' | 'received'
    content: string
    sentAt: string
  }

  let {
    profile,
  }: {
    profile: { displayName: string; avatarDataUrl?: string } | null
  } = $props()

  // Local temporary state -- will be replaced by the p2p store
  let peers = $state<P2PPeer[]>([])
  let activePeerID = $state<string | null>(null)
  let messages = $state<P2PMessage[]>([])
  let roomCode = $state('carregando...')
  let sending = $state(false)
  let showSettings = $state(false)

  const activePeer = $derived(peers.find(p => p.id === activePeerID) ?? null)
  const peerMessages = $derived(messages.filter(m => m.peerID === activePeerID))

  function handleSelectPeer(id: string) { activePeerID = id }
  function handleJoinRoom(code: string) { console.log('join room:', code) }
  function handleSend(content: string) { console.log('send:', content) }
</script>

<div class="flex h-screen w-screen overflow-hidden">
  <P2PPeerSidebar
    {peers}
    {activePeerID}
    {roomCode}
    {profile}
    onSelectPeer={handleSelectPeer}
    onJoinRoom={handleJoinRoom}
    onOpenSettings={() => showSettings = true}
  />
  <P2PChatArea
    peer={activePeer}
    messages={peerMessages}
    {sending}
    onSend={handleSend}
  />
</div>

<SettingsPanel
  bind:open={showSettings}
  currentUser={profile ? { username: profile.displayName, display_name: profile.displayName, avatar_url: profile.avatarDataUrl ?? '' } : null}
  onLogout={() => {}}
/>
