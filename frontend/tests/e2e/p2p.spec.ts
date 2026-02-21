import { test, expect } from '@playwright/test'

// Stubs dos bindings Wails para rodar fora do runtime desktop
const wailsStubs = `
  window['go'] = window['go'] || {}
  window['go']['main'] = window['go']['main'] || {}
  window['go']['main']['App'] = {
    InitP2PHost: () => Promise.resolve(null),
    GetP2PRoomCode: () => Promise.resolve('amber-4271'),
    GetP2PPeers: () => Promise.resolve([
      { id: 'peer-aabbccdd', addresses: ['/ip4/192.168.1.10/tcp/4001'], connected: true },
    ]),
    GetP2PPeerName: () => Promise.resolve('Alice'),
    SendP2PMessage: () => Promise.resolve(null),
    GetP2PMessages: () => Promise.resolve([]),
    SendP2PProfile: () => Promise.resolve(null),
    JoinP2PRoom: () => Promise.resolve(null),
    SelectAvatarFile: () => Promise.resolve(''),
    RestoreSession: () => Promise.resolve({ authenticated: false }),
    StartLogin: () => Promise.resolve({ device_code: '', user_code: '', verification_uri: '', expires_in: 0, interval: 0 }),
    CompleteLogin: () => Promise.resolve({ authenticated: false }),
    Logout: () => Promise.resolve(null),
    ListUserServers: () => Promise.resolve([]),
    GetServer: () => Promise.resolve(null),
    ListChannels: () => Promise.resolve([]),
    ListMembers: () => Promise.resolve([]),
    GetMessages: () => Promise.resolve([]),
    SendMessage: () => Promise.resolve(null),
    CreateServer: () => Promise.resolve(null),
    DeleteServer: () => Promise.resolve(null),
    CreateChannel: () => Promise.resolve(null),
    DeleteChannel: () => Promise.resolve(null),
    EditMessage: () => Promise.resolve(null),
    DeleteMessage: () => Promise.resolve(null),
    GenerateInvite: () => Promise.resolve(null),
    RedeemInvite: () => Promise.resolve(null),
    GetInviteInfo: () => Promise.resolve(null),
    UploadFile: () => Promise.resolve(null),
    DownloadFile: () => Promise.resolve(null),
    DeleteAttachment: () => Promise.resolve(null),
    GetAttachments: () => Promise.resolve([]),
    SearchMessages: () => Promise.resolve([]),
    Greet: () => Promise.resolve(''),
    GetVersion: () => Promise.resolve('test'),
    GetHealth: () => Promise.resolve({ status: 'ok' }),
    JoinVoice: () => Promise.resolve(null),
    LeaveVoice: () => Promise.resolve(null),
    ToggleMute: () => Promise.resolve(null),
    ToggleDeafen: () => Promise.resolve(null),
    GetVoiceStatus: () => Promise.resolve(null),
    EnableTranslation: () => Promise.resolve(null),
    DisableTranslation: () => Promise.resolve(null),
    GetTranslationStatus: () => Promise.resolve(null),
    UpdateServer: () => Promise.resolve(null),
    UpdateMemberRole: () => Promise.resolve(null),
    KickMember: () => Promise.resolve(null),
  }
  window['runtime'] = window['runtime'] || {}
  window['runtime']['EventsOnMultiple'] = window['runtime']['EventsOnMultiple'] || function() { return function() {} }
  window['runtime']['EventsOn'] = window['runtime']['EventsOn'] || function() { return function() {} }
  window['runtime']['EventsOff'] = window['runtime']['EventsOff'] || function() {}
  window['runtime']['EventsOffAll'] = window['runtime']['EventsOffAll'] || function() {}
  window['runtime']['EventsOnce'] = window['runtime']['EventsOnce'] || function() { return function() {} }
  window['runtime']['EventsEmit'] = window['runtime']['EventsEmit'] || function() {}
  window['runtime']['LogPrint'] = window['runtime']['LogPrint'] || function() {}
  window['runtime']['LogTrace'] = window['runtime']['LogTrace'] || function() {}
  window['runtime']['LogDebug'] = window['runtime']['LogDebug'] || function() {}
  window['runtime']['LogInfo'] = window['runtime']['LogInfo'] || function() {}
  window['runtime']['LogWarning'] = window['runtime']['LogWarning'] || function() {}
  window['runtime']['LogError'] = window['runtime']['LogError'] || function() {}
  window['runtime']['LogFatal'] = window['runtime']['LogFatal'] || function() {}
`

// Intercept /wails/runtime.js to avoid 404 blocking page load
async function mockWailsRuntime(page: import('@playwright/test').Page) {
  await page.route('**/wails/runtime.js', route => {
    route.fulfill({ status: 200, contentType: 'application/javascript', body: '// mocked wails runtime' })
  })
}

async function setupP2PApp(page: import('@playwright/test').Page) {
  await mockWailsRuntime(page)
  await page.addInitScript(wailsStubs)
  await page.addInitScript(() => {
    localStorage.setItem('concord-settings', JSON.stringify({
      networkMode: 'p2p',
      p2pProfile: { displayName: 'TestUser' },
    }))
  })
}

test.describe('P2P Onboarding', () => {
  test('exibe ModeSelector quando networkMode é null', async ({ page }) => {
    await mockWailsRuntime(page)
    await page.addInitScript(wailsStubs)
    await page.goto('/')
    await expect(page.getByText('Bem-vindo ao Concord')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('P2P')).toBeVisible()
    await expect(page.getByText('Servidor Oficial')).toBeVisible()
  })

  test('navega para P2PProfile ao clicar P2P', async ({ page }) => {
    await mockWailsRuntime(page)
    await page.addInitScript(wailsStubs)
    await page.addInitScript(() => {
      localStorage.setItem('concord-settings', JSON.stringify({ networkMode: null }))
    })
    await page.goto('/')
    await expect(page.getByText('Bem-vindo ao Concord')).toBeVisible({ timeout: 10000 })
    await page.getByText('P2P').first().click()
    await expect(page.getByText('Criar seu perfil P2P')).toBeVisible({ timeout: 5000 })
  })
})

test.describe('P2P App', () => {
  test('exibe room code na sidebar', async ({ page }) => {
    await setupP2PApp(page)
    await page.goto('/')
    await expect(page.getByText('amber-4271')).toBeVisible({ timeout: 15000 })
  })

  test('exibe empty state antes de selecionar peer', async ({ page }) => {
    await setupP2PApp(page)
    await page.goto('/')
    await expect(page.getByText('Selecione um peer para conversar')).toBeVisible({ timeout: 15000 })
  })

  test('exibe secao Sala na sidebar', async ({ page }) => {
    await setupP2PApp(page)
    await page.goto('/')
    await expect(page.getByText('Sala')).toBeVisible({ timeout: 15000 })
  })

  test('exibe peers na sidebar apos polling', async ({ page }) => {
    await setupP2PApp(page)
    await page.goto('/')
    // Store shows peer.id.slice(0,8) when no profile is cached
    await expect(page.getByText('peer-aab')).toBeVisible({ timeout: 15000 })
  })

  test('join room — preenche codigo e clica entrar', async ({ page }) => {
    await setupP2PApp(page)
    await page.goto('/')

    await expect(page.getByText('amber-4271')).toBeVisible({ timeout: 15000 })

    const joinInput = page.getByPlaceholder('Codigo da sala')
    await joinInput.fill('test-room-123')

    const joinButton = page.getByRole('button', { name: 'Entrar' })
    await expect(joinButton).toBeEnabled()
    await joinButton.click()

    await expect(joinInput).toHaveValue('')
  })
})
