# P2P Onboarding Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ao abrir o app pela primeira vez, o usuário escolhe entre modo P2P (sem conta, identidade local, descoberta mDNS+DHT) ou Servidor Oficial (login GitHub existente).

**Architecture:** A escolha de modo é salva em `settings` (`networkMode: 'p2p' | 'server'`). No boot, `App.svelte` lê o modo e roteia: null → ModeSelector → [P2PProfile ou Login GitHub] → app. No modo P2P, a sidebar mostra peers LAN (mDNS) e salas WAN (DHT com código curto).

**Tech Stack:** Svelte 5 runes, TailwindCSS v4, Wails v2, Go libp2p (já presente em `internal/network/p2p/`), zerolog, modernc SQLite.

---

### Task 1: Adicionar `networkMode` e `p2pProfile` ao settings store

**Files:**
- Modify: `frontend/src/lib/stores/settings.svelte.ts`

**Step 1: Adicionar tipos e estado**

Abrir `frontend/src/lib/stores/settings.svelte.ts` e adicionar ao state e ao tipo de settings:

```typescript
// Adicionar ao topo do arquivo (após os imports existentes):
export type NetworkMode = 'p2p' | 'server'

export interface P2PProfile {
  displayName: string
  avatarDataUrl?: string  // base64 data URL da imagem local
}
```

No objeto de estado (`$state`), adicionar:
```typescript
networkMode: null as NetworkMode | null,
p2pProfile: null as P2PProfile | null,
```

**Step 2: Atualizar `loadSettings` para ler os novos campos**

Na função `loadSettings()`, após carregar o objeto do localStorage, adicionar:
```typescript
if (data.networkMode) state.networkMode = data.networkMode
if (data.p2pProfile) state.p2pProfile = data.p2pProfile
```

**Step 3: Adicionar funções de escrita**

```typescript
export function setNetworkMode(mode: NetworkMode) {
  state.networkMode = mode
  saveSettings({ networkMode: mode })
}

export function setP2PProfile(profile: P2PProfile) {
  state.p2pProfile = profile
  saveSettings({ p2pProfile: profile })
}
```

**Step 4: Atualizar `getSettings()` para expor os novos campos**

```typescript
get networkMode() { return state.networkMode },
get p2pProfile() { return state.p2pProfile },
```

**Step 5: Verificar tipos (sem testes unitários para store simples)**

```bash
cd frontend && npx svelte-check --tsconfig ./tsconfig.app.json 2>&1 | grep '"src/' | head -20
```
Esperado: sem erros em src/.

**Step 6: Commit**

```bash
git add frontend/src/lib/stores/settings.svelte.ts
git commit -m "feat(settings): adiciona networkMode e p2pProfile ao store"
```

---

### Task 2: Expor `SelectAvatarFile` no backend Wails

**Files:**
- Modify: `app.go` (ou `main.go` onde estão os bindings Wails — verificar qual arquivo tem os métodos expostos ao frontend)

**Step 1: Verificar onde ficam os métodos Wails**

```bash
grep -rn "func.*App.*wails\|wails:.*func\|OpenFileDialog\|runtime.OpenFileDialog" . --include="*.go" | head -20
```

**Step 2: Adicionar método `SelectAvatarFile`**

No arquivo que contém os métodos expostos ao Wails (provavelmente `app.go`), adicionar:

```go
// SelectAvatarFile abre diálogo de seleção de arquivo de imagem e retorna
// o conteúdo como data URL base64 (para armazenar sem path absoluto).
// Complexity: O(n) onde n é o tamanho do arquivo de imagem.
func (a *App) SelectAvatarFile() (string, error) {
    path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title: "Escolha seu avatar",
        Filters: []runtime.FileFilter{
            {DisplayName: "Imagens", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp"},
        },
    })
    if err != nil || path == "" {
        return "", err
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("failed to read avatar file: %w", err)
    }

    // Detect MIME type and encode as data URL
    mime := http.DetectContentType(data)
    encoded := base64.StdEncoding.EncodeToString(data)
    return fmt.Sprintf("data:%s;base64,%s", mime, encoded), nil
}
```

Imports necessários: `"encoding/base64"`, `"net/http"`, `"os"`, `"fmt"`, `"github.com/wailsapp/wails/v2/pkg/runtime"`.

**Step 3: Build para verificar compilação**

```bash
go build ./...
```
Esperado: sem erros.

**Step 4: Regenerar bindings Wails**

```bash
wails generate module
```
Esperado: `frontend/wailsjs/go/main/App.js` e `App.d.ts` atualizados com `SelectAvatarFile`.

**Step 5: Commit**

```bash
git add app.go frontend/wailsjs/
git commit -m "feat(wails): expoe SelectAvatarFile para escolha de avatar local"
```

---

### Task 3: Criar `ModeSelector.svelte`

**Files:**
- Create: `frontend/src/lib/components/auth/ModeSelector.svelte`

**Step 1: Criar o componente**

```svelte
<script lang="ts">
  let {
    onSelectMode,
  }: {
    onSelectMode: (mode: 'p2p' | 'server') => void
  } = $props()
</script>

<div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
  <div class="w-full max-w-2xl space-y-8 px-6">
    <!-- Logo -->
    <div class="text-center">
      <!-- [mesmo SVG do logo que está em Login.svelte] -->
      <h1 class="mt-4 text-2xl font-bold text-void-text-primary">Bem-vindo ao Concord</h1>
      <p class="mt-2 text-sm text-void-text-muted">Como você quer se conectar?</p>
    </div>

    <!-- Cards de escolha -->
    <div class="grid grid-cols-2 gap-4">
      <!-- P2P Card -->
      <button
        onclick={() => onSelectMode('p2p')}
        class="group flex flex-col items-center gap-4 rounded-xl border border-void-border bg-void-bg-secondary p-8 text-left transition-all hover:border-void-accent hover:bg-void-bg-hover cursor-pointer"
      >
        <!-- ícone P2P (duas setas circulares) -->
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-void-bg-tertiary group-hover:bg-void-accent/10 transition-colors">
          <svg class="h-8 w-8 text-void-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M4 12v-1a8 8 0 0 1 8-8"/>
            <path d="M20 12v1a8 8 0 0 1-8 8"/>
            <polyline points="9 3 4 3 4 8"/>
            <polyline points="15 21 20 21 20 16"/>
          </svg>
        </div>
        <div class="text-center">
          <p class="text-base font-bold text-void-text-primary">P2P</p>
          <p class="mt-1 text-xs text-void-text-muted leading-relaxed">
            Sem servidor central.<br>
            Conecte-se direto com amigos<br>
            na LAN ou via código de sala.
          </p>
        </div>
        <div class="mt-auto flex flex-col gap-1 w-full">
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Sem conta necessária
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Descoberta automática na LAN
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            WAN via código de sala
          </span>
        </div>
      </button>

      <!-- Servidor Card -->
      <button
        onclick={() => onSelectMode('server')}
        class="group flex flex-col items-center gap-4 rounded-xl border border-void-border bg-void-bg-secondary p-8 text-left transition-all hover:border-void-accent hover:bg-void-bg-hover cursor-pointer"
      >
        <!-- ícone nuvem -->
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-void-bg-tertiary group-hover:bg-void-accent/10 transition-colors">
          <svg class="h-8 w-8 text-void-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
          </svg>
        </div>
        <div class="text-center">
          <p class="text-base font-bold text-void-text-primary">Servidor Oficial</p>
          <p class="mt-1 text-xs text-void-text-muted leading-relaxed">
            Infraestrutura centralizada.<br>
            Mensagens persistidas,<br>
            acesso de qualquer lugar.
          </p>
        </div>
        <div class="mt-auto flex flex-col gap-1 w-full">
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Login com GitHub
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Mensagens na nuvem
          </span>
          <span class="flex items-center gap-1.5 text-xs text-void-text-muted">
            <svg class="h-3 w-3 text-void-online shrink-0" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
            Servidores globais
          </span>
        </div>
      </button>
    </div>

    <p class="text-center text-xs text-void-text-muted">
      Esta escolha pode ser alterada nas configurações.
    </p>
  </div>
</div>
```

**Step 2: Verificar tipos**

```bash
cd frontend && npx svelte-check --tsconfig ./tsconfig.app.json 2>&1 | grep '"src/' | head -20
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/auth/ModeSelector.svelte
git commit -m "feat(ui): adiciona tela de selecao de modo P2P vs Servidor"
```

---

### Task 4: Criar `P2PProfile.svelte`

**Files:**
- Create: `frontend/src/lib/components/auth/P2PProfile.svelte`

**Step 1: Criar o componente**

```svelte
<script lang="ts">
  import { SelectAvatarFile } from '../../../wailsjs/go/main/App'
  import Button from '../ui/Button.svelte'

  let {
    onConfirm,
  }: {
    onConfirm: (profile: { displayName: string; avatarDataUrl?: string }) => void
  } = $props()

  let displayName = $state('')
  let avatarDataUrl = $state<string | undefined>(undefined)
  let error = $state('')
  let loadingAvatar = $state(false)

  async function handleSelectAvatar() {
    loadingAvatar = true
    try {
      const result = await SelectAvatarFile()
      if (result) avatarDataUrl = result
    } catch (e) {
      // usuário cancelou o diálogo — não é erro
    } finally {
      loadingAvatar = false
    }
  }

  function handleConfirm() {
    const name = displayName.trim()
    if (name.length < 2) { error = 'Nome deve ter pelo menos 2 caracteres'; return }
    if (name.length > 32) { error = 'Nome deve ter no máximo 32 caracteres'; return }
    error = ''
    onConfirm({ displayName: name, avatarDataUrl })
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleConfirm()
  }
</script>

<div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
  <div class="w-full max-w-sm space-y-6 px-6">
    <div class="text-center">
      <h1 class="text-xl font-bold text-void-text-primary">Criar seu perfil P2P</h1>
      <p class="mt-1 text-sm text-void-text-muted">Sua identidade é armazenada apenas localmente.</p>
    </div>

    <!-- Avatar picker -->
    <div class="flex flex-col items-center gap-3">
      <button
        onclick={handleSelectAvatar}
        disabled={loadingAvatar}
        class="relative h-20 w-20 rounded-full overflow-hidden border-2 border-dashed border-void-border hover:border-void-accent transition-colors cursor-pointer group"
      >
        {#if avatarDataUrl}
          <img src={avatarDataUrl} alt="Avatar" class="h-full w-full object-cover" />
          <div class="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity">
            <svg class="h-6 w-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 1 2-2v-7"/>
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
          </div>
        {:else}
          <div class="flex h-full w-full flex-col items-center justify-center gap-1 bg-void-bg-secondary">
            {#if loadingAvatar}
              <div class="h-5 w-5 animate-spin rounded-full border-2 border-void-accent border-t-transparent"></div>
            {:else}
              <svg class="h-7 w-7 text-void-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
              <span class="text-[10px] text-void-text-muted">Foto</span>
            {/if}
          </div>
        {/if}
      </button>
      <p class="text-xs text-void-text-muted">Clique para escolher uma foto</p>
    </div>

    <!-- Name input -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div onkeydown={handleKeydown}>
      <label class="mb-1.5 block text-sm font-medium text-void-text-secondary">
        Nome de exibição
      </label>
      <input
        type="text"
        bind:value={displayName}
        placeholder="Seu nome..."
        maxlength="32"
        class="w-full rounded-md border border-void-border bg-void-bg-secondary px-3 py-2 text-sm text-void-text-primary placeholder:text-void-text-muted focus:border-void-accent focus:outline-none focus:ring-2 focus:ring-void-accent
          {error ? 'border-void-danger focus:ring-void-danger' : ''}"
      />
      {#if error}
        <p class="mt-1 text-xs text-void-danger">{error}</p>
      {/if}
    </div>

    <Button variant="solid" size="lg" onclick={handleConfirm} disabled={!displayName.trim()}>
      Entrar no Concord P2P
    </Button>
  </div>
</div>
```

**Step 2: Verificar tipos**

```bash
cd frontend && npx svelte-check --tsconfig ./tsconfig.app.json 2>&1 | grep '"src/' | head -20
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/auth/P2PProfile.svelte
git commit -m "feat(ui): adiciona tela de perfil local para modo P2P"
```

---

### Task 5: Integrar roteamento de boot no `App.svelte`

**Files:**
- Modify: `frontend/src/App.svelte`

**Step 1: Importar novos componentes e funções**

Adicionar aos imports existentes:
```typescript
import ModeSelector from './lib/components/auth/ModeSelector.svelte'
import P2PProfile from './lib/components/auth/P2PProfile.svelte'
import {
  getSettings, loadSettings, setNetworkMode, setP2PProfile
} from './lib/stores/settings.svelte'
```

**Step 2: Adicionar estado derivado de modo**

```typescript
const settings = getSettings()

// derivados de roteamento
const networkMode = $derived(settings.networkMode)
const p2pProfile = $derived(settings.p2pProfile)
const needsModeSelection = $derived(networkMode === null)
const needsP2PProfile = $derived(networkMode === 'p2p' && !p2pProfile)
const isP2PMode = $derived(networkMode === 'p2p' && !!p2pProfile)
const isServerMode = $derived(networkMode === 'server')
```

**Step 3: Atualizar handlers**

```typescript
function handleModeSelect(mode: 'p2p' | 'server') {
  setNetworkMode(mode)
}

function handleP2PProfileConfirm(profile: { displayName: string; avatarDataUrl?: string }) {
  setP2PProfile(profile)
}
```

**Step 4: Atualizar template — adicionar roteamento antes do bloco `{#if auth.loading}`**

Substituir o bloco principal por:

```svelte
{#if needsModeSelection}
  <ModeSelector onSelectMode={handleModeSelect} />

{:else if needsP2PProfile}
  <P2PProfile onConfirm={handleP2PProfileConfirm} />

{:else if isP2PMode}
  <!-- Placeholder: app P2P (Task 6) -->
  <div class="flex h-screen w-screen items-center justify-center bg-void-bg-primary">
    <p class="text-void-text-muted">Modo P2P — em breve</p>
  </div>

{:else if auth.loading}
  <!-- loading splash existente -->

{:else if !auth.authenticated}
  <Login />

{:else}
  <!-- layout server existente (todo o bloco atual) -->
{/if}
```

**Step 5: Verificar tipos e build**

```bash
cd frontend && npx svelte-check --tsconfig ./tsconfig.app.json 2>&1 | grep '"src/' | head -20
go build ./...
```

**Step 6: Commit**

```bash
git add frontend/src/App.svelte frontend/src/lib/stores/settings.svelte.ts
git commit -m "feat(app): roteia boot para ModeSelector, P2PProfile ou fluxo servidor"
```

---

### Task 6: Backend P2P — discovery mDNS + sala DHT

**Files:**
- Modify: `internal/network/p2p/host.go` (ou o arquivo principal do p2p existente — verificar com `ls internal/network/p2p/`)
- Create: `internal/network/p2p/discovery.go`
- Create: `internal/network/p2p/room.go`

**Step 1: Verificar estrutura atual do p2p**

```bash
ls internal/network/p2p/
cat internal/network/p2p/*.go | head -80
```

**Step 2: Criar `discovery.go` — mDNS announce + discover**

```go
package p2p

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
    "github.com/rs/zerolog"
)

const mdnsServiceTag = "_concord._tcp"

// PeerInfo contém dados de um peer descoberto.
type PeerInfo struct {
    ID          string `json:"id"`
    DisplayName string `json:"display_name"`
    AvatarHash  string `json:"avatar_hash"`
}

// discoveryNotifee recebe notificações do mDNS.
type discoveryNotifee struct {
    h      host.Host
    logger zerolog.Logger
    found  chan peer.AddrInfo
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
    n.logger.Info().Str("peer", pi.ID.String()).Msg("mDNS peer discovered")
    n.found <- pi
}

// StartMDNS inicia o serviço de descoberta mDNS e retorna canal de peers encontrados.
// Complexity: O(1) setup, O(n) peers na rede local.
func StartMDNS(ctx context.Context, h host.Host, logger zerolog.Logger) (<-chan peer.AddrInfo, error) {
    found := make(chan peer.AddrInfo, 32)
    notifee := &discoveryNotifee{h: h, logger: logger, found: found}

    svc := mdns.NewMdnsService(h, mdnsServiceTag, notifee)
    if err := svc.Start(); err != nil {
        return nil, fmt.Errorf("mdns start: %w", err)
    }

    go func() {
        <-ctx.Done()
        svc.Close()
        close(found)
    }()

    logger.Info().Msg("mDNS discovery started")
    return found, nil
}
```

**Step 3: Criar `room.go` — sala DHT com código curto**

```go
package p2p

import (
    "context"
    "crypto/sha256"
    "fmt"
    "math/big"

    dht "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
    "github.com/rs/zerolog"
)

// wordlist simples para gerar códigos legíveis (subset de 1024 palavras)
var roomWordlist = []string{
    "alpha", "bravo", "cobra", "delta", "echo", "foxtrot", "golf", "hotel",
    "india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
    // ... em produção usar lista maior embeddada
}

// RoomCode gera um código curto legível a partir do peerId do host.
// Complexity: O(1).
func RoomCode(h host.Host) string {
    id := h.ID().String()
    hash := sha256.Sum256([]byte(id))
    n := new(big.Int).SetBytes(hash[:4])
    n1 := new(big.Int).Mod(n, big.NewInt(int64(len(roomWordlist)))).Int64()
    n2 := new(big.Int).Mod(new(big.Int).Rsh(n, 10), big.NewInt(9000)).Int64() + 1000
    return fmt.Sprintf("%s-%d", roomWordlist[n1], n2)
}

// JoinRoom conecta o host a uma sala DHT identificada pelo código.
// Complexity: O(log n) DHT lookup.
func JoinRoom(ctx context.Context, h host.Host, kadDHT *dht.IpfsDHT, code string, logger zerolog.Logger) (<-chan peer.AddrInfo, error) {
    rd := drouting.NewRoutingDiscovery(kadDHT)

    // Announce presença nesta sala
    _, err := rd.Advertise(ctx, "concord-room/"+code)
    if err != nil {
        return nil, fmt.Errorf("room advertise: %w", err)
    }

    // Descobrir outros peers na sala
    peerChan, err := rd.FindPeers(ctx, "concord-room/"+code)
    if err != nil {
        return nil, fmt.Errorf("room find peers: %w", err)
    }

    logger.Info().Str("room", code).Msg("joined P2P room")
    return peerChan, nil
}
```

**Step 4: Build para verificar compilação**

```bash
go build ./internal/network/p2p/...
```

Se faltarem dependências no go.mod:
```bash
go get github.com/libp2p/go-libp2p-kad-dht
go mod tidy
```

**Step 5: Escrever teste unitário para `RoomCode`**

Em `internal/network/p2p/room_test.go`:
```go
package p2p_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/libp2p/go-libp2p"
    "github.com/concord-chat/concord/internal/network/p2p"
)

func TestRoomCode_Deterministic(t *testing.T) {
    h1, err := libp2p.New()
    require.NoError(t, err)
    defer h1.Close()

    code1 := p2p.RoomCode(h1)
    code2 := p2p.RoomCode(h1)
    assert.Equal(t, code1, code2, "mesmo host deve gerar mesmo código")
    assert.Contains(t, code1, "-", "código deve ter separador")
}
```

**Step 6: Rodar teste**

```bash
go test -short ./internal/network/p2p/... -run TestRoomCode
```

**Step 7: Commit**

```bash
git add internal/network/p2p/discovery.go internal/network/p2p/room.go internal/network/p2p/room_test.go
git commit -m "feat(p2p): descoberta mDNS e salas DHT com codigo curto"
```

---

### Task 7: Expor P2P discovery ao frontend via Wails

**Files:**
- Modify: `app.go` (métodos Wails)

**Step 1: Adicionar métodos P2P ao App**

```go
// GetP2PRoomCode retorna o código de sala do peer local.
func (a *App) GetP2PRoomCode() string {
    if a.p2pHost == nil {
        return ""
    }
    return p2p.RoomCode(a.p2pHost)
}

// JoinP2PRoom entra numa sala pelo código.
func (a *App) JoinP2PRoom(code string) error {
    // implementação conecta ao DHT room
    return nil // stub — completo na integração futura
}

// GetP2PPeers retorna lista de peers descobertos (LAN + sala).
func (a *App) GetP2PPeers() []p2p.PeerInfo {
    return a.p2pPeers // campo mantido pelo loop de descoberta
}
```

**Step 2: Build + regenerar bindings**

```bash
go build ./... && wails generate module
```

**Step 3: Commit**

```bash
git add app.go frontend/wailsjs/
git commit -m "feat(wails): expoe GetP2PRoomCode, JoinP2PRoom e GetP2PPeers"
```

---

### Task 8: Verificação final e cleanup

**Step 1: Rodar todos os testes**

```bash
go test -short ./... 2>&1 | tail -25
```
Esperado: todos `ok`.

**Step 2: Verificar tipos frontend**

```bash
cd frontend && npx svelte-check --tsconfig ./tsconfig.app.json 2>&1 | grep '"src/' | head -20
```
Esperado: sem erros.

**Step 3: Build completo**

```bash
go build ./...
```

**Step 4: Commit final**

```bash
git add -A
git commit -m "feat(p2p-onboarding): selecao de modo P2P vs Servidor no primeiro boot"
```
