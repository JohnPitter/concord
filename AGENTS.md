# Repository Guidelines

## Project Structure & Module Organization
Concord is a Go + Wails desktop app with a Svelte frontend.

- `main.go`: desktop entrypoint (Wails runtime + bindings).
- `cmd/server/`: central HTTP/WebSocket server binary.
- `internal/`: core backend domains (`api`, `auth`, `chat`, `network`, `voice`, `store`, etc.).
- `pkg/`: shared reusable packages (e.g., crypto/protocol helpers).
- `frontend/src/`: Svelte 5 UI, stores, API clients, and components.
- `frontend/tests/e2e/`: Playwright end-to-end specs.
- `deployments/docker/`: Dockerfiles, compose files, Nginx config.
- `docs/`: architecture, scaling, API, and operational notes.

## Build, Test, and Development Commands
Run from repository root unless noted.

- `go mod download` and `cd frontend && npm install`: install dependencies.
- `wails dev`: run desktop app in hot-reload development mode.
- `wails build -clean`: build desktop app artifacts.
- `CGO_ENABLED=0 go build -o build/concord-server ./cmd/server`: build server binary.
- `go test -short ./...`: fast backend tests.
- `go test -v -race ./...`: full backend test pass with race detector.
- `cd frontend && npm run build`: production frontend bundle.
- `cd frontend && npm run check`: Svelte + TypeScript checks.

## Coding Style & Naming Conventions
- Go: use `gofmt` formatting; keep packages lower-case and cohesive.
- Svelte/TypeScript: 2-space indentation and strict typing where practical.
- File naming:
  - Go tests: `*_test.go`
  - Svelte components: `PascalCase.svelte`
  - Stores/helpers: descriptive lower-case names (e.g., `chat.svelte.ts`).
- Prefer small, focused functions and explicit error handling in Go.

## Testing Guidelines
- Backend uses Go’s `testing` package (with testify in several modules).
- Frontend E2E uses Playwright (`frontend/tests/e2e`).
- Name tests by behavior (`TestXxx_Yyy`) and cover success + failure paths.
- Before a PR, run at least:
  - `go test ./internal/api ./internal/network/signaling`
  - `go test -short ./...`
  - `cd frontend && npm run check && npm run build`

## Commit & Pull Request Guidelines
- Follow conventional prefixes seen in history: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`.
- Keep commits scoped and atomic (one logical change per commit).
- PRs should include:
  - clear summary and motivation,
  - impacted modules/paths,
  - validation commands executed,
  - screenshots/GIFs for UI changes,
  - linked issue/ticket when applicable.

## Security & Configuration Tips
- Do not commit real secrets; use local config files/env vars.
- Use `config.server.json` as a baseline for server settings.
- For production-like checks, prefer `deployments/docker/docker-compose.prod.yml`.

## Prompt Mestre
1. Arquitetura e código limpo: manter código altamente legível, evitar duplicação e maximizar reuso.
2. Performance orientada por Big O Notation: preferir estruturas e algoritmos com melhor complexidade.
3. Mitigação contra principais CVEs: revisar dependências, hardening e práticas seguras.
4. Resiliência e cache: aplicar fallback, retry/circuit breaker quando aplicável e cache em fluxos críticos.
5. Design moderno baseado no contexto do produto e do usuário.
6. Garantia funcional via pirâmide de testes (unitário, integração, E2E).
7. Segurança contra vazamento de dados: minimizar exposição de dados sensíveis em código, logs e respostas.
8. Observabilidade: aplicar logs nos fluxos relevantes e métricas/tracing quando necessário.
9. Princípios de Design System: consistência visual, componentes reutilizáveis e tokens de interface.
10. Planejamento por fases e subfases antes de mudanças extensas.
11. Documentar alterações no `CHANGELOG.md`.
12. Toda alteração deve manter build funcional e remover imports não utilizados.

## Agente
1. Se um comando demorar demais, cancelar ou converter em tarefa em background.
2. Se uma abordagem falhar, tentar nova estratégia e pesquisar na internet quando necessário.
3. Economia de tokens: foco em implementação, com resumos objetivos.
