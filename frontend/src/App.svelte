<script lang="ts">
  import ServerSidebar from './lib/components/layout/ServerSidebar.svelte'
  import ChannelSidebar from './lib/components/layout/ChannelSidebar.svelte'
  import MainContent from './lib/components/layout/MainContent.svelte'
  import MemberSidebar from './lib/components/layout/MemberSidebar.svelte'

  // Mock data
  const servers = [
    { id: '1', name: 'Gaming Squad', hasNotification: true },
    { id: '2', name: 'Dev Team' },
    { id: '3', name: 'Music' },
  ]

  const channels = [
    { id: 'general', name: 'general', type: 'text' as const, unreadCount: 3 },
    { id: 'gaming', name: 'gaming', type: 'text' as const },
    { id: 'memes', name: 'memes', type: 'text' as const },
    { id: 'voice-1', name: 'Voice Chat', type: 'voice' as const },
    { id: 'voice-2', name: 'AFK', type: 'voice' as const },
  ]

  const members = [
    { id: '1', name: 'Alice', status: 'online' as const, role: 'Admin' },
    { id: '2', name: 'Bob', status: 'online' as const },
    { id: '3', name: 'Eve', status: 'idle' as const },
    { id: '4', name: 'Charlie', status: 'dnd' as const, role: 'Moderator' },
    { id: '5', name: 'Dave', status: 'offline' as const },
    { id: '6', name: 'Frank', status: 'offline' as const },
  ]

  let activeServerId = $state('1')
  let activeChannelId = $state('general')

  const activeChannel = $derived(channels.find(c => c.id === activeChannelId))
</script>

<div class="flex h-screen w-screen overflow-hidden">
  <ServerSidebar
    {servers}
    {activeServerId}
    onSelectServer={(id) => activeServerId = id}
  />
  <ChannelSidebar
    serverName="Gaming Squad"
    {channels}
    {activeChannelId}
    onSelectChannel={(id) => activeChannelId = id}
  />
  <MainContent channelName={activeChannel?.name ?? 'general'} />
  <MemberSidebar {members} />
</div>
