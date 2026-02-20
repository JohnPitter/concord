# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

## [0.1.0] - TBD

Initial development release - in progress.

### Planned Features

- Phase 2: GitHub OAuth authentication
- Phase 3: Server management (CRUD, channels, members, invites)
- Phase 4: Real-time text chat with WebSocket
- Phase 5: P2P networking with libp2p
- Phase 6: Voice chat with WebRTC
- Phase 7: File sharing
- Phase 8: Voice translation with NVIDIA PersonaPlex
- Phase 9: Central server (PostgreSQL, Redis, REST API)
- Phase 10: Production hardening and release

---

## Version History

- **0.1.0-dev** - Current development version
