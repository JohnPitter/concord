# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
