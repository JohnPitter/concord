# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.14.0] - 2026-02-22

### Added

#### Friends System Backend + Server Join Fix (2026-02-22)

- **Friends migration** (`008_friends.sql`): `friend_requests` table (sender/receiver/status with pending/accepted/rejected/blocked states, unique pair constraint) and `friends` table (bidirectional relationship with cascading deletes)
- **Friends repository** (`internal/friends/repository.go`): full CRUD — `SendRequest`, `GetPendingRequests` (incoming+outgoing with JOIN users), `AcceptRequest` (transactional: update status + insert bidirectional friendship), `RejectRequest`, `GetFriends`, `RemoveFriend`, `BlockUser` (removes friendship + upserts blocked status), `UnblockUser`, `ExistingRequest`, `AreFriends`
- **Friends service** (`internal/friends/service.go`): validation layer — prevents self-add, duplicate requests, sending to blocked users; username lookup via repository
- **Wails bindings** (`main.go`): `SendFriendRequest`, `GetPendingRequests`, `AcceptFriendRequest`, `RejectFriendRequest`, `GetFriends`, `RemoveFriend`, `BlockUser`, `UnblockUser` exposed to frontend
- **REST API endpoints** (`internal/api/handlers_friends.go`, `server.go`): `POST /friends/request`, `GET /friends/requests`, `PUT /friends/requests/{id}/accept`, `DELETE /friends/requests/{id}`, `GET /friends`, `DELETE /friends/{id}`, `POST /friends/{id}/block`, `DELETE /friends/{id}/block`
- **Frontend API client** (`frontend/src/lib/api/friends.ts`): typed wrappers for all friend REST endpoints

### Changed

- **Friends store refactored** (`frontend/src/lib/stores/friends.svelte.ts`): all mutations now call backend (Wails bindings in P2P mode, HTTP API in server mode); 30s polling for incoming friend requests; localStorage retained as offline cache; DM conversations auto-synced from friends list
- **Server Join fix** (`CreateServer.svelte`): `handleJoin()` now properly awaits the async `onJoin` callback; shows loading spinner during join; displays error messages from backend in the modal instead of silently failing
- **Server Join error propagation** (`App.svelte`): `handleJoinServer` returns `string | null` (error message or null on success) instead of `void`; catches and surfaces errors from `redeemInvite`

## [0.13.0] - 2026-02-22

### Added

#### UX Improvements Round 7 — DMs, Welcome, Friends, Voice (2026-02-22)

- **DM messaging funcional** (`friends.svelte.ts`, `App.svelte`): mensagens diretas entre amigos com persistência em localStorage, tipo `DMMessage`, lista de mensagens com bolhas estilo chat (enviadas=verde, recebidas=escuro), integração com `MessageInput` (emoji picker, file attach, keyboard handling)
- **Modal de boas-vindas (onboarding)** (`WelcomeModal.svelte`, `settings.svelte.ts`): 4 etapas em slide — Bem-vindo, Servidores & Canais, Amigos & DMs, Pronto!; indicador de steps (dots), botões Skip/Next/Finish, click-outside-to-close, backdrop blur; `hasSeenWelcome` persistido em settings
- **Logo oficial no Home button** (`ServerSidebar.svelte`): substituído SVG inline simplificado pela logo oficial `logo.png` (pomba completa)
- **Botão "Nova DM" funcional** (`DMSidebar.svelte`, `App.svelte`): botão `+` no sidebar de DMs agora navega para a view de Amigos
- **Menu de opções do amigo** (`FriendsList.svelte`): dropdown context menu nos três-pontos com "Enviar mensagem", "Remover amigo" e "Bloquear"; click-outside fecha
- **Som ao entrar em voice channel** (`voice.svelte.ts`): `playJoinSound()` chamado ao entrar no voice (incluindo o próprio usuário), não apenas quando outros entram
- **Sanitização de @username** (`friends.svelte.ts`): `sendFriendRequest()` remove `@` do início do username automaticamente
- **i18n**: adicionadas keys `dm.placeholder`, `welcome.*` (11 keys), `friends.removeFriend`, `friends.blockFriend` em todos os 5 idiomas (PT, EN, ES, ZH, JA)

#### Dynamic Server URL Discovery (2026-02-22)

- **Server URL via GitHub Gist** (`frontend/src/lib/api/client.ts`): client descobre URL do servidor central automaticamente via GitHub Gist público, com fallback para URL hardcoded; cache em localStorage com TTL de 1h

### Changed

- **Dependências atualizadas**: `pion/webrtc/v4` v4.2.8→v4.2.9, `ipfs/boxo` v0.36.0→v0.37.0, `ipld-prime` v0.21.0→v0.22.0, `multiaddr-dns` v0.4.1→v0.5.0, e diversas `golang.org/x/*` atualizadas

### Security

- **govulncheck**: GO-2026-4479 (`pion/dtls/v2`) e GO-2024-3218 (`go-libp2p-kad-dht`) são vulnerabilidades conhecidas sem fix upstream (Fixed in: N/A). Ambas são issues do ecossistema libp2p/pion que afetam todos os usuários. O código DTLS v2 vulnerável não é utilizado nos code paths do Concord (usamos TCP+QUIC+Noise, não DTLS)

### Added

#### HTTP API Client — Phase 9 Desktop→Server (2026-02-22)

- **API client base** (`frontend/src/lib/api/client.ts`): HTTP client singleton with JWT Bearer auth, automatic token refresh (2min pre-expiry buffer), localStorage token persistence, typed `get`/`post`/`put`/`del` methods, `publicRequest` for unauthenticated endpoints
- **API service modules** (`frontend/src/lib/api/auth.ts`, `servers.ts`, `chat.ts`): typed wrappers for all REST API endpoints — device-code auth, token exchange, session refresh, servers CRUD, channels, members, invites, messages (pagination, search, edit, delete)
- **Dual-mode stores**: `auth.svelte.ts`, `servers.svelte.ts`, `chat.svelte.ts` refactored with `isServerMode()` conditional — when `networkMode === 'server'`, calls go to central server REST API via HTTP; in P2P mode, continues using Wails bindings as before
- **Server URL setting** (`settings.svelte.ts`, `SettingsPanel.svelte`): `serverURL` field persisted to localStorage, configurable input in Settings > Account (visible only in server mode), syncs to `apiClient.setBaseURL()` on change
- **App initialization** (`App.svelte`): `apiClient.setBaseURL(serverURL)` called on mount when in server mode, before auth init
- **i18n**: added `settings.serverURL`, `settings.serverURLDesc`, `settings.serverURLPlaceholder` keys in all 5 locales (PT, EN, ES, ZH, JA)
- **Barrel export** (`frontend/src/lib/api/index.ts`): re-exports client, auth, servers, chat, mode helper

#### Real-time Voice Translation — Phase 8.2 (2026-02-22)

- **STT client** (`internal/voice/stt.go`): HTTP client for OpenAI Whisper-compatible APIs, multipart/form-data upload, configurable model/language/timeout
- **TTS client** (`internal/voice/tts.go`): HTTP client for OpenAI TTS-compatible APIs, JSON request with model/voice/format selection
- **VoiceTranslator pipeline** (`internal/voice/translator.go`): orchestrates Opus frame accumulation (2-3s segments) → OGG packing (pion oggwriter) → STT (Whisper) → text translation (LibreTranslate) → TTS → Wails event emission
- **OGG container packing**: Opus frames wrapped into OGG via `pion/webrtc/v4/pkg/media/oggwriter` with proper RTP headers — avoids CGo Opus decoding
- **Engine integration** (`internal/voice/engine.go`): `handleRemoteTrack` now feeds Opus payloads to VoiceTranslator when enabled; `SetTranslator()` method added
- **Config** (`internal/config/`): `VoiceTranslationConfig` struct with STT/TTS URLs, API keys, voice, format, segment length, timeout; env vars `WHISPER_URL`, `WHISPER_API_KEY`, `WHISPER_MODEL`, `TTS_URL`, `TTS_API_KEY`, `TTS_VOICE`, `TTS_FORMAT`
- **Wails bindings** (`main.go`): `EnableVoiceTranslation`, `DisableVoiceTranslation`, `GetVoiceTranslationStatus`; VoiceTranslator initialized in startup, cleaned up in shutdown
- **Frontend audio playback** (`voice.svelte.ts`): Wails event listener `voice:translated-audio` decodes base64 MP3 and plays via Web Audio API `decodeAudioData`/`BufferSource`; setup on join, teardown on leave
- **TranslationToggle updated** (`TranslationToggle.svelte`): calls `EnableVoiceTranslation`/`DisableVoiceTranslation` instead of text-only translation bindings
- **13 unit tests**: STT client (3), TTS client (3), VoiceTranslator (7 — enable/disable, language validation, OGG packing, full pipeline with mock servers, accumulator flush, disabled push safety)

### Changed

#### LibreTranslate Integration — Phase 8 Refactor (2026-02-22)

- **Translation backend replaced**: PersonaPlex (NVIDIA) → LibreTranslate (open-source, self-hosted)
- **Client refactored** (`internal/translation/client.go`): request/response structs aligned to LibreTranslate API (`q`/`source`/`target`/`translatedText`), removed WebSocket streaming, removed Bearer auth (uses `api_key` in body), circuit breaker and latency tracking preserved
- **Streaming pipeline removed** (`internal/translation/stream.go`): deleted — LibreTranslate is HTTP-only, no audio streaming
- **Service simplified** (`internal/translation/service.go`): removed `StartPipeline`/`StopPipeline`/`PipelineActive`, kept `TranslateText` with cache + circuit breaker
- **Config updated** (`internal/config/`): `PersonaPlexURL` → `URL` (default `http://localhost:5000`), env vars `LIBRETRANSLATE_URL`/`LIBRETRANSLATE_API_KEY` replace `PERSONAPLEX_*`
- **New Wails binding** (`main.go`): `TranslateText(text, sourceLang, targetLang)` exposes backend translation to frontend
- **Frontend unified** (`MessageBubble.svelte`): translation now goes through Go backend (cache + circuit breaker) instead of direct MyMemory API call
- **Tests updated**: 14 tests covering client, circuit breaker, cache, service, and concurrency — pipeline tests removed
- **Auto-translate**: new `autoTranslate` setting in Settings > Language — when enabled, incoming messages from other users are automatically translated without clicking the translate button
- **TranslateTextDirect**: new service method that bypasses the Enable/Disable gate, allowing on-demand translation regardless of service state
- **i18n**: added `settings.autoTranslate` and `settings.autoTranslateDesc` keys in all 5 locales (PT, EN, ES, ZH, JA)

### Added

#### Internationalization (i18n) — 5 Languages (2026-02-22)

- **i18n core infrastructure** (`frontend/src/lib/i18n/`): lightweight i18n system using Svelte 5 `writable`/`derived` stores with `t()` function, dynamic locale loading with in-memory cache, locale detection chain (localStorage > navigator.language > default 'pt')
- **5 locale files** (`locales/pt.json`, `en.json`, `es.json`, `zh.json`, `ja.json`): ~260 translation keys covering all UI components — Portuguese (default), English, Spanish, Chinese Simplified, Japanese
- **All 22 components migrated**: Login, ModeSelector, P2PProfile, ServerSidebar, ChannelSidebar, MainContent, DMSidebar, FriendsList, ActiveNow, NoServers, MemberSidebar, MessageInput, MessageList, MessageBubble, FileAttachment, CreateServer, JoinServer, ServerInfoModal, VoiceControls, TranslationToggle, SettingsPanel, P2PChatArea, P2PPeerSidebar, App.svelte
- **UI language selector** in Settings > Language: flag-labeled buttons for all 5 locales, persisted to localStorage, instant switching without reload
- **Portuguese translations** fully localized: all previously English-only strings in PT locale now properly translated (auth, chat, channel, settings, voice, etc.)

#### Scaling Phase 1 — Production Readiness (2026-02-21)

- **Prometheus HTTP metrics wired** (`api/middleware.go`, `api/server.go`): `MetricsMiddleware` coleta `concord_http_requests_total`, `concord_http_request_duration_milliseconds` e `concord_http_response_size_bytes` com path normalization para prevenir cardinality explosion
- **Rate limiting com headers** (`api/middleware.go`): `RateLimitWithHeaders` adiciona `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` e `Retry-After` nas respostas; RPS configuravel via `ServerConfig.RateLimitRPS`
- **CSP corrigido para WebSocket** (`api/middleware.go`): Content-Security-Policy agora inclui `connect-src 'self' ws: wss:` para permitir conexoes WebSocket
- **Docker Compose producao** (`deployments/docker/docker-compose.prod.yml`): Nginx reverse proxy com gzip, WebSocket upgrade, resource limits (CPU/RAM) para todos os servicos, PostgreSQL tuning otimizado (shared_buffers=1GB, effective_cache_size=3GB, max_connections=200)
- **Nginx config** (`deployments/docker/nginx.conf`): reverse proxy com gzip compression, WebSocket support, keepalive connections, security headers
- **Graceful shutdown melhorado** (`cmd/server/main.go`): drain ordenado — HTTP requests primeiro, depois Redis, depois PostgreSQL, com logging detalhado em cada etapa
- **Plano de escala 100k usuarios** (`docs/SCALING.md`): documentacao completa com 4 fases (0-5k, 5k-20k, 20k-50k, 50k-100k), sizing de VPS, topologias e prioridades de implementacao

#### UX Improvements Round 6 (2026-02-21)

- **Mode switch cleanup** (`App.svelte`): ao trocar de modo (P2P/Server) ou fazer logout, o app sai do voice channel, reseta chat, reseta voice, fecha modais e limpa navegacao antes de redirecionar ao ModeSelector
- **Badge "AO VIVO"** (`ChannelSidebar.svelte`, `voice.svelte.ts`): badge vermelho pulsante "AO VIVO" aparece ao lado do username de quem esta compartilhando tela no voice channel; `SpeakerData` agora inclui campo `screenSharing`
- **Traducao inline de mensagens** (`MessageBubble.svelte`, `settings.svelte.ts`): botao de traducao (icone globe) aparece ao hover em cada mensagem; usa API MyMemory (gratuita, sem API key) para traduzir entre 11 idiomas configurados em Settings > Language; resultado exibido abaixo da mensagem com borda verde
- **Limpeza de arquivos do git** (`.gitignore`): removidos `*.exe`, `coverage.out`, `frontend/dist/`, `frontend/package.json.md5` do tracking; adicionados ao `.gitignore`
- **Logo mascote Concord** (`frontend/src/assets/logo.svg`, `logo-full.svg`): novo logo com mascote fofo (criatura tipo raposa com headphones e coracao, dentro de balao de chat verde); aplicado em Login, ModeSelector, NoServers e favicon
- **GitHub Pages landing page** (`docs/site/index.html`): pagina de divulgacao com hero animado, secoes de features, modos P2P/Servidor, download multi-plataforma; design sofisticado com micro-interacoes, glow effects e tipografia hierarquica
- **GitHub Pages deploy workflow** (`.github/workflows/pages.yml`): deploy automatico ao push em `docs/site/` na branch main

#### UX Improvements Round 5 (2026-02-21)

- **Supressao de ruido e compartilhamento de tela** (`VoiceControls.svelte`, `voice.svelte.ts`): botoes de toggle para noise suppression (ativado por padrao, icone de ondas) e screen share (icone de monitor) no painel de controles de voz, com separador visual entre controles extras e mute/deafen
- **Avatar do GitHub no voice channel** (`ChannelSidebar.svelte`, `voice.svelte.ts`): imagem de perfil do GitHub exibida no voice channel, com fallback para iniciais; cross-reference com membros do servidor para obter avatar_url
- **Deteccao de atividade de voz (VAD)** (`voice.svelte.ts`): deteccao client-side de volume via Web Audio API — icone wifi pulsa verde quando o usuario fala, sem depender do backend; threshold de volume configurado para deteccao precisa
- **Gerenciamento de canais por admin** (`ChannelSidebar.svelte`, `App.svelte`): botao de delete (lixeira) aparece ao hover em text/voice channels para owner, admin e moderator; usa binding `DeleteChannel` existente
- **Gerenciamento de membros** (`MemberSidebar.svelte`, `App.svelte`): admin/owner pode clicar em membros para abrir popover com opcoes de alterar cargo (Admin/Moderator/Member) ou expulsar; usa `UpdateMemberRole` e `KickMember` bindings existentes
- **Amigos no voice no ActiveNow** (`ActiveNow.svelte`): amigos que estao no mesmo canal de voz aparecem no painel "Ativo agora" com card de voice mostrando icone wifi e nome do servidor

#### UX Improvements Round 4 (2026-02-21)

- **Paleta de cores verde** (`app.css`): accent trocado de roxo (#7c3aed) para verde (#16a34a) e backgrounds com undertone verde — contraste visual ao Discord
- **@ do GitHub no voice channel** (`ChannelSidebar.svelte`): usuarios conectados ao voice exibem `@github-username` (username real do GitHub OAuth)
- **Timer de voice channel** (`voice.svelte.ts`, `ChannelSidebar.svelte`): cronometro ao lado do nome do canal mostra tempo conectado (formato `mm:ss` ou `h:mm:ss`)
- **Icone de sinal de voz** (`ChannelSidebar.svelte`): cada usuario no voice channel tem icone wifi/sinal verde (speaking) ou cinza (mudo) ao lado direito, estilo Discord
- **Instrucoes de invite** (`FriendsList.svelte`): secao explicativa na aba "Pendente" com passo-a-passo de como adicionar amigos via @username do GitHub

#### UX Improvements (2026-02-21)

- **P2P: Create New Room** (`P2PPeerSidebar.svelte`, `p2p.svelte.ts`): botao "+" ao lado do label "Sala" permite criar nova sala, gerando novo room code via Wails binding (com fallback local). Sala inicia vazia e so aparece apos criar
- **Mode Switch** (`SettingsPanel.svelte`, `settings.svelte.ts`): secao "Connection Mode" na tab Account mostra modo atual (P2P/Servidor) com botao "Trocar modo" que redireciona ao ModeSelector
- **Light Theme** (`app.css`, `settings.svelte.ts`, `SettingsPanel.svelte`): tema claro completo via CSS custom properties em `html.light`, botoes Dark/Light funcionais em Appearance, tema persistido no localStorage
- **Message Search** (`MainContent.svelte`): botao de busca no header abre barra de pesquisa que chama `SearchMessages` via Wails binding e exibe resultados inline
- **Create Channels** (`ChannelSidebar.svelte`): botoes "+" nos headers de Text e Voice Channels com input inline para criar novos canais
- **Voice Channel Users** (`ChannelSidebar.svelte`, `voice.svelte.ts`): usuarios conectados aparecem abaixo do voice channel ativo com indicador verde de speaking, polling de status a cada 2s
- **Server Info Modal** (`ServerInfoModal.svelte`): clicar no nome do servidor abre modal com contagem de membros, convite (gerar/copiar) e opcao de excluir para owners
- **Home Page Funcional** (`FriendsList.svelte`, `DMSidebar.svelte`, `App.svelte`): clicar "Mensagem" em um amigo abre a DM, busca no sidebar DM filtra conversas, selecionar DM mostra area de conversa

#### UX Improvements Round 3 (2026-02-21)

- **Voice Channel @ Username** (`ChannelSidebar.svelte`): usuarios conectados ao voice channel exibem `@username` e indicador verde (bolinha sobre avatar, estilo Discord)
- **Animacoes globais** (`app.css`): adicionados keyframes fade-in, fade-in-up, fade-in-down, slide-in-left, slide-in-right, scale-in com classes utilitarias; aplicados em search bar, formularios, friend rows, DM view
- **Tooltip posicao fixa** (`Tooltip.svelte`): reescrito para usar `position: fixed` calculando coordenadas via getBoundingClientRect, eliminando overflow lateral no ServerSidebar
- **Pesquisa sem resultados** (`MainContent.svelte`): exibe mensagem "Nenhum resultado encontrado" com icone quando busca de mensagens retorna vazio
- **Sistema de amigos funcional** (`friends.svelte.ts`, `FriendsList.svelte`, `App.svelte`): substituido mock por sistema real com persistencia localStorage — adicionar amigo, pedidos pendentes (enviados/recebidos), aceitar/rejeitar, bloquear/desbloquear, busca de amigos
- **ActiveNow sem mock** (`ActiveNow.svelte`): removido viewer count aleatorio, componente mostra estado vazio quando nao ha amigos com atividade
- **Removido Concord Premium** (`DMSidebar.svelte`): botao decorativo "Concord Premium" removido do sidebar de DMs

### Fixed

- **P2P Logout** (`P2PApp.svelte`, `App.svelte`): botao "Log Out" no modo P2P agora para o P2P store e reseta o modo, voltando ao ModeSelector (antes era noop)
- **P2P Room Auto-creation** (`p2p.svelte.ts`): modo P2P nao cria sala automaticamente ao entrar; usuario deve clicar "Criar Sala" explicitamente
- **Status Icon Movement** (`ServerSidebar.svelte`): icone de status do usuario no final da sidebar nao se mexe mais ao passar o mouse sobre servidores

#### Phase 9: Central Server Online (2026-02-21)

- **PostgreSQL stdlib bridge** (`postgres.StdlibDB()`): bridges `pgxpool.Pool` to `database/sql` interface via `pgx/v5/stdlib`, enabling existing repositories to work with PostgreSQL
- **Placeholder adapter** (`postgres.Adapter`): translates SQLite `?` placeholders to PostgreSQL `$1, $2...` style, implements `querier` interface for auth/server/chat repositories
- **PG migration aligned** (`001_init.sql`): fixed `auth_sessions` column names to match repository code, added `search_vector` tsvector column with GIN index and auto-update trigger
- **PG full-text search** (`postgres.ChatSearcher`): PostgreSQL-specific search using `tsvector`/`plainto_tsquery`/`ts_headline` replacing SQLite FTS5
- **Central server wiring** (`cmd/server/main.go`): replaced nil service stubs with real PG-backed auth, server, and chat services
- **Docker deployment** (`deployments/docker/`): multi-stage Dockerfile, docker-compose with PostgreSQL 17 + Redis 7, health checks, `.env.example`
- **Server config template** (`config.server.json`): production-ready configuration for central server mode
- **API + adapter tests**: unit tests for placeholder replacement, adapter interface compliance, and API handler responses

#### P2P Full Mode — Modo P2P Completo (estilo Discord)

- **Layout espelho do Discord** (`P2PApp.svelte`): sidebar de peers (240px) + área de chat (flex-1), sem servidor central
- **P2PPeerSidebar** (`components/p2p/`): perfil local no topo, card de sala com room code copiável, input para entrar em sala, lista de peers com badge LAN/WAN
- **P2PChatArea** (`components/p2p/`): empty state, header com peer selecionado, chat bubbles (sent=direita, received=esquerda), input com Enter para enviar
- **P2P store reativo** (`p2p.svelte.ts`): polling de peers a cada 3s, `EventsOn('p2p:message')` para mensagens recebidas, cache de perfis, fallback silencioso fora do Wails
- **Protocolo P2P** (`internal/network/p2p/protocol.go`): envelope JSON com `type: 'profile'|'chat'`, funções `EncodeEnvelope`/`DecodeEnvelope`
- **Persistência SQLite** (`internal/store/sqlite/p2p_repository.go` + migration `007_p2p_messages.sql`): tabela `p2p_messages(id, peer_id, direction, content, sent_at)` com índice por peer
- **Handshake de perfil**: ao conectar um peer, troca automática de `{displayName, avatarDataUrl}` via protocolo Concord
- **Novos Wails bindings** (`main.go`): `InitP2PHost`, `SendP2PMessage`, `GetP2PMessages`, `SendP2PProfile`, `GetP2PPeerName`
- **E2E tests** (`tests/e2e/p2p.spec.ts`): testa ModeSelector, P2PProfile, room code, peers list, empty state — com stubs dos bindings Wails

#### P2P Onboarding — Seleção de Modo no Boot

- **Mode selector screen** (`ModeSelector.svelte`): primeira tela ao abrir o app, permite escolher entre modo P2P e Servidor Oficial
- **P2P profile screen** (`P2PProfile.svelte`): criação de identidade local (nome + avatar) para modo P2P sem conta
- **Boot routing** (`App.svelte`): roteamento de boot por `networkMode` — `null→ModeSelector`, `p2p+sem-perfil→P2PProfile`, `p2p+perfil→P2PApp`, `server→fluxo GitHub OAuth`
- **Settings store** (`settings.svelte.ts`): novos campos `networkMode: 'p2p'|'server'|null` e `p2pProfile: {displayName, avatarDataUrl?}`, persistidos no localStorage
- **`SelectAvatarFile` Wails binding** (`main.go`): diálogo nativo de seleção de imagem que retorna data URL base64
- **`RoomCode()` e `RoomRendezvous()`** (`internal/network/p2p/room.go`): código de sala determinístico e legível gerado do peer ID (e.g. `"amber-4271"`)
- Testes unitários para `RoomCode` (determinismo e formato)

## [1.0.0] - 2026-02-20

### Added

#### Phase 10: Polish & Hardening (2026-02-20)

- Settings panel with categories: Account, Audio, Appearance, Notifications, Language
  - Audio device selection via WebAudio API (input/output)
  - Theme selector (dark only, light prepared)
  - Notification toggle and sounds toggle
  - Translation language preferences (source/target)
  - Persisted to localStorage
- Desktop notification service wrapper (browser Notification API)
- Toast notification system (success/error/warning/info variants, auto-dismiss)
- Settings button wired in ChannelSidebar user panel
- E2E test skeletons with Playwright (auth, server, chat flows)
- CI/CD pipelines:
  - `ci.yml` — Go test + lint + frontend type check on PR
  - `release.yml` — Cross-platform desktop build + server binary on tag push
  - `security.yml` — govulncheck, npm audit, Trivy container scan (weekly)
- Security hardening: CSP headers, rate limiting, input sanitization, request size limits
- Documentation: API.md, SECURITY.md, CONTRIBUTING.md, P2P-PROTOCOL.md, VOICE-PIPELINE.md
- Version bumped to v1.0.0

#### Phase 9: Central Server (2026-02-20)

- PostgreSQL layer (`internal/store/postgres/`)
  - Connection pool via pgx/v5 with health check
  - Embedded SQL migration system with version tracking
  - Full schema: users, servers, channels, messages, attachments, server_members, sessions, audit_log, invites
  - Integration tests (skip if no PG available)
- Redis layer (`internal/store/redis/`)
  - Client wrapper with Set, Get, Delete, SetNX, Incr, Expire, Publish, Subscribe
  - Health check, retry, pool management via go-redis/v9
  - Integration tests (skip if no Redis available)
- HTTP API server (`internal/api/`) with chi v5 router
  - Full REST API: auth (device-code, token, refresh), servers CRUD, channels, members, invites, messages
  - Middleware stack: JWT auth, CORS, rate limiting, request logging, security headers, max body size, recovery
  - Cursor-based message pagination, full-text search endpoint
  - WebSocket signaling integration route
  - Offline message queue for reconnection delivery
  - httptest-based test suite
- Central server implementation (`cmd/server/main.go`)
  - PostgreSQL + Redis + chi HTTP server initialization
  - Graceful shutdown in reverse order
  - Health checks for all infrastructure components
- Docker deployment (`deployments/docker/`)
  - Multi-stage Dockerfile for Go server (distroless runtime)
  - Coturn TURN/relay server with configuration
  - docker-compose.yml: server + PostgreSQL 16 + Redis 7 + coturn
  - Health checks, volumes, environment variable configuration
  - `.env.example` template

#### Phase 8: Voice Translation (2026-02-20)

- PersonaPlex API client (`internal/translation/personaplex.go`)
  - HTTP text translation with JSON request/response
  - WebSocket streaming audio translation
  - Circuit breaker: auto-disable after consecutive high-latency failures, auto-reset on success
  - Configurable timeout, latency threshold, failure threshold
- Streaming translation pipeline (`internal/translation/stream.go`)
  - Goroutine bridge: voice engine → PersonaPlex → output
  - Graceful degradation: passes original audio if PersonaPlex fails
  - Start/Stop lifecycle with context cancellation
- Translation cache (`internal/translation/cache.go`)
  - Wraps existing LRU cache with SHA-256 key hashing
  - Cache key format: `translate:{src}:{tgt}:{hash(text)}`, TTL 1h
- Translation service (`internal/translation/service.go`)
  - Enable/Disable/GetStatus orchestration
  - Text translation with cache-first strategy
- SQLite migration `006_translation.sql` for persistent cache
- Wails bindings: EnableTranslation, DisableTranslation, GetTranslationStatus
- 14 unit tests (client, circuit breaker, cache, pipeline, service, concurrency)

#### Cache + Refresh Token (2026-02-20)

- LRU cache package (`internal/cache/lru.go`) — generic, thread-safe, with per-key TTL and prefix invalidation
- LRU cache integrated into server service (ListUserServers, ListChannels, ListMembers, GetServer) with 5min TTL
- Cache invalidation on all write operations (create/update/delete server, channel, member)
- Automatic access token refresh in frontend auth store with 2min pre-expiry buffer
- `ensureValidToken()` guard in servers, chat, and voice frontend stores
- CLAUDE.md principles 13 (Cache) and 14 (Refresh Token)

### Fixed

- RestoreSession now fetches full user profile from DB (display_name, avatar_url) instead of only JWT claims
- Added `GetUser(ctx, userID)` to auth repository for primary key lookup
- MemberSidebar now hidden by default, toggled via Members button in channel header
- Removed tooltip from user avatar in ServerSidebar that caused clipped tooltip artifact on hover

#### Phase 1: Foundation (2026-02-20)

- Project scaffolding with Wails v2 + Go 1.23
- Configuration system with environment variable support
  - JSON-based configuration with sensible defaults
  - Per-OS default paths (AppData/Library/XDG)
  - Full configuration validation
- Logging infrastructure with zerolog
  - Structured JSON logging
  - Performance logging utilities
  - Context-aware logger middleware
  - Automatic sensitive data sanitization
- Observability infrastructure
  - Prometheus metrics for all major subsystems (voice, chat, P2P, files, translation)
  - Health check system with component-level monitoring
  - Automatic health caching with configurable TTL
- SQLite database layer
  - Pure Go SQLite driver (modernc.org/sqlite, no CGo)
  - WAL mode for improved concurrency
  - Automatic migration system with embedded SQL files
  - Database utilities (vacuum, optimize, backup, integrity check)
  - Transaction helpers with automatic rollback
- Initial database schema (migration 001_init)
  - Users, servers, channels, messages tables
  - Full-text search for messages (FTS5)
  - File attachments support
  - Direct messages
  - Server member roles and permissions
  - Voice channel tracking
  - Authentication sessions
  - Server invites
  - User settings and preferences
  - P2P peer caching
  - Translation result caching
- Desktop application entry point (cmd/concord/main.go)
  - Wails integration
  - Application lifecycle management
  - Health check registration
  - Database initialization and migration
- Central server entry point stub (cmd/server/main.go)
  - Graceful shutdown handling
  - Signal handling (SIGINT, SIGTERM)
- Version management system
- Makefile with comprehensive build, test, and development commands
- Project documentation
  - README.md
  - ARCHITECTURE.md (comprehensive technical specification)
  - LICENSE (MIT)

#### Phase 7: File Sharing (2026-02-20)

- File system backend (internal/files)
  - Attachment model with SHA-256 hash for deduplication
  - File chunker: O(n/c) splitting with per-chunk and full-file SHA-256 integrity verification
  - File reassembly with chunk ordering validation and hash verification
  - Local filesystem storage with path traversal prevention (filepath.Base)
  - Unique path generation to avoid overwrites (numeric suffix)
  - File scanner with MIME whitelist and extension blocklist
    - Content-based MIME detection via http.DetectContentType (first 512 bytes)
    - MIME parameter normalization (strips "; charset=utf-8")
    - Blocked: .exe, .bat, .cmd, .com, .msi, .dll, .ps1, .scr, .pif, .vbs, .sys, .drv, .cpl, .inf, .reg
    - Allowed: images, documents, archives, audio, video, code/text
  - Service layer: upload (validate + hash + deduplicate + store), download, delete (ref-counted)
  - P2P transfer support: PrepareOffer, ChunkAttachment, StartReceive, ReceiveChunk, CompleteReceive
  - Repository: SQLite CRUD with GetByHash for deduplication
- Database migration (005_attachments.sql)
  - Attachments table with FK CASCADE to messages
  - Indexes on message_id and hash
- Wails bindings: 4 file methods (UploadFile, DownloadFile, GetAttachments, DeleteAttachment)
- Frontend file store extensions (chat.svelte.ts)
  - AttachmentData type and attachmentsByMessage reactive state
  - uploadFile, downloadFile, deleteAttachment, loadAttachments functions
- Frontend components
  - FileAttachment.svelte: file type icons (image, audio, video, pdf, archive, text), size formatting, download/delete actions
  - MessageInput updated: file selection via attach button, pending file preview with remove, combined text+file send
  - MessageBubble updated: displays file attachments below message content
  - MessageList/MainContent: attachment prop threading
  - App.svelte: file upload handler, browser download trigger, attachment loading per message
- 18 unit tests (chunker 6, scanner 6, storage 4, constants 2)
- MaxFileSize: 50 MB, DefaultChunkSize: 256 KB

#### Phase 6: Voice Chat (2026-02-20)

- Voice engine (internal/voice)
  - Audio constants: 48kHz sample rate, mono, 20ms frame duration, 960 samples/frame
  - PCM conversion utilities: int16↔float32 with clamping
  - Adaptive jitter buffer with sequence-ordered insertion and uint16 wraparound
  - Multi-stream audio mixer with per-stream volume and soft clipping (tanh-like saturation)
  - Energy-based Voice Activity Detection (VAD)
    - Adaptive noise floor estimation during silence
    - Dynamic threshold: noise floor + 15 dB margin
    - Configurable hangover frames to prevent speech cut-off
  - Voice engine orchestrator
    - State machine: disconnected → connecting → connected
    - WebRTC peer connections with Opus audio tracks (Pion WebRTC v4)
    - Mute/deafen toggle with deafen-implies-mute logic
    - Active speaker tracking, peer management (add/remove)
    - SDP offer/answer exchange, ICE candidate handling
    - State change and speaker change callbacks
- Wails bindings: 5 voice methods (JoinVoice, LeaveVoice, ToggleMute, ToggleDeafen, GetVoiceStatus)
- Voice engine cleanup on application shutdown
- Frontend voice store (voice.svelte.ts)
  - Reactive state with Svelte 5 runes
  - Join/leave, toggle mute/deafen, status refresh
- Voice UI components
  - VoiceControls: connected indicator, active speakers list, mute/deafen buttons, disconnect button
  - ChannelSidebar updated: voice channel click to join/toggle, connected channel indicator, VoiceControls integration
- App.svelte wired with voice store (join/leave handlers, mute/deafen controls)
- 29 unit tests (codec 4, jitter buffer 5, mixer 5, VAD 4, engine 6, utilities 5)

#### Phase 5: P2P Networking (2026-02-20)

- Wire protocol (pkg/protocol)
  - Binary wire format: [1 byte type][4 bytes length (big-endian)][payload (msgpack)]
  - 17 message types: text (send/edit/delete), voice (join/leave/data/mute), file (offer/accept/chunk/complete), server sync, presence, typing, ping/pong
  - Encode/Decode with msgpack serialization, 1 MB max payload
  - 7 unit tests (round-trip, all types, payload size limit, edge cases)
- End-to-end encryption (pkg/crypto)
  - X25519 key exchange for peer key agreement
  - HKDF-SHA256 key derivation with "concord-e2ee-v1" info
  - AES-256-GCM symmetric encryption (nonce || ciphertext format)
  - E2EEManager: AddPeerKey, RemovePeer, Encrypt, Decrypt, thread-safe
  - 7 unit tests (key generation, bidirectional encrypt/decrypt, tamper detection, third-party cannot decrypt)
- libp2p P2P host (internal/network/p2p)
  - QUIC + TCP dual transport with Noise security
  - NAT port mapping + hole punching + relay enabled
  - mDNS for LAN peer discovery (auto-connect)
  - Kademlia DHT for internet peer discovery with bootstrap peers
  - Stream-based message handler (/concord/1.0.0 protocol)
  - Connect, SendData, Peers, PeerCount operations
  - FindPeers via DHT routing discovery + advertise
  - 5 integration tests (host lifecycle, connect, send/receive, peer info)
- WebSocket signaling (internal/network/signaling)
  - Signal types: join, leave, offer, answer, peer_list, peer_joined, peer_left, error
  - Signaling server: HTTP handler with WebSocket upgrade, channel-based peer tracking
  - Per-connection mutex for concurrent write safety (gorilla/websocket)
  - Broadcast to channel peers, forward offers to specific peers
  - Signaling client: connect, on(handler), join/leave channel, send offers
  - 11 unit tests (signal encode/decode, server-client integration, peer join/leave, offer forwarding, multiple channels)

#### Phase 4: Text Chat (2026-02-20)

- Chat domain layer (internal/chat)
  - Message model with author JOIN (author_name, author_avatar)
  - Cursor-based pagination (before/after message ID, configurable limit)
  - Full-text search via FTS5 with snippet extraction and ranking
  - Content validation (max 4000 chars, no empty/whitespace-only)
- Chat repository
  - Save, GetByID, GetByChannel (3 query modes: before/after/latest)
  - Update with automatic edited_at timestamp
  - Delete, Search with FTS5 snippet() and rank ordering
  - CountByChannel for stats
- Chat service
  - SendMessage, GetMessages, EditMessage (author-only), DeleteMessage (author or manager)
  - SearchMessages with configurable result limit
- Database migration (004_messages.sql)
  - messages table with channel_id, author_id FK, type CHECK constraint
  - Composite index on (channel_id, created_at DESC) for O(log n) pagination
  - FTS5 virtual table with auto-sync triggers (INSERT/UPDATE/DELETE)
- Wails bindings: 5 chat methods (SendMessage, GetMessages, EditMessage, DeleteMessage, SearchMessages)
- Frontend chat store (chat.svelte.ts)
  - Reactive message list with Svelte 5 runes
  - Auto-reverse API response (newest-first → oldest-first for display)
  - Cursor-based load-older-messages support
  - Search with results and query state
- Chat UI components
  - MessageBubble: avatar grouping, timestamp formatting (today/yesterday/date), edit/delete hover actions, (edited) indicator
  - MessageList: auto-scroll to bottom, load-older on scroll-to-top, 5-minute avatar grouping, welcome message for empty channels
  - MessageInput: Enter-to-send (Shift+Enter for newline), attach/emoji buttons, conditional send button, disabled while sending
- MainContent updated with real MessageList + MessageInput (replaced mock messages)
- App.svelte wired with chat store (loadMessages on channel select, send/delete handlers, manager role detection)
- Unit tests: 4 tests (max length, content validation, pagination defaults, pagination limit)

#### Phase 3: Server Management (2026-02-20)

- Server CRUD operations
  - Create server with auto-generated invite code and default channels (#general text + General voice)
  - Update server name/icon (requires PermManageServer)
  - Delete server (owner only, cascades to channels + members)
  - List user's servers via JOIN on server_members
- Channel management
  - Create text/voice channels (requires PermManageChannels)
  - List channels ordered by position
  - Update channel name/type/position
  - Delete channels (requires PermManageChannels)
- Member management with RBAC permissions
  - 4 roles: owner > admin > moderator > member
  - 6 permissions: ManageServer, ManageChannels, ManageMembers, CreateInvite, SendMessages, ManageMessages
  - Role hierarchy enforcement: cannot kick/modify equal or higher roles
  - Cannot promote above own role or assign owner directly
- Invite system
  - Generate 8-character random invite codes (base32, cryptographically random)
  - Redeem invite to join server (idempotent — returns server if already member)
  - Inspect invite for server name + member count
- Database migration (003_servers.sql)
  - servers, channels, server_members tables
  - Indexes on owner_id, invite_code (unique), server_id, user_id
- Wails bindings: 14 server management methods exposed to frontend
- Frontend server store with Svelte 5 runes
  - Reactive server/channel/member lists
  - Auto-load channels + members on server selection
- UI: CreateServer modal and JoinServer modal
- ServerSidebar updated with onAddServer callback
- App.svelte wired with real server data (servers, channels, members from Go backend)
- Unit tests: 6 tests (permissions, role hierarchy, invite code generation)

#### Phase 2: GitHub OAuth Authentication (2026-02-20)

- GitHub Device Flow (RFC 8628) for desktop app authentication
  - No callback server needed — device code + user code flow
  - Handles authorization_pending, slow_down, expired_token, access_denied
- JWT token management (HS256)
  - Access token (15 min) + refresh token (30 days) pair generation
  - Token validation with issuer distinction (concord vs concord-refresh)
  - Refresh token rotation on session restore
- Encrypted session storage
  - Refresh tokens encrypted at rest with AES-256-GCM
  - SHA-256 token hash for lookup, base64 encoding for storage
  - Automatic expired session cleanup
- Auth repository with SQLite persistence
  - User upsert from GitHub profile (ON CONFLICT DO UPDATE)
  - Session CRUD with indexed queries on user_id and expires_at
  - Database migration (002_auth.sql) for auth_sessions table
- Auth service orchestration
  - StartLogin → CompleteLogin → RestoreSession → Logout lifecycle
  - Wails bindings exposed to frontend
- Frontend authentication UI
  - Login page with GitHub Device Flow (user code display, clipboard copy, browser open)
  - Auth state management with Svelte 5 runes ($state, $derived, $effect)
  - Session persistence via localStorage user ID + encrypted refresh token
  - Loading splash, polling state, error display
  - Conditional routing: Login view vs Layout Shell based on auth state
- JWT unit tests (8 tests, all passing)
  - Token generation, validation, refresh, wrong secret, issuer check
- Config: `CONCORD_GITHUB_CLIENT_ID` environment variable support

#### Phase 1.6: Layout Shell (2026-02-20)

- Discord-like 4-panel layout (ServerSidebar, ChannelSidebar, MainContent, MemberSidebar)
- ServerSidebar: server icons with active indicator, notification dots, home/add buttons
- ChannelSidebar: server name header, text/voice channel list with unread badges, user panel with mic/deafen/settings
- MainContent: channel header with search/members actions, welcome message, mock chat messages, message input bar
- MemberSidebar: online/offline member groups with avatar, status, and role display
- All layout components in `frontend/src/lib/components/layout/`
- Mock data for servers, channels, members, and messages

#### Phase 1.2: Void Design System (2026-02-20)

- Frontend scaffolding with Svelte 5 + Vite + TailwindCSS v4 + TypeScript
- Void design tokens: color palette, shadows, radius, typography (Inter, JetBrains Mono)
- TailwindCSS v4 @theme integration with Void CSS custom properties
- 9 base UI components in `frontend/src/lib/components/ui/`:
  - Button (solid/outline/ghost/danger variants, sm/md/lg sizes, loading state)
  - Input (text/password/search, error state, password visibility toggle)
  - Modal (dialog-based, focus trap, Escape key, backdrop click)
  - Badge (default/success/warning/danger variants)
  - Avatar (image/initials fallback, status indicator: online/idle/dnd/offline)
  - Tooltip (top/bottom/left/right positioning, configurable delay)
  - Toggle (switch role, accessible, keyboard navigation)
  - Dropdown (keyboard navigation, click outside, highlighted state)
  - Card (static and interactive variants with glow hover)
- All components: Svelte 5 runes ($props, $state, $derived), ARIA attributes, CSS transitions
- Barrel export from `frontend/src/lib/components/ui/index.ts`
- Custom scrollbar styling for Void theme
- Showcase App.svelte demonstrating all components

### Security

- JWT secret validation (minimum 32 chars in production)
- SQLite database encryption support
- Rate limiting configuration
- File upload validation and size limits
- Sensitive data sanitization in logs
- Foreign key constraints enforcement
- Prepared statements for SQL injection prevention

### Performance

- O(1) cache operations with LRU cache
- O(log n) database queries with proper indexing
- Connection pooling for database connections
- WAL mode for SQLite (improved concurrency)
- Memory-mapped I/O for SQLite (30GB mmap_size)
- 64MB SQLite cache
- Optimized pragmas for performance

---

## Version History

- **1.0.0** - Production release with all 10 phases complete
- **0.1.0-dev** - Initial development version
