# P2P Full Mode — Plano de Implementação

> Data: 2026-02-21
> Objetivo: Substituir o placeholder "Modo P2P — em breve" pelo modo P2P completo com layout espelho do Discord (peer sidebar + chat direto + persistência SQLite + handshake de perfil).

---

## Arquitetura

```
App.svelte
  └─ isP2PMode → <P2PApp>
       ├─ P2PPeerSidebar (240px) — perfil local, room code, lista de peers LAN/WAN
       └─ P2PChatArea (flex-1)  — histórico + input de chat com peer selecionado
```

### Protocolo de mensagens P2P

Envelope JSON trafegado pelo stream libp2p (`/concord/1.0.0`):
```json
{ "type": "profile"|"chat", "sender_id": "<peerID>", "payload": "<base64 json>" }
```
- `profile` → `{display_name, avatar_data_url?}` — enviado no connect e quando perfil muda
- `chat` → `{content, sent_at}` — mensagem de texto

### SQLite — migration 007

```sql
CREATE TABLE IF NOT EXISTS p2p_messages (
  id        TEXT PRIMARY KEY,
  peer_id   TEXT NOT NULL,
  direction TEXT NOT NULL CHECK(direction IN ('sent','received')),
  content   TEXT NOT NULL,
  sent_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_p2p_messages_peer ON p2p_messages(peer_id, sent_at);
```

---

## Tasks

### Task A — Backend Go (sequencial, 1 agente)

**A1. `internal/network/p2p/protocol.go`**
Tipos: `MessageType`, `Envelope`, `ProfilePayload`, `ChatPayload`. Funções: `EncodeEnvelope`, `DecodeEnvelope`.

**A2. `internal/store/sqlite/migrations/007_p2p_messages.sql`**
Migration conforme schema acima.

**A3. `internal/store/sqlite/p2p_repository.go`**
```go
type P2PRepository interface {
  SaveMessage(ctx, msg P2PMessage) error
  GetMessages(ctx, peerID string, limit int) ([]P2PMessage, error)
}
type P2PMessage struct {
  ID        string `json:"id"`
  PeerID    string `json:"peer_id"`
  Direction string `json:"direction"`  // "sent"|"received"
  Content   string `json:"content"`
  SentAt    string `json:"sent_at"`
}
```
Implementar `sqliteP2PRepo` com `INSERT` e `SELECT ... ORDER BY sent_at DESC LIMIT ?`.

**A4. `main.go` — novos campos e métodos Wails**

Campos no App struct:
```go
p2pRepo   *sqlite.P2PRepo  // (ou interface)
```

Métodos a adicionar:
```go
// InitP2PHost inicia o host libp2p lazy (só chamado no modo P2P).
// Registra stream handler para processar envelopes recebidos.
func (a *App) InitP2PHost() error

// SendP2PMessage envia mensagem de chat para um peer.
// Persiste com direction="sent" antes de enviar.
func (a *App) SendP2PMessage(peerID, content string) error

// GetP2PMessages retorna histórico de mensagens com um peer (mais recentes primeiro).
func (a *App) GetP2PMessages(peerID string, limit int) ([]P2PMessage, error)
```

`InitP2PHost`:
1. Cria `p2p.New(p2p.DefaultConfig(), a.logger)` se `a.p2pHost == nil`
2. Registra `OnMessage` handler que decodifica `Envelope`:
   - `TypeProfile` → atualiza peerstore em memória (map `peerID → ProfilePayload`)
   - `TypeChat` → persiste `P2PMessage{direction:"received"}` + emite evento Wails `runtime.EventsEmit(a.ctx, "p2p:message", msg)`
3. Envia handshake de perfil para todos os peers conectados

`SendP2PMessage`:
1. Gera UUID para o ID
2. Persiste `P2PMessage{direction:"sent"}`
3. Serializa `Envelope{TypeChat, peerID, payload}` e chama `a.p2pHost.SendData`

**A5. Regenerar bindings Wails**
```bash
wails generate module
```

**A6. Testes Go**
- `p2p_repository_test.go` — SaveMessage + GetMessages com SQLite in-memory
- `protocol_test.go` — EncodeEnvelope/DecodeEnvelope round-trip

---

### Task B — Frontend Store (após A5)

**`frontend/src/lib/stores/p2p.svelte.ts`**

```typescript
export interface P2PPeer {
  id: string
  displayName: string
  avatarDataUrl?: string
  connected: boolean
  source: 'lan' | 'room'
}
export interface P2PMessage {
  id: string
  peerID: string
  direction: 'sent' | 'received'
  content: string
  sentAt: string
}

// Estado
let peers = $state<P2PPeer[]>([])
let activePeerID = $state<string | null>(null)
let messages = $state<Record<string, P2PMessage[]>>({})
let roomCode = $state('')
let joining = $state(false)
let initialized = $state(false)

// Inicialização
export async function initP2PStore() — chama InitP2PHost(), GetP2PRoomCode()
// Polling peers a cada 3s
export function startPeerPolling() — setInterval GetP2PPeers(), merge com displayNames conhecidos
// Escutar eventos de mensagem
export function listenMessages() — EventsOn('p2p:message', handler)
// Ações
export function setActivePeer(id: string | null)
export async function sendMessage(content: string)
export async function joinRoom(code: string)
export async function loadMessages(peerID: string)
export function getP2P() — retorna reactive getters
```

Fallback: se `InitP2PHost` falhar (fora do Wails runtime), silencia o erro e usa dados mock para dev/E2E.

---

### Task C — Componentes Frontend (paralelo com B)

**`frontend/src/lib/components/p2p/P2PPeerSidebar.svelte`**

Props: `peers`, `activePeerID`, `roomCode`, `profile`, `onSelectPeer`, `onJoinRoom`, `onOpenSettings`

Layout (240px, `bg-void-bg-secondary`):
- Topo: avatar + displayName do perfil local
- Card "Sala": room code em fonte mono com botão copy, input + botão "Entrar"
- Separadores: "NA REDE LOCAL" e "NA SALA" (se tiver peers WAN)
- Linha de peer: avatar (inicial se sem foto) + displayName + dot de status
- User panel no rodapé (igual ao ChannelSidebar)

**`frontend/src/lib/components/p2p/P2PChatArea.svelte`**

Props: `peer`, `messages`, `sending`, `onSend`

- Empty state se `!peer`: ícone + "Selecione um peer para conversar"
- Header: avatar + displayName do peer + badge "LAN" ou "WAN"
- Lista de mensagens (igual ao MainContent mas sem canais/servidores)
- Input de mensagem no rodapé

**`frontend/src/lib/components/p2p/P2PApp.svelte`**

Monta store P2P no `$effect`, combina sidebar + chat:
```svelte
<div class="flex h-screen w-screen overflow-hidden">
  <P2PPeerSidebar ... />
  <P2PChatArea ... />
</div>
```

---

### Task D — Integração App.svelte

Substituir o bloco `{:else if isP2PMode}` placeholder:
```svelte
{:else if isP2PMode}
  <P2PApp profile={settings.p2pProfile} />
```

---

### Task E — E2E Tests

**`frontend/tests/e2e/p2p.spec.ts`**

Injeta stubs via `addInitScript` antes de cada teste:
```typescript
window.__wails_mock = {
  InitP2PHost: () => Promise.resolve(null),
  GetP2PRoomCode: () => Promise.resolve('amber-4271'),
  GetP2PPeers: () => Promise.resolve([
    { id: 'peer-1', addresses: [], connected: true },
    { id: 'peer-2', addresses: [], connected: true },
  ]),
  SendP2PMessage: () => Promise.resolve(null),
  GetP2PMessages: () => Promise.resolve([]),
}
```

Testes:
1. **Mode selection** — localStorage vazio → vê ModeSelector → clica P2P → vê P2PProfile
2. **Profile creation** — preenche nome → clica confirmar → vê P2PApp
3. **Peer list renders** — P2PApp carrega → peers mockados aparecem na sidebar
4. **Room code visible** — room code "amber-4271" visível na sidebar
5. **Join room** — preenche código → clica entrar → botão fica loading
6. **Send message** — seleciona peer → digita mensagem → envia → aparece na lista

---

## Ordem de execução

```
A (backend) ──────────────────────────────────────► A5 (bindings)
                                                          │
B (store) ──────────────────────────────────────────────►│
C (componentes) ─────────────────────────────────────────┤
                                                          ▼
                                                    D (integração)
                                                          │
                                                          ▼
                                                    E (E2E tests)
```

A e C podem rodar em paralelo (C usa tipos mockados). B depende de A5.
