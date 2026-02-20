# CONCORD — Architectural Plan

> Open-source, privacy-first voice & text chat platform built in Go + Wails.
> "Chat de voz para amigos que jogam com maximo de privacidade. No Scam Bro"

---

## Table of Contents

1. [Summary](#1-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Technology Stack](#3-technology-stack)
4. [Project Structure](#4-project-structure)
5. [Module Architecture](#5-module-architecture)
6. [Hybrid Networking Model (P2P + Central Server)](#6-hybrid-networking-model)
7. [Voice Pipeline](#7-voice-pipeline)
8. [Data Layer](#8-data-layer)
9. [Authentication (GitHub OAuth)](#9-authentication)
10. [Security Model](#10-security-model)
11. [Frontend Architecture & Design System](#11-frontend-architecture)
12. [Observability & Logging](#12-observability--logging)
13. [Testing Strategy](#13-testing-strategy)
14. [Phased Build Plan](#14-phased-build-plan)
15. [Big O Complexity Analysis](#15-big-o-complexity-analysis)
16. [Risks & Edge Cases](#16-risks--edge-cases)
17. [Files to Create](#17-files-to-create)
18. [Recommended Agents](#18-recommended-agents)

---

## 1. Summary

**Concord** is a privacy-first, open-source Discord alternative designed for gamers. It provides real-time voice chat, text messaging, file sharing, and server management — all running as a native desktop application built with **Go (Wails)** and **Svelte 5**. It operates in a **hybrid P2P + centralized server** model (similar to Hamachi), where voice traffic flows peer-to-peer when possible and through relay servers when NAT traversal fails. A "power-up" feature enables **real-time voice translation** via NVIDIA PersonaPlex. Authentication is exclusively through **GitHub OAuth**.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    CONCORD DESKTOP APP                       │
│  ┌──────────────────────┐  ┌─────────────────────────────┐  │
│  │   Go Backend (Wails) │  │   Svelte 5 Frontend (UI)    │  │
│  │                      │◄─►                              │  │
│  │  ┌────────────────┐  │  │  ┌───────────────────────┐  │  │
│  │  │ Auth Service    │  │  │  │ Design System (Void)  │  │  │
│  │  │ Chat Service    │  │  │  │ Voice Controls        │  │  │
│  │  │ Voice Engine    │  │  │  │ Chat Interface        │  │  │
│  │  │ File Service    │  │  │  │ Server Browser        │  │  │
│  │  │ Server Manager  │  │  │  │ Settings Panel        │  │  │
│  │  │ P2P Manager     │  │  │  │ File Manager          │  │  │
│  │  │ Translation Svc │  │  │  └───────────────────────┘  │  │
│  │  └────────────────┘  │  └─────────────────────────────┘  │
│  └──────────┬───────────┘                                    │
│             │                                                │
│  ┌──────────▼───────────┐                                    │
│  │   Networking Layer    │                                    │
│  │  ┌─────────┐ ┌─────┐ │                                    │
│  │  │ libp2p  │ │Pion │ │                                    │
│  │  │ (data)  │ │(RTC)│ │                                    │
│  │  └────┬────┘ └──┬──┘ │                                    │
│  └───────┼─────────┼────┘                                    │
└──────────┼─────────┼────────────────────────────────────────┘
           │         │
     ┌─────▼─────────▼─────┐
     │  Internet / LAN      │
     │  ┌─────────────────┐ │
     │  │ Signaling Server │ │   (Go, WebSocket)
     │  │ Relay Server     │ │   (TURN/libp2p relay)
     │  │ Auth Server      │ │   (GitHub OAuth)
     │  │ API Server       │ │   (REST + WS)
     │  └─────────────────┘ │
     └──────────────────────┘
```

### Key Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Desktop Framework | Wails v2 | Go-native, small binary, uses OS webview (not Electron) |
| Frontend | Svelte 5 + TypeScript | Reactive, compiled, small bundle, excellent DX |
| P2P Layer | libp2p (go-libp2p) | Mature NAT traversal, QUIC transport, relay circuits, hole punching |
| Voice/WebRTC | Pion WebRTC v4 | Pure Go, MIT license, excellent P2P media support |
| Audio Codec | Opus (pion/opus) | Pure Go, no CGo, royalty-free, optimized for voice |
| Database (Client) | SQLite (modernc.org/sqlite) | Pure Go, no CGo, embedded, zero-config |
| Database (Server) | PostgreSQL | Scalable, ACID, rich querying for server-side data |
| Cache | Redis (server) + in-memory LRU (client) | Low-latency caching for both modes |
| Auth | golang.org/x/oauth2 + GitHub OAuth | Official lib, proven, secure |
| Voice Translation | NVIDIA PersonaPlex API | Open model, ~170ms latency, full-duplex |
| CSS Framework | TailwindCSS v4 | Utility-first, tree-shaken, design-system friendly |
| Build Tool | Vite | Fast HMR, excellent Svelte integration |
| Noise Suppression | RNNoise (WASM) | ML-based, runs in frontend via WebAssembly |
| Logging | zerolog | Zero-allocation JSON logger for Go |
| Testing | Go testing + testify + Playwright | Unit/integration/e2e pyramid |

---

## 3. Technology Stack

### Backend (Go)
```
go 1.23+
├── github.com/wailsapp/wails/v2          # Desktop framework
├── github.com/pion/webrtc/v4             # WebRTC (voice P2P)
├── github.com/pion/opus                  # Opus codec (pure Go)
├── github.com/libp2p/go-libp2p          # P2P networking + NAT traversal
├── golang.org/x/oauth2                   # GitHub OAuth
├── github.com/gorilla/websocket          # WebSocket for signaling
├── modernc.org/sqlite                    # SQLite (pure Go, no CGo)
├── github.com/rs/zerolog                 # Structured logging
├── github.com/go-chi/chi/v5             # HTTP router (server)
├── github.com/redis/go-redis/v9         # Redis client (server mode)
├── github.com/golang-jwt/jwt/v5         # JWT tokens
├── github.com/stretchr/testify          # Test assertions
├── github.com/prometheus/client_golang   # Metrics
└── golang.org/x/crypto                   # Cryptographic primitives
```

### Frontend (Svelte 5 + TypeScript)
```
├── svelte@5                              # UI framework
├── @sveltejs/vite-plugin-svelte         # Vite integration
├── tailwindcss@4                         # Styling
├── @tailwindcss/vite                    # TailwindCSS Vite plugin
├── typescript@5                          # Type safety
├── @anthropic-ai/rnnoise-wasm          # Noise suppression (or equivalent WASM build)
├── @playwright/test                      # E2E testing
├── vitest                               # Unit testing
└── @iconify/svelte                      # Icon system
```

---

## 4. Project Structure

```
concord/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                       # CI pipeline
│   │   ├── release.yml                  # Release builds
│   │   └── security.yml                 # Security scanning
│   └── ISSUE_TEMPLATE/
├── cmd/
│   ├── concord/                         # Desktop app entry point
│   │   └── main.go
│   └── server/                          # Central server entry point
│       └── main.go
├── internal/                            # Private application code
│   ├── auth/                            # GitHub OAuth + JWT
│   │   ├── github.go
│   │   ├── jwt.go
│   │   ├── middleware.go
│   │   └── auth_test.go
│   ├── chat/                            # Text messaging
│   │   ├── service.go
│   │   ├── handler.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   └── chat_test.go
│   ├── voice/                           # Voice engine
│   │   ├── engine.go                    # Core voice engine
│   │   ├── capture.go                   # Audio capture
│   │   ├── playback.go                  # Audio playback
│   │   ├── mixer.go                     # Audio mixing
│   │   ├── codec.go                     # Opus encode/decode
│   │   ├── isolation.go                 # Voice isolation logic
│   │   ├── jitter.go                    # Jitter buffer
│   │   └── voice_test.go
│   ├── translation/                     # NVIDIA PersonaPlex integration
│   │   ├── personaplex.go              # API client
│   │   ├── stream.go                    # Streaming pipeline
│   │   ├── cache.go                     # Translation cache
│   │   └── translation_test.go
│   ├── server/                          # Server management (create/join/manage)
│   │   ├── service.go
│   │   ├── handler.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   ├── permissions.go
│   │   └── server_test.go
│   ├── files/                           # File sharing
│   │   ├── service.go
│   │   ├── handler.go
│   │   ├── storage.go                   # Storage abstraction
│   │   ├── chunker.go                   # File chunking for P2P
│   │   ├── scanner.go                   # Malware/content scanner
│   │   └── files_test.go
│   ├── network/                         # Networking layer
│   │   ├── p2p/
│   │   │   ├── host.go                  # libp2p host setup
│   │   │   ├── discovery.go             # Peer discovery
│   │   │   ├── relay.go                 # Relay management
│   │   │   ├── nat.go                   # NAT traversal config
│   │   │   └── p2p_test.go
│   │   ├── signaling/
│   │   │   ├── server.go               # WebSocket signaling
│   │   │   ├── client.go               # Signaling client
│   │   │   └── signaling_test.go
│   │   └── transport/
│   │       ├── quic.go                  # QUIC transport config
│   │       ├── webrtc.go               # Pion WebRTC setup
│   │       └── transport_test.go
│   ├── store/                           # Data layer
│   │   ├── sqlite/
│   │   │   ├── sqlite.go               # SQLite connection
│   │   │   ├── migrations/             # SQL migrations
│   │   │   │   ├── 001_init.sql
│   │   │   │   ├── 002_servers.sql
│   │   │   │   └── ...
│   │   │   └── sqlite_test.go
│   │   ├── postgres/                    # Server-side DB
│   │   │   ├── postgres.go
│   │   │   ├── migrations/
│   │   │   └── postgres_test.go
│   │   ├── cache/
│   │   │   ├── lru.go                   # Client-side LRU cache
│   │   │   ├── redis.go                # Server-side Redis
│   │   │   └── cache_test.go
│   │   └── repository.go               # Repository interfaces
│   ├── security/                        # Security utilities
│   │   ├── crypto.go                    # Encryption helpers
│   │   ├── ratelimit.go                # Rate limiting
│   │   ├── sanitize.go                 # Input sanitization
│   │   ├── csrf.go                     # CSRF protection
│   │   └── security_test.go
│   ├── observability/                   # Logging & metrics
│   │   ├── logger.go                    # zerolog setup
│   │   ├── metrics.go                  # Prometheus metrics
│   │   ├── tracing.go                  # Distributed tracing
│   │   └── health.go                   # Health checks
│   └── config/                          # Configuration
│       ├── config.go                    # Config struct + loading
│       ├── defaults.go                 # Default values
│       └── config_test.go
├── pkg/                                 # Public reusable packages
│   ├── protocol/                        # Wire protocol definitions
│   │   ├── messages.go                 # Message types
│   │   ├── events.go                   # Event types
│   │   └── errors.go                   # Error codes
│   ├── crypto/                          # Public crypto utilities
│   │   ├── e2ee.go                     # End-to-end encryption
│   │   ├── keys.go                     # Key management
│   │   └── crypto_test.go
│   └── version/
│       └── version.go                   # Version info
├── frontend/                            # Svelte 5 frontend
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── ui/                  # Design system primitives
│   │   │   │   │   ├── Button.svelte
│   │   │   │   │   ├── Input.svelte
│   │   │   │   │   ├── Modal.svelte
│   │   │   │   │   ├── Avatar.svelte
│   │   │   │   │   ├── Badge.svelte
│   │   │   │   │   ├── Tooltip.svelte
│   │   │   │   │   ├── Toast.svelte
│   │   │   │   │   ├── Dropdown.svelte
│   │   │   │   │   ├── Sidebar.svelte
│   │   │   │   │   └── index.ts
│   │   │   │   ├── chat/
│   │   │   │   │   ├── MessageList.svelte
│   │   │   │   │   ├── MessageInput.svelte
│   │   │   │   │   ├── MessageBubble.svelte
│   │   │   │   │   ├── FileAttachment.svelte
│   │   │   │   │   └── EmojiPicker.svelte
│   │   │   │   ├── voice/
│   │   │   │   │   ├── VoiceChannel.svelte
│   │   │   │   │   ├── VoiceControls.svelte
│   │   │   │   │   ├── VoiceIndicator.svelte
│   │   │   │   │   ├── TranslationToggle.svelte
│   │   │   │   │   └── NoiseGate.svelte
│   │   │   │   ├── server/
│   │   │   │   │   ├── ServerList.svelte
│   │   │   │   │   ├── ServerCard.svelte
│   │   │   │   │   ├── CreateServer.svelte
│   │   │   │   │   ├── ChannelList.svelte
│   │   │   │   │   └── MemberList.svelte
│   │   │   │   └── auth/
│   │   │   │       ├── LoginScreen.svelte
│   │   │   │       └── GitHubButton.svelte
│   │   │   ├── stores/                  # Svelte stores (state management)
│   │   │   │   ├── auth.ts
│   │   │   │   ├── chat.ts
│   │   │   │   ├── voice.ts
│   │   │   │   ├── servers.ts
│   │   │   │   ├── ui.ts
│   │   │   │   └── settings.ts
│   │   │   ├── services/                # Frontend services (call Go bindings)
│   │   │   │   ├── wails.ts            # Wails runtime helpers
│   │   │   │   ├── audio.ts            # Audio device management
│   │   │   │   └── notifications.ts    # Desktop notifications
│   │   │   ├── utils/
│   │   │   │   ├── format.ts
│   │   │   │   ├── validators.ts
│   │   │   │   └── constants.ts
│   │   │   └── types/
│   │   │       ├── chat.ts
│   │   │       ├── voice.ts
│   │   │       ├── server.ts
│   │   │       └── user.ts
│   │   ├── pages/
│   │   │   ├── Login.svelte
│   │   │   ├── Home.svelte
│   │   │   ├── Server.svelte
│   │   │   ├── Settings.svelte
│   │   │   └── DirectMessage.svelte
│   │   ├── App.svelte                   # Root component
│   │   ├── main.ts                      # Entry point
│   │   └── app.css                      # Global styles + Tailwind
│   ├── static/
│   │   ├── fonts/
│   │   └── icons/
│   ├── tests/
│   │   ├── unit/
│   │   └── e2e/
│   ├── index.html
│   ├── vite.config.ts
│   ├── svelte.config.js
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   └── package.json
├── deployments/                         # Server deployment configs
│   ├── docker/
│   │   ├── Dockerfile.server
│   │   ├── Dockerfile.relay
│   │   └── docker-compose.yml
│   └── k8s/
│       └── ...
├── scripts/                             # Build & dev scripts
│   ├── build.sh
│   ├── dev.sh
│   └── generate-bindings.sh
├── docs/                                # Documentation
│   ├── API.md
│   ├── P2P-PROTOCOL.md
│   ├── VOICE-PIPELINE.md
│   ├── SECURITY.md
│   └── CONTRIBUTING.md
├── go.mod
├── go.sum
├── wails.json                           # Wails config
├── Makefile
├── CHANGELOG.md
├── LICENSE                              # MIT or AGPL-3.0
└── README.md
```

---

## 5. Module Architecture

### Clean Architecture Layers

```
┌────────────────────────────────────────┐
│         Presentation Layer             │  Wails Bindings + Svelte UI
├────────────────────────────────────────┤
│         Application Layer              │  Services (business logic orchestration)
├────────────────────────────────────────┤
│         Domain Layer                   │  Models, Interfaces, Business Rules
├────────────────────────────────────────┤
│         Infrastructure Layer           │  DB, Network, External APIs, Cache
└────────────────────────────────────────┘
```

### Dependency Rules
- **Domain** has ZERO external dependencies
- **Application** depends only on Domain interfaces
- **Infrastructure** implements Domain interfaces
- **Presentation** calls Application services via Wails bindings

### Core Interfaces (Domain Layer)

```go
// internal/chat/repository.go
type MessageRepository interface {
    Save(ctx context.Context, msg *Message) error
    GetByChannel(ctx context.Context, channelID string, opts PaginationOpts) ([]*Message, error)
    GetByID(ctx context.Context, id string) (*Message, error)
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, query string, opts SearchOpts) ([]*Message, error)
}

// internal/server/repository.go
type ServerRepository interface {
    Create(ctx context.Context, server *Server) error
    GetByID(ctx context.Context, id string) (*Server, error)
    ListByUser(ctx context.Context, userID string) ([]*Server, error)
    Update(ctx context.Context, server *Server) error
    Delete(ctx context.Context, id string) error
    AddMember(ctx context.Context, serverID, userID string, role Role) error
    RemoveMember(ctx context.Context, serverID, userID string) error
}

// internal/voice/engine.go
type VoiceEngine interface {
    JoinChannel(ctx context.Context, channelID string) error
    LeaveChannel(ctx context.Context) error
    Mute() error
    Unmute() error
    SetInputDevice(deviceID string) error
    SetOutputDevice(deviceID string) error
    EnableTranslation(targetLang string) error
    DisableTranslation() error
    GetActiveSpeakers() []SpeakerInfo
}

// internal/network/p2p/host.go
type P2PHost interface {
    Start(ctx context.Context) error
    Stop() error
    Connect(ctx context.Context, peerID string) error
    SendData(ctx context.Context, peerID string, data []byte) error
    OnMessage(handler func(peerID string, data []byte))
    Peers() []PeerInfo
    ID() string
}
```

---

## 6. Hybrid Networking Model

### Strategy: P2P First, Relay Fallback

```
┌──────────────────────────────────────────────────┐
│              Connection Decision Tree              │
├──────────────────────────────────────────────────┤
│                                                    │
│  1. Try DIRECT P2P (same LAN?)                     │
│     ├─ YES → mDNS discovery, direct connection     │
│     └─ NO  → continue                              │
│                                                    │
│  2. Try NAT HOLE PUNCHING (libp2p + QUIC)          │
│     ├─ SUCCESS → direct P2P over internet           │
│     └─ FAIL    → continue                           │
│                                                    │
│  3. Use RELAY (libp2p circuit relay v2)             │
│     └─ Route through central relay server           │
│                                                    │
│  4. VOICE: Always attempt Pion WebRTC first         │
│     ├─ ICE candidates → direct P2P                  │
│     └─ TURN server fallback                         │
│                                                    │
└──────────────────────────────────────────────────┘
```

### P2P Architecture Details

**Transport Stack:**
- **QUIC** as primary transport (low latency, multiplexed, encrypted by default)
- **TCP** as fallback transport
- **WebRTC** for voice/media streams specifically

**NAT Traversal (Hamachi-like behavior):**
1. Client starts → creates libp2p host with QUIC transport
2. Registers with **signaling/rendezvous server** (central)
3. When connecting to peer:
   - AutoNAT detects NAT type
   - DCUtR (Direct Connection Upgrade through Relay) protocol attempts hole punching
   - If hole punching succeeds → direct P2P
   - If fails → traffic flows through relay server

**Virtual Network (Hamachi model):**
- Each server (guild) creates a **virtual overlay network**
- Members get a virtual peer ID within the network
- Data messages, files, and voice all flow through this overlay
- The overlay is encrypted end-to-end using server-specific keys

### Data Flow

```
Text Messages:
  Sender → libp2p stream → [P2P or Relay] → Recipient
  + Async sync to central server for persistence/offline delivery

Voice:
  Sender → Audio Capture → Opus Encode → Pion WebRTC → [P2P/TURN] → Recipient
  → Opus Decode → Jitter Buffer → Audio Playback

Files:
  Sender → Chunk File → Encrypt Chunks → libp2p bitswap → Recipient
  + Metadata stored on central server
```

### Central Server Responsibilities
1. **Signaling** — WebSocket-based, coordinates WebRTC/libp2p connections
2. **User Registry** — GitHub OAuth, profiles, online status
3. **Server Registry** — Guild metadata, channels, permissions
4. **Message Relay** — Stores messages for offline users, syncs on reconnect
5. **Relay Node** — libp2p circuit relay for peers that can't hole punch
6. **TURN Server** — Fallback for WebRTC voice when P2P fails

---

## 7. Voice Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                        VOICE PIPELINE                            │
│                                                                   │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌───────────┐  │
│  │ Capture   │───►│ Noise    │───►│ Voice    │───►│ Opus      │  │
│  │ (Mic)     │    │ Suppress │    │ Isolate  │    │ Encode    │  │
│  │           │    │ (RNNoise)│    │ (WASM)   │    │ (Pion)    │  │
│  └──────────┘    └──────────┘    └──────────┘    └─────┬─────┘  │
│                                                         │        │
│                                    ┌────────────────────┘        │
│                                    ▼                             │
│                          ┌──────────────┐                        │
│                          │ [Optional]   │                        │
│                          │ PersonaPlex  │  ◄── Translation       │
│                          │ Translation  │      Power-Up          │
│                          └──────┬───────┘                        │
│                                 │                                │
│                                 ▼                                │
│                     ┌───────────────────┐                        │
│                     │  Pion WebRTC      │                        │
│                     │  DataChannel /    │  ───► P2P / TURN       │
│                     │  MediaTrack       │                        │
│                     └───────────────────┘                        │
│                                                                   │
│  ═══════════════════ RECEIVING SIDE ═════════════════════════     │
│                                                                   │
│  ┌───────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │ WebRTC    │───►│ Jitter   │───►│ Opus     │───►│ Audio    │  │
│  │ Receive   │    │ Buffer   │    │ Decode   │    │ Playback │  │
│  │           │    │ (50ms)   │    │ (Pion)   │    │ (Speaker)│  │
│  └───────────┘    └──────────┘    └──────────┘    └──────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Voice Engine Implementation

```go
// internal/voice/engine.go
type Engine struct {
    mu            sync.RWMutex
    capturer      *Capturer          // Audio input
    player        *Player            // Audio output
    codec         *OpusCodec         // Opus encode/decode
    jitter        *JitterBuffer      // Adaptive jitter buffer
    mixer         *Mixer             // Multi-stream audio mixer
    webrtc        *WebRTCTransport   // Pion WebRTC
    translator    *TranslationPipe   // PersonaPlex (optional)
    channelID     string
    muted         bool
    deafened      bool
    logger        zerolog.Logger
    metrics       *VoiceMetrics
}
```

### Audio Settings
- **Sample Rate:** 48000 Hz
- **Channels:** 1 (mono) for voice
- **Frame Size:** 20ms (960 samples at 48kHz)
- **Bitrate:** 64 kbps (configurable 32-128 kbps)
- **Jitter Buffer:** Adaptive, 50ms default, range 20-200ms

### Voice Isolation Strategy
1. **Frontend (WASM):** RNNoise for real-time noise suppression before encoding
2. **Backend (Go):** Silence detection (VAD - Voice Activity Detection) to skip transmitting silence
3. **Optional Enhancement:** WebRTC's built-in AEC (Acoustic Echo Cancellation) via Pion

### Translation Power-Up (NVIDIA PersonaPlex)

```go
// internal/translation/personaplex.go
type PersonaPlexClient struct {
    apiURL     string
    httpClient *http.Client
    streamConn *websocket.Conn  // Persistent WS for streaming
    cache      *TranslationCache
    logger     zerolog.Logger
}

// Stream audio to PersonaPlex for real-time translation
func (c *PersonaPlexClient) TranslateStream(
    ctx context.Context,
    inputAudio <-chan []byte,     // Opus-encoded audio frames
    sourceLang string,
    targetLang string,
) (<-chan []byte, error)          // Translated audio frames
```

**Flow:**
1. User enables "Translation Power-Up" in voice channel settings
2. Outgoing audio → sent to PersonaPlex API via WebSocket stream
3. PersonaPlex returns translated audio frames in ~170ms
4. Translated frames injected into the outgoing WebRTC stream
5. Other users hear the translated version

**Caching:** Common phrases/words cached locally to reduce API calls.

---

## 8. Data Layer

### Client-Side (SQLite)

```sql
-- migrations/001_init.sql
CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    github_id   INTEGER UNIQUE NOT NULL,
    username    TEXT NOT NULL,
    avatar_url  TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS servers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    icon_url    TEXT,
    owner_id    TEXT NOT NULL REFERENCES users(id),
    invite_code TEXT UNIQUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS channels (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL CHECK(type IN ('text', 'voice')),
    position    INTEGER DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id          TEXT PRIMARY KEY,
    channel_id  TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id   TEXT NOT NULL REFERENCES users(id),
    content     TEXT NOT NULL,
    type        TEXT DEFAULT 'text' CHECK(type IN ('text', 'file', 'system')),
    edited_at   DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS attachments (
    id          TEXT PRIMARY KEY,
    message_id  TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename    TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL,
    mime_type   TEXT NOT NULL,
    hash        TEXT NOT NULL,  -- SHA-256 for integrity
    local_path  TEXT,           -- Cached file path
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS server_members (
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT DEFAULT 'member' CHECK(role IN ('owner', 'admin', 'moderator', 'member')),
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (server_id, user_id)
);

-- Indexes for query performance
CREATE INDEX idx_messages_channel_created ON messages(channel_id, created_at DESC);
CREATE INDEX idx_messages_author ON messages(author_id);
CREATE INDEX idx_channels_server ON channels(server_id);
CREATE INDEX idx_server_members_user ON server_members(user_id);
CREATE INDEX idx_servers_invite ON servers(invite_code);
```

### Server-Side (PostgreSQL)

Same schema as above plus:

```sql
-- Additional server-side tables
CREATE TABLE IF NOT EXISTS sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    ip_address  INET,
    user_agent  TEXT,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   TEXT REFERENCES servers(id),
    actor_id    TEXT NOT NULL REFERENCES users(id),
    action      TEXT NOT NULL,
    target_type TEXT,
    target_id   TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_audit_server_created ON audit_log(server_id, created_at DESC);
```

### Cache Strategy

**Client-Side (In-Memory LRU):**
```go
// internal/store/cache/lru.go
type LRUCache struct {
    maxEntries int           // Default: 10000
    ttl        time.Duration // Default: 5 minutes
    mu         sync.RWMutex
    items      map[string]*entry
    evictList  *list.List
}
```

Cached items:
- Recent messages per channel (last 100)
- User profiles (avatar, username)
- Server metadata
- Channel lists

**Server-Side (Redis):**
- Online user presence (SET with TTL)
- Rate limiting counters (INCR + EXPIRE)
- Session tokens
- Pub/Sub for real-time events across server instances

---

## 9. Authentication

### GitHub OAuth Flow

```
┌──────────┐     ┌──────────────┐     ┌──────────┐     ┌──────────┐
│  Concord  │     │  Central     │     │  GitHub   │     │  Concord  │
│  Client   │     │  Server      │     │  OAuth    │     │  Client   │
└─────┬─────┘     └──────┬───────┘     └─────┬─────┘     └─────┬─────┘
      │                   │                   │                  │
      │  1. Click Login   │                   │                  │
      ├──────────────────►│                   │                  │
      │                   │                   │                  │
      │  2. Auth URL +    │                   │                  │
      │     state token   │                   │                  │
      │◄──────────────────┤                   │                  │
      │                   │                   │                  │
      │  3. Open browser  │                   │                  │
      ├───────────────────┼──────────────────►│                  │
      │                   │                   │                  │
      │                   │  4. Callback with  │                  │
      │                   │     auth code      │                  │
      │                   │◄──────────────────┤                  │
      │                   │                   │                  │
      │                   │  5. Exchange for   │                  │
      │                   │     access token   │                  │
      │                   ├──────────────────►│                  │
      │                   │◄──────────────────┤                  │
      │                   │                   │                  │
      │  6. JWT token +   │                   │                  │
      │     user profile  │                   │                  │
      │◄──────────────────┤                   │                  │
      │                   │                   │                  │
```

### JWT Token Structure

```go
type Claims struct {
    UserID    string `json:"uid"`
    GitHubID  int64  `json:"gid"`
    Username  string `json:"usr"`
    jwt.RegisteredClaims
}

// Access token: 15 minutes
// Refresh token: 30 days (stored encrypted in SQLite)
```

### Device Flow (Alternative for Desktop)
Use GitHub's **Device Authorization Grant** (RFC 8628) via `github.com/cli/oauth`:
1. App requests device code from GitHub
2. Shows user a URL + code to enter
3. Polls GitHub until user authorizes
4. Receives access token

This avoids the need for a local HTTP callback server.

---

## 10. Security Model

### Threat Model & Mitigations

| Threat | CVE Category | Mitigation |
|---|---|---|
| SQL Injection | CWE-89 | Parameterized queries only, no string concat |
| XSS in UI | CWE-79 | Svelte auto-escapes, CSP headers, DOMPurify for rich content |
| MITM on P2P | CWE-300 | TLS 1.3 on QUIC, DTLS on WebRTC, E2EE for messages |
| Token Theft | CWE-522 | JWT in memory only, refresh token encrypted at rest (AES-256-GCM) |
| File Upload Attacks | CWE-434 | File type validation, size limits (50MB), hash verification |
| DoS via Voice | CWE-400 | Rate limiting, max peers per channel (25), bandwidth caps |
| Data Leakage | CWE-200 | SQLite encryption (sqlcipher or manual AES), zero-log policy for voice |
| Privilege Escalation | CWE-269 | RBAC with server-side permission checks, never trust client |
| Dependency Vulns | CWE-1395 | Dependabot, `govulncheck`, Snyk CI |
| Buffer Overflow (Audio) | CWE-120 | Bounds checking on all audio buffers, Go's memory safety |

### End-to-End Encryption (E2EE)

```go
// pkg/crypto/e2ee.go

// Each server generates a shared secret (X25519 key exchange)
// Messages encrypted with AES-256-GCM before transmission
// Voice: DTLS already provides encryption, optional double-encrypt for paranoid mode

type E2EEManager struct {
    privateKey ed25519.PrivateKey
    peerKeys   map[string]ed25519.PublicKey // peerID -> publicKey
    sessionKey []byte                       // Derived via X25519 + HKDF
}

func (m *E2EEManager) Encrypt(plaintext []byte) ([]byte, error)
func (m *E2EEManager) Decrypt(ciphertext []byte) ([]byte, error)
```

### Rate Limiting

```go
// Per-user limits:
// - Messages: 10/second, 100/minute
// - File uploads: 5/minute, 500MB/hour
// - Voice channels: 1 simultaneous connection
// - API requests: 60/minute (general)
// - Server creation: 10/day
```

### Input Validation
- All user input sanitized before storage
- File names sanitized (path traversal prevention)
- Message content: max 4000 chars, UTF-8 validated
- Username: alphanumeric + limited special chars, 2-32 chars

---

## 11. Frontend Architecture & Design System

### Design System: "Void"

**Philosophy:** Dark-first, gaming-oriented, minimal but powerful. Inspired by the void — deep, immersive, and distraction-free.

**Color Palette:**

```css
:root {
  /* Core */
  --void-bg-primary:    #0a0a0f;    /* Deep void black */
  --void-bg-secondary:  #12121a;    /* Slightly lighter */
  --void-bg-tertiary:   #1a1a28;    /* Card backgrounds */
  --void-bg-hover:      #22223a;    /* Hover states */

  /* Accent */
  --void-accent:        #7c3aed;    /* Primary purple */
  --void-accent-hover:  #8b5cf6;    /* Lighter purple */
  --void-accent-glow:   rgba(124, 58, 237, 0.3); /* Glow effect */

  /* Text */
  --void-text-primary:  #e4e4e7;    /* Primary text */
  --void-text-secondary: #a1a1aa;   /* Secondary text */
  --void-text-muted:    #52525b;    /* Muted text */

  /* Status */
  --void-online:        #22c55e;    /* Green */
  --void-idle:          #f59e0b;    /* Amber */
  --void-dnd:           #ef4444;    /* Red */
  --void-offline:       #52525b;    /* Gray */

  /* Borders */
  --void-border:        #27272a;
  --void-border-active: #7c3aed;

  /* Radius */
  --void-radius-sm:     6px;
  --void-radius-md:     10px;
  --void-radius-lg:     16px;
  --void-radius-full:   9999px;

  /* Shadows */
  --void-shadow-sm:     0 1px 2px rgba(0, 0, 0, 0.5);
  --void-shadow-md:     0 4px 12px rgba(0, 0, 0, 0.4);
  --void-shadow-glow:   0 0 20px var(--void-accent-glow);
}
```

**Typography:**
- **Headings:** Inter (700 weight) — clean, modern, highly legible
- **Body:** Inter (400/500 weight) — consistent across OS
- **Monospace:** JetBrains Mono — for code snippets, IDs

**Component Library:**
All components in `frontend/src/lib/components/ui/` follow these patterns:
- Slot-based composition
- Props for variants (size, color, disabled)
- CSS custom properties for theming
- Accessible (ARIA attributes, keyboard navigation)
- Animated with CSS transitions (60fps)

### Layout Structure

```
┌─────────────────────────────────────────────────┐
│ ┌─────┐ ┌──────────┐ ┌───────────────────┐ ┌──┐│
│ │     │ │          │ │                   │ │  ││
│ │ S   │ │ Channel  │ │   Main Content    │ │ M││
│ │ e   │ │ List     │ │   (Chat/Voice)    │ │ e││
│ │ r   │ │          │ │                   │ │ m││
│ │ v   │ │ #general │ │  ┌─────────────┐  │ │ b││
│ │ e   │ │ #gaming  │ │  │ Messages    │  │ │ e││
│ │ r   │ │ 🔊voice-1│ │  │             │  │ │ r││
│ │     │ │ 🔊voice-2│ │  │             │  │ │ s││
│ │ L   │ │          │ │  └─────────────┘  │ │  ││
│ │ i   │ │          │ │  ┌─────────────┐  │ │  ││
│ │ s   │ │          │ │  │ Input Bar   │  │ │  ││
│ │ t   │ │          │ │  └─────────────┘  │ │  ││
│ │     │ │          │ │                   │ │  ││
│ ├─────┤ ├──────────┤ ├───────────────────┤ └──┘│
│ │User │ │Voice Ctrl│ │                   │     │
│ │Panel│ │🎤 🔇 ⚙️  │ │                   │     │
│ └─────┘ └──────────┘ └───────────────────┘     │
└─────────────────────────────────────────────────┘
```

### State Management

Using Svelte 5 **runes** ($state, $derived, $effect) + custom stores:

```typescript
// frontend/src/lib/stores/voice.ts
import { writable, derived } from 'svelte/store';

interface VoiceState {
  connected: boolean;
  channelId: string | null;
  muted: boolean;
  deafened: boolean;
  translationEnabled: boolean;
  translationLang: string;
  activeSpeakers: Map<string, SpeakerInfo>;
  inputDevice: string;
  outputDevice: string;
  inputVolume: number;
  outputVolume: number;
}

export const voiceState = writable<VoiceState>({
  connected: false,
  channelId: null,
  muted: false,
  deafened: false,
  translationEnabled: false,
  translationLang: 'en',
  activeSpeakers: new Map(),
  inputDevice: 'default',
  outputDevice: 'default',
  inputVolume: 100,
  outputVolume: 100,
});
```

---

## 12. Observability & Logging

### Logging (zerolog)

```go
// internal/observability/logger.go
func NewLogger(service string) zerolog.Logger {
    return zerolog.New(os.Stderr).
        With().
        Timestamp().
        Str("service", service).
        Str("version", version.Get()).
        Logger()
}

// Usage in every service:
logger.Info().
    Str("channel_id", channelID).
    Str("user_id", userID).
    Str("action", "join_voice").
    Dur("latency", latency).
    Msg("user joined voice channel")
```

### Structured Log Format
```json
{
  "level": "info",
  "service": "voice",
  "version": "0.1.0",
  "channel_id": "ch_abc123",
  "user_id": "usr_xyz789",
  "action": "join_voice",
  "latency": 42,
  "time": "2026-02-20T10:30:00Z",
  "message": "user joined voice channel"
}
```

### Log Levels by Service
| Service | Debug | Info | Warn | Error |
|---|---|---|---|---|
| Auth | Token refresh | Login/logout | Invalid token | OAuth failure |
| Chat | Message parse | Send/receive | Rate limited | DB write fail |
| Voice | Audio frames | Join/leave | High latency | Connection lost |
| P2P | Handshake detail | Connect/disconnect | NAT traverse fail | Host crash |
| Files | Chunk transfer | Upload/download | Size limit | Corruption |
| Translation | API frames | Enable/disable | High latency | API error |

### Metrics (Prometheus)

```go
// internal/observability/metrics.go
var (
    VoiceChannelUsers = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{Name: "concord_voice_channel_users"},
        []string{"channel_id"},
    )
    MessagesSent = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "concord_messages_sent_total"},
        []string{"server_id"},
    )
    P2PConnectionType = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "concord_p2p_connection_type"},
        []string{"type"}, // "direct", "hole_punch", "relay"
    )
    VoiceLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "concord_voice_latency_ms",
            Buckets: []float64{10, 25, 50, 100, 200, 500},
        },
        []string{"channel_id"},
    )
    TranslationLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "concord_translation_latency_ms",
            Buckets: []float64{50, 100, 170, 250, 500, 1000},
        },
        []string{"lang_pair"},
    )
)
```

### Health Checks

```go
// internal/observability/health.go
type HealthChecker struct {
    checks map[string]func() error
}

// Registered checks:
// - "sqlite"      → db.Ping()
// - "p2p_host"    → host.Status()
// - "webrtc"      → rtc.ConnectionState()
// - "signaling"   → ws.ReadyState()
// - "redis"       → rdb.Ping() (server only)
// - "postgres"    → db.Ping() (server only)
```

---

## 13. Testing Strategy

### Testing Pyramid

```
         ╱  ╲
        ╱ E2E ╲          ~10% — Playwright (critical flows)
       ╱────────╲
      ╱Integration╲      ~30% — Go test + testcontainers
     ╱──────────────╲
    ╱   Unit Tests    ╲   ~60% — Go test + vitest
   ╱────────────────────╲
```

### Unit Tests (Go)

```go
// internal/chat/chat_test.go
func TestMessageService_Send(t *testing.T) {
    repo := mocks.NewMockMessageRepository(t)
    svc := chat.NewService(repo, logger)

    repo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(m *chat.Message) bool {
        return m.Content == "hello" && m.AuthorID == "usr1"
    })).Return(nil)

    msg, err := svc.Send(context.Background(), "usr1", "ch1", "hello")
    assert.NoError(t, err)
    assert.Equal(t, "hello", msg.Content)
}
```

### Integration Tests (Go)

```go
// internal/network/p2p/p2p_test.go
func TestP2P_DirectConnection(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    host1, _ := NewHost(Config{Port: 0})
    host2, _ := NewHost(Config{Port: 0})
    defer host1.Stop()
    defer host2.Stop()

    err := host1.Connect(context.Background(), host2.ID())
    assert.NoError(t, err)

    received := make(chan []byte, 1)
    host2.OnMessage(func(peerID string, data []byte) {
        received <- data
    })

    host1.SendData(context.Background(), host2.ID(), []byte("ping"))
    assert.Equal(t, []byte("ping"), <-received)
}
```

### E2E Tests (Playwright)

```typescript
// frontend/tests/e2e/login.spec.ts
test('user can login with GitHub', async ({ page }) => {
  await page.goto('/');
  await page.click('[data-testid="github-login-btn"]');
  // Mock GitHub OAuth callback
  await page.waitForSelector('[data-testid="server-list"]');
  expect(await page.textContent('[data-testid="username"]')).toBeTruthy();
});
```

### Coverage Targets
- **Unit:** ≥ 80% line coverage
- **Integration:** All critical paths (auth, P2P connect, message send/receive, voice join/leave)
- **E2E:** Login, send message, join voice, create server, upload file

---

## 14. Phased Build Plan

### Phase 1: Foundation (Weeks 1-2)

| Sub-Phase | Description | Files |
|---|---|---|
| 1.1 | Project scaffolding: Wails init, Go modules, frontend setup (Svelte 5 + Vite + Tailwind) | `go.mod`, `wails.json`, `frontend/package.json`, `Makefile` |
| 1.2 | Design system "Void": color tokens, typography, base components (Button, Input, Modal) | `frontend/src/lib/components/ui/*`, `frontend/src/app.css` |
| 1.3 | Configuration system: env loading, defaults, validation | `internal/config/*` |
| 1.4 | Logging & observability setup: zerolog, metrics stubs | `internal/observability/*` |
| 1.5 | SQLite setup + migrations runner | `internal/store/sqlite/*` |
| 1.6 | Basic Wails window with Void theme, layout shell | `cmd/concord/main.go`, `frontend/src/App.svelte` |

**Deliverable:** Running desktop app with themed UI shell and working database.

---

### Phase 2: Authentication (Week 3)

| Sub-Phase | Description | Files |
|---|---|---|
| 2.1 | GitHub OAuth device flow implementation | `internal/auth/github.go` |
| 2.2 | JWT token generation and validation | `internal/auth/jwt.go` |
| 2.3 | Auth middleware for Go services | `internal/auth/middleware.go` |
| 2.4 | Login screen UI (GitHubButton, loading states, error handling) | `frontend/src/lib/components/auth/*`, `frontend/src/pages/Login.svelte` |
| 2.5 | Auth store + token persistence (encrypted in SQLite) | `frontend/src/lib/stores/auth.ts` |
| 2.6 | Auth tests (unit + integration) | `internal/auth/auth_test.go` |

**Deliverable:** Users can log in with GitHub and see their profile.

---

### Phase 3: Server Management (Weeks 4-5)

| Sub-Phase | Description | Files |
|---|---|---|
| 3.1 | Server CRUD (create, read, update, delete) | `internal/server/service.go`, `internal/server/repository.go` |
| 3.2 | Channel management (text + voice channels) | `internal/server/models.go` (Channel model) |
| 3.3 | Member management + RBAC permissions | `internal/server/permissions.go` |
| 3.4 | Invite system (generate/redeem invite codes) | `internal/server/service.go` (InviteCode methods) |
| 3.5 | Server UI: ServerList, ServerCard, CreateServer, ChannelList, MemberList | `frontend/src/lib/components/server/*` |
| 3.6 | Server store + Wails bindings | `frontend/src/lib/stores/servers.ts` |
| 3.7 | Server tests | `internal/server/server_test.go` |

**Deliverable:** Users can create servers, channels, invite others, manage roles.

---

### Phase 4: Text Chat (Weeks 5-6)

| Sub-Phase | Description | Files |
|---|---|---|
| 4.1 | Message service (send, receive, edit, delete) | `internal/chat/service.go`, `internal/chat/repository.go` |
| 4.2 | WebSocket connection for real-time messaging | `internal/network/signaling/client.go` |
| 4.3 | Message persistence (SQLite) with pagination | `internal/chat/repository.go` |
| 4.4 | Chat UI: MessageList, MessageInput, MessageBubble | `frontend/src/lib/components/chat/*` |
| 4.5 | Chat store + real-time updates | `frontend/src/lib/stores/chat.ts` |
| 4.6 | Message search (FTS5 in SQLite) | `internal/chat/repository.go` |
| 4.7 | Chat tests | `internal/chat/chat_test.go` |

**Deliverable:** Real-time text chat within server channels.

---

### Phase 5: P2P Networking (Weeks 7-9)

| Sub-Phase | Description | Files |
|---|---|---|
| 5.1 | libp2p host setup with QUIC transport | `internal/network/p2p/host.go`, `internal/network/transport/quic.go` |
| 5.2 | Peer discovery (mDNS for LAN, DHT for internet) | `internal/network/p2p/discovery.go` |
| 5.3 | NAT traversal + hole punching (DCUtR) | `internal/network/p2p/nat.go` |
| 5.4 | Relay fallback (circuit relay v2) | `internal/network/p2p/relay.go` |
| 5.5 | Signaling server (WebSocket-based) | `internal/network/signaling/server.go` |
| 5.6 | E2EE implementation (X25519 + AES-256-GCM) | `pkg/crypto/e2ee.go`, `pkg/crypto/keys.go` |
| 5.7 | Wire protocol definition (protobuf or msgpack) | `pkg/protocol/messages.go` |
| 5.8 | P2P integration tests | `internal/network/p2p/p2p_test.go` |

**Deliverable:** Peers can connect directly (P2P) or via relay, encrypted.

---

### Phase 6: Voice Chat (Weeks 10-12)

| Sub-Phase | Description | Files |
|---|---|---|
| 6.1 | Audio capture/playback using OS audio APIs | `internal/voice/capture.go`, `internal/voice/playback.go` |
| 6.2 | Opus encode/decode with Pion | `internal/voice/codec.go` |
| 6.3 | Pion WebRTC setup for media streams | `internal/network/transport/webrtc.go` |
| 6.4 | Jitter buffer (adaptive) | `internal/voice/jitter.go` |
| 6.5 | Audio mixer (multi-peer mixing) | `internal/voice/mixer.go` |
| 6.6 | Voice Activity Detection (silence suppression) | `internal/voice/capture.go` (VAD) |
| 6.7 | Noise suppression via RNNoise WASM | `frontend/src/lib/services/audio.ts` |
| 6.8 | Voice UI: VoiceChannel, VoiceControls, VoiceIndicator | `frontend/src/lib/components/voice/*` |
| 6.9 | Voice store + active speaker detection | `frontend/src/lib/stores/voice.ts` |
| 6.10 | Voice engine integration tests | `internal/voice/voice_test.go` |

**Deliverable:** Working voice chat with noise suppression and P2P connectivity.

---

### Phase 7: File Sharing (Week 13)

| Sub-Phase | Description | Files |
|---|---|---|
| 7.1 | File service (upload, download, chunking) | `internal/files/service.go`, `internal/files/chunker.go` |
| 7.2 | File storage abstraction (local + future S3) | `internal/files/storage.go` |
| 7.3 | P2P file transfer via libp2p streams | `internal/files/service.go` |
| 7.4 | File validation (type check, size limit, hash) | `internal/files/scanner.go` |
| 7.5 | File attachment UI in chat | `frontend/src/lib/components/chat/FileAttachment.svelte` |
| 7.6 | File tests | `internal/files/files_test.go` |

**Deliverable:** Users can share files in chat, transferred P2P.

---

### Phase 8: Voice Translation Power-Up (Weeks 14-15)

| Sub-Phase | Description | Files |
|---|---|---|
| 8.1 | PersonaPlex API client (HTTP + WebSocket streaming) | `internal/translation/personaplex.go` |
| 8.2 | Translation streaming pipeline (audio → translate → inject) | `internal/translation/stream.go` |
| 8.3 | Translation cache (common phrases) | `internal/translation/cache.go` |
| 8.4 | Translation UI: TranslationToggle, language selector | `frontend/src/lib/components/voice/TranslationToggle.svelte` |
| 8.5 | Translation integration tests | `internal/translation/translation_test.go` |

**Deliverable:** Users can enable real-time voice translation in voice channels.

---

### Phase 9: Central Server (Weeks 16-17)

| Sub-Phase | Description | Files |
|---|---|---|
| 9.1 | HTTP API server (chi router) | `cmd/server/main.go` |
| 9.2 | PostgreSQL setup + migrations | `internal/store/postgres/*` |
| 9.3 | Redis setup (sessions, presence, pub/sub) | `internal/store/cache/redis.go` |
| 9.4 | WebSocket signaling server | `internal/network/signaling/server.go` |
| 9.5 | TURN server setup (coturn config or embedded) | `deployments/docker/Dockerfile.relay` |
| 9.6 | Offline message queue + sync | `internal/chat/service.go` (sync methods) |
| 9.7 | Docker compose for full server stack | `deployments/docker/docker-compose.yml` |
| 9.8 | Server API tests | `cmd/server/*_test.go` |

**Deliverable:** Central server running with full API, signaling, and relay.

---

### Phase 10: Polish & Hardening (Weeks 18-19)

| Sub-Phase | Description | Files |
|---|---|---|
| 10.1 | Security audit: rate limiting, input validation, CSP | `internal/security/*` |
| 10.2 | Performance profiling + optimization | All services |
| 10.3 | E2E tests with Playwright | `frontend/tests/e2e/*` |
| 10.4 | Error handling improvements (user-friendly messages) | All UI components |
| 10.5 | Settings panel (audio devices, theme, notifications) | `frontend/src/pages/Settings.svelte` |
| 10.6 | Desktop notifications | `frontend/src/lib/services/notifications.ts` |
| 10.7 | Documentation (API.md, P2P-PROTOCOL.md, SECURITY.md, CONTRIBUTING.md) | `docs/*` |
| 10.8 | CHANGELOG.md + version tagging | `CHANGELOG.md` |
| 10.9 | CI/CD pipeline (GitHub Actions) | `.github/workflows/*` |
| 10.10 | Release builds (Windows/Mac/Linux via Wails) | `scripts/build.sh`, `.github/workflows/release.yml` |

**Deliverable:** Production-ready v1.0.0 release.

---

## 15. Big O Complexity Analysis

| Operation | Target | Implementation |
|---|---|---|
| Send message | O(1) | Direct insert + broadcast |
| Load messages (paginated) | O(log n) | B-tree index on (channel_id, created_at) |
| Search messages | O(log n) | FTS5 full-text index |
| Find peer (P2P) | O(log n) | DHT-based discovery (Kademlia in libp2p) |
| Voice mixing (k peers) | O(k) | Linear mix of k audio streams per frame |
| Server member lookup | O(1) | Hash-based member map in memory |
| LRU cache get/put | O(1) | HashMap + doubly-linked list |
| File chunk transfer | O(n/c) | n = file size, c = chunk size, parallelized |
| NAT traversal | O(1) | Single hole-punch attempt per peer |
| Message encryption | O(n) | AES-256-GCM, n = message length |

### Performance Targets
- **Voice latency:** < 100ms (P2P), < 200ms (relay)
- **Message delivery:** < 50ms (P2P), < 150ms (server relay)
- **App startup:** < 2 seconds
- **Memory usage:** < 150MB idle, < 300MB in voice
- **Binary size:** < 30MB

---

## 16. Risks & Edge Cases

| Risk | Impact | Mitigation |
|---|---|---|
| Symmetric NAT (cannot hole punch) | Users can't connect P2P | Automatic fallback to relay server; detect NAT type early |
| PersonaPlex API downtime | Translation unavailable | Graceful degradation; disable feature with toast notification |
| PersonaPlex latency spike (>500ms) | Voice becomes unusable | Circuit breaker pattern; auto-disable if latency exceeds threshold |
| Large voice channels (>25 users) | CPU/bandwidth explosion | SFU mode for large channels (route through server, not P2P mesh) |
| File transfer interruption | Partial files | Resumable chunks, hash verification per chunk |
| SQLite concurrent writes | Data corruption | WAL mode, single-writer mutex, retry with backoff |
| WebRTC ICE failure | No voice connection | Multiple STUN servers, TURN fallback, user notification |
| GitHub OAuth rate limiting | Login failures | Cache tokens aggressively, refresh before expiry |
| Audio device changes (hot-plug) | Audio stream breaks | OS event listener, auto-reconnect to new device |
| Memory leaks in voice (long sessions) | App crash | Periodic goroutine/memory profiling, pool audio buffers |
| Cross-platform audio API differences | Inconsistent behavior | Abstract audio layer, platform-specific implementations |
| Wails webview rendering differences | UI inconsistency | Test on Windows (Edge WebView2), Mac (WebKit), Linux (WebKitGTK) |

---

## 17. Files to Create

### Phase 1 Initial Files (Priority Order)

1. `go.mod` — Go module definition
2. `wails.json` — Wails project configuration
3. `Makefile` — Build, dev, test commands
4. `cmd/concord/main.go` — Desktop app entry point
5. `cmd/server/main.go` — Server entry point
6. `internal/config/config.go` — Configuration
7. `internal/config/defaults.go` — Default values
8. `internal/observability/logger.go` — Logging setup
9. `internal/observability/metrics.go` — Metrics setup
10. `internal/store/sqlite/sqlite.go` — SQLite connection
11. `internal/store/sqlite/migrations/001_init.sql` — Initial schema
12. `frontend/package.json` — Frontend dependencies
13. `frontend/vite.config.ts` — Vite configuration
14. `frontend/svelte.config.js` — Svelte configuration
15. `frontend/tailwind.config.ts` — Tailwind configuration
16. `frontend/src/main.ts` — Frontend entry point
17. `frontend/src/App.svelte` — Root component
18. `frontend/src/app.css` — Global styles + Void design tokens
19. `frontend/src/lib/components/ui/Button.svelte` — First UI component
20. `CHANGELOG.md` — Changelog
21. `LICENSE` — License file

---

## 18. Recommended Agents

| Phase | Agent | Reason |
|---|---|---|
| 1 (Foundation) | **backend_dev** | Go scaffolding, config, DB, logging |
| 1.2 (Design System) | **frontend_dev** | Svelte components, CSS, Tailwind |
| 2 (Auth) | **backend_dev** | OAuth flow, JWT, crypto — security-critical |
| 3 (Servers) | **both** | Go services + Svelte UI simultaneously |
| 4 (Chat) | **both** | Go WebSocket + Svelte chat UI |
| 5 (P2P) | **backend_dev** | Heavy Go networking (libp2p, crypto) |
| 6 (Voice) | **both** | Go audio engine + Svelte voice UI |
| 7 (Files) | **both** | Go file service + Svelte file UI |
| 8 (Translation) | **backend_dev** | API integration, streaming pipeline |
| 9 (Server) | **backend_dev** | PostgreSQL, Redis, Docker, API |
| 10 (Polish) | **both** | Security, E2E tests, documentation |

---

## Appendix A: Wire Protocol (Message Types)

```go
// pkg/protocol/messages.go
type MessageType uint8

const (
    TypeTextMessage    MessageType = 0x01
    TypeTextEdit       MessageType = 0x02
    TypeTextDelete     MessageType = 0x03
    TypeVoiceJoin      MessageType = 0x10
    TypeVoiceLeave     MessageType = 0x11
    TypeVoiceData      MessageType = 0x12
    TypeVoiceMute      MessageType = 0x13
    TypeFileOffer      MessageType = 0x20
    TypeFileAccept     MessageType = 0x21
    TypeFileChunk      MessageType = 0x22
    TypeFileComplete   MessageType = 0x23
    TypeServerSync     MessageType = 0x30
    TypePresenceUpdate MessageType = 0x31
    TypeTypingStart    MessageType = 0x32
    TypeTypingStop     MessageType = 0x33
    TypePing           MessageType = 0xFE
    TypePong           MessageType = 0xFF
)

// Wire format: [1 byte type][4 bytes length][payload (msgpack)]
```

## Appendix B: Key Go Interfaces Summary

```go
// All services follow this pattern:
type Service interface {
    // Business logic methods
}

type Repository interface {
    // Data access methods
}

type Handler interface {
    // Wails-bound methods exposed to frontend
}

// This enables:
// 1. Easy mocking in tests
// 2. Swappable implementations (SQLite ↔ PostgreSQL)
// 3. Clean dependency injection
```

---

*This architecture document is the single source of truth for the Concord project. All implementation must follow these patterns and decisions. Any deviation requires updating this document first.*
