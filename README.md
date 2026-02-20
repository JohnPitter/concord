# Concord

<p align="center">
  <strong>Chat de voz para amigos que jogam com máximo de privacidade. No Scam Bro.</strong>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge" alt="MIT License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.24"></a>
  <a href="https://wails.io/"><img src="https://img.shields.io/badge/Wails-v2-red?style=for-the-badge" alt="Wails v2"></a>
  <a href="https://svelte.dev/"><img src="https://img.shields.io/badge/Svelte-5-FF3E00?style=for-the-badge&logo=svelte&logoColor=white" alt="Svelte 5"></a>
</p>

**Concord** is a privacy-first, open-source Discord alternative designed for gamers. Real-time voice chat, text messaging, file sharing, and server management — all running as a native desktop app built with **Go (Wails)** and **Svelte 5**. Voice traffic flows peer-to-peer when possible, through relay servers when NAT traversal fails. A power-up feature enables **real-time voice translation** via NVIDIA PersonaPlex.

[Architecture](ARCHITECTURE.md) · [Changelog](CHANGELOG.md) · [License](LICENSE)

## How it works

```
┌─────────────────────────────────────────────────────┐
│               CONCORD DESKTOP APP                   │
│                                                     │
│   Go Backend (Wails)  ◄──►  Svelte 5 Frontend      │
│   ┌──────────────────┐      ┌──────────────────┐   │
│   │ Auth Service      │      │ Design System    │   │
│   │ Chat Service      │      │ Voice Controls   │   │
│   │ Voice Engine      │      │ Chat Interface   │   │
│   │ P2P Manager       │      │ Server Browser   │   │
│   │ File Service      │      │ Settings Panel   │   │
│   │ Translation Svc   │      │ File Manager     │   │
│   └────────┬─────────┘      └──────────────────┘   │
│            │                                        │
│   ┌────────▼─────────┐                              │
│   │ Networking Layer  │                              │
│   │  libp2p + Pion   │                              │
│   └────────┬─────────┘                              │
└────────────┼────────────────────────────────────────┘
             │
    ┌────────▼─────────┐
    │  Internet / LAN   │
    │  Signaling Server │
    │  Relay Server     │
    │  Auth Server      │
    └───────────────────┘
```

## Key decisions

| Decision | Choice | Why |
|---|---|---|
| Desktop Framework | Wails v2 | Go-native, small binary, OS webview (not Electron) |
| Frontend | Svelte 5 + TypeScript | Reactive, compiled, small bundle |
| P2P Layer | libp2p | NAT traversal, QUIC transport, relay circuits |
| Voice | Pion WebRTC v4 | Pure Go, MIT license, P2P media |
| Audio Codec | Opus | Pure Go, no CGo, royalty-free |
| Database (Client) | SQLite (modernc.org) | Pure Go, no CGo, embedded |
| Database (Server) | PostgreSQL | Scalable, ACID |
| Auth | GitHub OAuth | Single sign-on, no passwords |
| Voice Translation | NVIDIA PersonaPlex | ~170ms latency, full-duplex |
| Logging | zerolog | Zero-allocation structured JSON |

## Install

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)
- [Node.js 20+](https://nodejs.org/)

### From source

```bash
git clone https://github.com/concord-chat/concord.git
cd concord

# Install dependencies
go mod download && go mod tidy
cd frontend && npm install && cd ..

# Run in dev mode (hot reload)
wails dev
```

### Build

```bash
# Desktop app
wails build -clean -upx

# Central server
make build-server
```

## Development

```bash
# Dev server with hot reload
make dev

# Run tests
make test              # all tests
make test-unit         # unit only
make test-integration  # integration only

# Lint & format
make lint
make fmt

# Security scan
make sec
```

Run `make help` for all available targets.

## Project structure

```
concord/
├── cmd/
│   ├── concord/        # Desktop app entry point (Wails)
│   └── server/         # Central server entry point
├── internal/
│   ├── config/         # Configuration (JSON + env overrides)
│   ├── observability/  # Logging, metrics, health checks
│   ├── security/       # Crypto, rate limiting, validation
│   └── store/sqlite/   # SQLite layer + migrations
├── pkg/version/        # Public version info
├── frontend/           # Svelte 5 UI (in progress)
├── ARCHITECTURE.md     # Full technical specification
├── Makefile            # Build automation
└── go.mod
```

## Roadmap

| Phase | Feature | Status |
|---|---|---|
| 1 | Foundation (config, logging, SQLite, observability) | Done |
| 1.2 | Frontend design system "Void" | Pending |
| 2 | GitHub OAuth authentication | Planned |
| 3 | Server management (CRUD, channels, members) | Planned |
| 4 | Real-time text chat (WebSocket) | Planned |
| 5 | P2P networking (libp2p, NAT traversal) | Planned |
| 6 | Voice chat (WebRTC, Opus) | Planned |
| 7 | File sharing | Planned |
| 8 | Voice translation (PersonaPlex) | Planned |
| 9 | Central server (PostgreSQL, Redis, REST) | Planned |
| 10 | Production hardening & release | Planned |

See [ARCHITECTURE.md](ARCHITECTURE.md) Section 14 for the full phased build plan.

## Security

- E2EE with X25519 + AES-256-GCM (planned)
- DTLS on WebRTC, TLS 1.3 on QUIC
- Prepared statements (SQL injection prevention)
- Sensitive data sanitization in logs
- Rate limiting on all endpoints
- File upload validation (type, size, hash)

Report vulnerabilities by opening a private issue.

## License

[MIT](LICENSE) — Copyright (c) 2026 Concord Team
