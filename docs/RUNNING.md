# Como Rodar o Concord

O Concord tem dois modos de execução: **Desktop App** (Wails) e **Central Server** (Docker).

---

## 1. Desktop App (Wails)

O app desktop usa Wails v2 com Svelte 5 + Go.

### Pré-requisitos

- Go 1.25+
- Node.js 20+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Instalar dependências

```bash
go mod download && go mod tidy
cd frontend && npm install && cd ..
```

### Rodar em modo dev (hot reload)

```bash
wails dev
```

O frontend fica disponível em `http://localhost:5173` (mas deve ser acessado via janela Wails).

> **Importante:** Nunca rode `npm run dev` sozinho — os bindings Wails não existem fora do runtime.

### Build para produção

```bash
wails build -clean -upx
```

O executável é gerado em `build/bin/`.

---

## 2. Central Server (Docker)

O servidor central roda Go + PostgreSQL 17 + Redis 7 via Docker Compose.

### Pré-requisitos

- Docker 24+
- Docker Compose v2+

### Setup rápido

```bash
# 1. Copiar template de env vars
cp deployments/docker/.env.example deployments/docker/.env

# 2. Editar com seus valores (OBRIGATÓRIO mudar pelo menos estes):
#    - CONCORD_JWT_SECRET (mín. 32 chars)
#    - POSTGRES_PASSWORD
#    - CONCORD_GITHUB_CLIENT_ID (criar em https://github.com/settings/developers)
nano deployments/docker/.env

# 3. Subir a stack
docker compose -f deployments/docker/docker-compose.yml up -d

# 4. Verificar health
curl http://localhost:8080/health
```

### Variáveis de ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `SERVER_PORT` | `8080` | Porta HTTP do servidor |
| `CONCORD_ENV` | `production` | Ambiente (dev/staging/production) |
| `CONCORD_JWT_SECRET` | — | Secret para tokens JWT (mín. 32 chars) |
| `CONCORD_GITHUB_CLIENT_ID` | — | GitHub OAuth App Client ID |
| `POSTGRES_USER` | `concord` | Usuário do PostgreSQL |
| `POSTGRES_PASSWORD` | — | **Obrigatório.** Senha do PostgreSQL |
| `POSTGRES_DB` | `concord` | Nome do banco |
| `POSTGRES_PORT` | `5432` | Porta do PostgreSQL |
| `REDIS_PASSWORD` | (vazio) | Senha do Redis (opcional) |
| `REDIS_PORT` | `6379` | Porta do Redis |
| `LOG_LEVEL` | `info` | Nível de log (debug/info/warn/error) |

### Endpoints da API

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/api/v1/auth/device-code` | Iniciar GitHub Device Flow |
| `POST` | `/api/v1/auth/token` | Trocar device code por token |
| `POST` | `/api/v1/auth/refresh` | Renovar sessão |
| `GET` | `/api/v1/servers` | Listar servidores do usuário |
| `POST` | `/api/v1/servers` | Criar servidor |
| `GET` | `/api/v1/servers/:id` | Obter servidor |
| `PUT` | `/api/v1/servers/:id` | Atualizar servidor |
| `DELETE` | `/api/v1/servers/:id` | Deletar servidor |
| `GET` | `/api/v1/servers/:id/channels` | Listar canais |
| `POST` | `/api/v1/servers/:id/channels` | Criar canal |
| `GET` | `/api/v1/servers/:id/members` | Listar membros |
| `DELETE` | `/api/v1/servers/:id/members/:uid` | Kick membro |
| `PUT` | `/api/v1/servers/:id/members/:uid/role` | Alterar role |
| `POST` | `/api/v1/servers/:id/invite` | Gerar invite |
| `POST` | `/api/v1/invite/:code/redeem` | Resgatar invite |
| `GET` | `/api/v1/channels/:id/messages` | Listar mensagens |
| `POST` | `/api/v1/channels/:id/messages` | Enviar mensagem |
| `PUT` | `/api/v1/messages/:id` | Editar mensagem |
| `DELETE` | `/api/v1/messages/:id` | Deletar mensagem |
| `GET` | `/api/v1/channels/:id/messages/search` | Buscar mensagens (FTS) |

Rotas protegidas exigem header `Authorization: Bearer <jwt>`.

### Comandos úteis

```bash
# Ver logs do servidor
docker compose -f deployments/docker/docker-compose.yml logs -f server

# Parar tudo
docker compose -f deployments/docker/docker-compose.yml down

# Parar e limpar volumes (CUIDADO: apaga dados)
docker compose -f deployments/docker/docker-compose.yml down -v

# Rebuild após mudanças no código
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

---

## 3. Build do servidor sem Docker

Para rodar o servidor diretamente (sem Docker), você precisa de PostgreSQL e Redis rodando localmente.

```bash
# Build
CGO_ENABLED=0 go build -o concord-server ./cmd/server

# Configurar env vars
export POSTGRES_HOST=localhost
export POSTGRES_PASSWORD=sua-senha
export CONCORD_JWT_SECRET=sua-secret-de-32-chars-minimo-aqui
export CONCORD_GITHUB_CLIENT_ID=seu-client-id

# Rodar
./concord-server
```

O servidor carrega `config.json` (ou `config.server.json`) e aplica env vars por cima.

---

## 4. Testes

```bash
# Todos os testes Go (modo curto, sem integracao)
go test -v -short ./...

# Testes do adapter PostgreSQL
go test -v -short ./internal/store/postgres/...

# Testes da API
go test -v -short ./internal/api/...

# Lint
go vet ./...
```

---

## Arquitetura

```
┌─────────────────┐     ┌──────────────────────────────────┐
│   Desktop App   │     │       Central Server              │
│   (Wails v2)    │     │   (cmd/server/main.go)            │
│                 │     │                                    │
│  Svelte 5 UI   │     │  chi router + JWT middleware       │
│  + SQLite local │     │  + PostgreSQL 17 + Redis 7        │
│  + libp2p P2P   │     │  + Auth/Server/Chat services      │
└─────────────────┘     └──────────────────────────────────┘
```
