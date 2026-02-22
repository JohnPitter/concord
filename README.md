<p align="center">
  <img src="frontend/src/assets/logo.png" alt="Concord" width="120" height="120" />
</p>

<h1 align="center">Concord</h1>

<p align="center">
  <strong>Privacy. Freedom. Friendship.</strong><br/>
  Comunicacao para gamers sem coleta de dados. Open-source, peer-to-peer, criptografado.
</p>

<p align="center">
  <a href="https://github.com/JohnPitter/concord/actions/workflows/ci.yml"><img src="https://github.com/JohnPitter/concord/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/JohnPitter/concord/releases/latest"><img src="https://img.shields.io/github/v/release/JohnPitter/concord?style=flat-square&color=16a34a" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="MIT License"></a>
</p>

<p align="center">
  <a href="https://johnpitter.github.io/concord/">Site</a> ·
  <a href="https://github.com/JohnPitter/concord/releases/latest">Download</a> ·
  <a href="ARCHITECTURE.md">Arquitetura</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>

---

## Por que Concord?

> **Concord** e o oposto de **Discord**. Simples assim.
>
> Enquanto Discord significa *discordia*, **Concord** significa *concordia, harmonia*.
>
> Este projeto nasceu como resposta a imposicao de coleta de dados biometricos para verificacao de idade. Ninguem deveria entregar scans faciais ou documentos de identidade apenas para conversar com amigos.
>
> **Sem biometria. Sem rastreamento. Sem scam.**

---

## Download

| Plataforma | Download |
|------------|----------|
| Windows (x64) | [concord-windows-amd64.zip](https://github.com/JohnPitter/concord/releases/latest) |
| macOS (Apple Silicon) | [concord-macos-arm64.zip](https://github.com/JohnPitter/concord/releases/latest) |
| Linux (x64) | [concord-linux-amd64.tar.gz](https://github.com/JohnPitter/concord/releases/latest) |

> Servidor central tambem disponivel nos [releases](https://github.com/JohnPitter/concord/releases/latest) (Linux, macOS, Windows).

---

## Funcionalidades

- **Chat de texto** — mensagens em tempo real, busca full-text, historico
- **Chat de voz** — peer-to-peer, codec Opus, deteccao de voz, baixa latencia
- **Compartilhamento de tela** — transmissao direta entre usuarios
- **Servidores e canais** — crie servidores, organize canais de texto e voz, gerencie membros
- **Modo P2P** — conexao direta sem servidor central, via mDNS na rede local ou pela internet
- **Modo Servidor** — servidor central com PostgreSQL e Redis para escala
- **Traducao em tempo real** — traduza mensagens entre 11 idiomas
- **Login via GitHub** — OAuth Device Flow, sem senhas
- **Convites** — codigos de convite com expiracao para servidores
- **Temas** — modo escuro e modo claro

---

## Como funciona

```
┌───────────────────────────────────────────────┐
│            CONCORD DESKTOP APP                │
│                                               │
│  Go Backend (Wails)  ◄──►  Svelte 5 Frontend │
│  ┌────────────────┐       ┌────────────────┐  │
│  │ Auth           │       │ Design System  │  │
│  │ Chat           │       │ Voice Controls │  │
│  │ Voice (WebRTC) │       │ Chat UI        │  │
│  │ P2P (libp2p)   │       │ Server Browser │  │
│  │ Translation    │       │ Settings       │  │
│  └───────┬────────┘       └────────────────┘  │
│          │                                    │
│  ┌───────▼────────┐                           │
│  │ Networking      │                           │
│  │ libp2p + Pion   │                           │
│  └───────┬────────┘                           │
└──────────┼────────────────────────────────────┘
           │
   ┌───────▼────────┐
   │ Internet / LAN  │
   │ Signaling       │
   │ Relay           │
   └─────────────────┘
```

---

## Compilar do fonte

### Pre-requisitos

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Build

```bash
git clone https://github.com/JohnPitter/concord.git
cd concord

# Instalar dependencias
go mod download
cd frontend && npm install && cd ..

# Rodar em modo desenvolvimento (hot reload)
wails dev

# Build desktop app
wails build -clean

# Build servidor central
CGO_ENABLED=0 go build -o concord-server ./cmd/server
```

### Testes

```bash
go test -short ./...          # Testes rapidos
go test -v -race ./...        # Todos com race detection
```

### Lint

```bash
golangci-lint run ./...       # Go lint (v2)
go vet ./...                  # Go vet
```

---

## Estrutura do projeto

```
concord/
├── cmd/server/          # Servidor central (PostgreSQL + REST API)
├── internal/
│   ├── api/             # HTTP handlers + middleware (chi v5)
│   ├── auth/            # GitHub OAuth + JWT
│   ├── chat/            # Mensagens + busca
│   ├── config/          # Configuracao (JSON + env vars)
│   ├── network/         # P2P (libp2p) + Signaling (WebSocket)
│   ├── observability/   # Logging (zerolog) + Metrics (Prometheus)
│   ├── security/        # Crypto, rate limiting, validacao
│   ├── server/          # Servidores, canais, membros, convites
│   ├── store/           # SQLite (client) + PostgreSQL (server) + Redis
│   ├── voice/           # WebRTC + Opus + VAD + jitter buffer
│   └── translation/     # Traducao de mensagens
├── frontend/            # Svelte 5 + TypeScript + TailwindCSS v4
├── deployments/docker/  # Docker Compose (dev + prod com Nginx)
├── docs/                # Documentacao (SCALING.md, site)
├── main.go              # Entry point desktop (Wails v2)
├── ARCHITECTURE.md      # Especificacao tecnica completa
└── CHANGELOG.md
```

---

## Roadmap

| Fase | Feature | Status |
|------|---------|--------|
| 1 | Foundation (config, logging, SQLite, observability) | Completo |
| 2 | Design System "Void" + Layout Shell | Completo |
| 3 | GitHub OAuth (Device Flow) | Completo |
| 4 | Servidores (CRUD, canais, membros, convites) | Completo |
| 5 | Chat de texto (FTS5, paginacao) | Completo |
| 6 | P2P (libp2p, NAT traversal, signaling) | Completo |
| 7 | Voice (WebRTC, Opus, VAD, jitter buffer) | Completo |
| 8 | UX (temas, traducao, screen share, VAD visual) | Completo |
| 9 | Scaling Fase 1 (metrics, rate limit, Docker prod) | Completo |
| 10 | Traducao de voz (PersonaPlex) | Planejado |

Ver [ARCHITECTURE.md](ARCHITECTURE.md) para o plano completo e [docs/SCALING.md](docs/SCALING.md) para o plano de escala ate 100k usuarios.

---

## Seguranca

- DTLS no WebRTC, TLS 1.3 no QUIC
- Prepared statements (prevencao SQL injection)
- Sanitizacao de dados sensiveis nos logs
- Rate limiting com headers padrao em todos endpoints
- Validacao de uploads (tipo, tamanho, hash)
- JWT com refresh token automatico

Reporte vulnerabilidades abrindo uma issue privada.

---

## Licenca

[MIT](LICENSE) — Copyright (c) 2026 Concord Team
