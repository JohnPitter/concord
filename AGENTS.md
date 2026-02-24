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

## Princípios Mestres

### 1. Arquitetura e Código Limpo
**Código altamente legível, sem duplicação, com máximo reuso.**

- **Separação de responsabilidades:** routes → services → database (3 camadas)
- **Middleware pipeline:** cors → json → cookie → logger → rateLimiter → auth → routes → errorHandler
- Shared types entre frontend e backend via `@agenthub/shared` — **nunca duplicar interfaces**
- Sem lógica de negócio em routes — delegar para services/managers
- **DRY (Don't Repeat Yourself):** extrair funções utilitárias para `lib/` quando padrão se repete 2+ vezes
- **Single Responsibility:** cada arquivo/função faz UMA coisa bem feita
- Nomes descritivos: `fetchTasksByProject()` > `getData()`, `isTaskCompleted()` > `check()`
- Funções curtas (< 50 linhas); se exceder, extrair sub-funções
- Evitar `any` — usar tipos específicos ou generics; `unknown` quando tipo é realmente desconhecido
- Barrel exports (`index.ts`) em cada package para API pública limpa
- Constantes em `@agenthub/shared` — nunca hardcodar strings/números mágicos no código
- Imports organizados: 1) bibliotecas externas, 2) packages internos, 3) imports relativos

### 2. Performance Baseada na Teoria do Big O Notation
**Toda operação deve ser analisada pela complexidade algorítmica.**

- **O(1) preferível:** usar Maps/Sets para lookups ao invés de `Array.find()` em listas grandes
- **Evitar O(n²):** nunca loops aninhados sobre a mesma coleção; usar index/hash maps
- **Queries SQL otimizadas:** sempre WHERE + índices; nunca `SELECT *` seguido de filtro em memória
- **Paginação obrigatória:** todo endpoint que retorna lista deve aceitar `limit` + `offset`
- **Debounce/Throttle:** eventos de alta frequência (socket, input, resize) com 100-300ms
- **Lazy loading:** componentes pesados (Monaco Editor, Recharts, React Flow) com `React.lazy()`
- **Memoização:** `useMemo`/`useCallback` para cálculos caros e callbacks em listas
- **Batch operations:** agrupar updates de DB quando possível (ex: `INSERT INTO ... VALUES (...), (...)`)
- **Evitar re-renders:** Zustand selectors granulares (`useStore(s => s.field)` ao invés de `useStore()`)
- **Índices no DB:** toda coluna usada em WHERE, JOIN ou ORDER BY deve ter índice

### 3. Mitigação Contra Principais CVEs
**Proteção ativa contra OWASP Top 10 e CVEs conhecidas.**

- **Command Injection (CWE-78):** sempre `execFile` (nunca `exec`/`execSync`); args como array, nunca concatenar
- **Path Traversal (CWE-22):** normalizar paths com `path.resolve()` e validar que resultado está dentro do diretório permitido
- **SQL Injection (CWE-89):** usar Drizzle ORM parameterized queries; nunca interpolar strings em SQL
- **XSS (CWE-79):** usar `react-markdown` para conteúdo dinâmico; nunca injetar HTML raw diretamente no DOM; sanitizar dados do banco antes de renderizar
- **SSRF (CWE-918):** validar URLs fornecidas pelo usuário; whitelist de hosts permitidos
- **Broken Auth (CWE-287):** JWT com expiração curta (24h), refresh token rotation, cookie httpOnly+secure
- **Sensitive Data Exposure (CWE-200):** API nunca retorna campos sensíveis (accessToken, credentials, encryptionKey)
- **Dependency vulnerabilities:** `pnpm audit` antes de cada release; CI com `pnpm audit --audit-level moderate`
- **Rate limiting:** toda rota API protegida contra brute force
- **CORS restrito:** apenas origens explícitas (`localhost:5173`, `localhost:5174`)

### 4. Resiliência dos Serviços e Uso de Cache
**Serviços devem sobreviver a falhas parciais e minimizar operações repetidas.**

- **Retry com backoff exponencial:** operações de rede com 3 tentativas (1s, 2s, 4s delay)
- **Timeout protection:** git local 5s, git remote 60s, API calls 30s, agent execution 5min
- **Circuit breaker pattern:** se serviço externo falha 3x consecutivas, parar de chamar por 30s
- **Graceful degradation:** se git remote falha, operações locais continuam; se WhatsApp desconecta, auto-reconnect
- **Error boundaries (React):** cada rota/página isolada; falha em Analytics não afeta Dashboard
- **Stash automático:** antes de pull/checkout para preservar work-in-progress
- **Cache de dados estáticos:** project list, agent configs — Zustand com TTL (5 min stale)
- **Cache de API:** respostas de analytics/dashboard com header `Cache-Control: max-age=60`
- **Deduplicação de requests:** Zustand stores com flag `loading` para prevenir fetch duplicado
- **Reconnect automático:** Socket.io com backoff; WhatsApp sessions restauradas no startup

### 5. Design Moderno Baseado no Contexto
**Interface profissional, consistente e responsiva.**

- **Paleta principal:** beige page (`#FFFDF7`), warm gray sidebar (`#1C1917`), orange primary (`#F97316`), dark hero gradient
- **Paleta semântica:** green=sucesso, red=erro/danger, yellow=warning, purple=git, blue=info/push
- **Tipografia hierárquica:** 24px números/stats, 14px headers/body, 12px labels, 11px captions
- **Spacing consistente:** 4px base grid (p-1=4px, p-2=8px, p-4=16px, p-6=24px)
- **Cards:** bordas sutis (`border-stroke`), cantos arredondados (`rounded-xl`), sombras suaves
- **Ícones:** exclusivamente Lucide React — nunca misturar icon sets
- **Responsivo:** mobile-first com Tailwind breakpoints (sm, md, lg, xl)
- **Animações:** transições suaves (150-300ms), skeleton loaders para loading states
- **Dark theme:** variáveis CSS preparadas para toggle futuro
- **Acessibilidade:** labels em inputs, alt em imagens, focus-visible em interativos, contraste WCAG AA

### 6. Garantia das Funcionalidades Através da Pirâmide de Testes
**Cada camada da aplicação deve ter cobertura proporcional.**

- **Base — Unit Tests (70%):** services, utils, helpers, pure functions, state machines
  - Framework: Vitest
  - Mock de dependências externas (DB, API, SDK)
  - Cada service novo DEVE ter testes unitários antes de merge
- **Meio — Integration Tests (20%):** API endpoints, middleware, database queries
  - Framework: Vitest + Supertest
  - DB in-memory para isolamento
  - Testar happy path + error cases + edge cases
- **Topo — E2E Tests (10%):** fluxos críticos de usuário
  - Framework: Playwright (planejado)
  - Fluxos: login → create project → create task → agent execution → review → done
- **Coverage mínimo:** 60% geral, 80% para módulos críticos (agent-manager, auth-service, task-watcher)
- **Testes rodam no CI:** `pnpm test` em cada PR via GitHub Actions
- **Naming convention:** `*.test.ts` para unit, `*.integration.test.ts` para integration
- **Princípio:** nenhuma feature é "completa" sem teste que valide o comportamento esperado

### 7. Segurança Contra Vazamento de Dados
**Dados sensíveis nunca devem ser expostos em logs, respostas API, ou código-fonte.**

- **Criptografia em repouso:** `users.accessToken` e `integrations.credentials` criptografados com AES-256-GCM
- **Encryption module:** `apps/orchestrator/src/lib/encryption.ts` — `encrypt()` / `decrypt()`
- **ENCRYPTION_KEY:** obrigatória em produção, derivada de env var, nunca hardcoded
- **`.env` no `.gitignore`:** nunca commitar secrets, API keys, tokens
- **Tokens efêmeros:** injetados em URLs temporariamente com cleanup automático após uso
- **API response sanitization:** nunca retornar campos `accessToken`, `credentials`, `encryptionKey`, `config` com secrets
- **Logs seguros:** nunca logar tokens, senhas, ou dados pessoais decriptados; mascarar com `***`
- **Cookie security:** `httpOnly`, `secure` (prod), `sameSite: strict`, expiração 24h
- **Rate limiting:** proteção contra brute force em endpoints de autenticação (20 req/15min)
- **Segregação de dados:** cada projeto isolado; queries sempre filtram por `projectId`
- **Audit trail:** toda operação de acesso a dados sensíveis registrada em `task_logs`

### 8. Aplicação de Logs em Todos os Fluxos e Conceitos de Observabilidade
**Todo fluxo de execução deve ser rastreável do início ao fim.**

- **Logger estruturado:** `apps/orchestrator/src/lib/logger.ts` com levels: `info`, `warn`, `error`, `debug`
- **Context tags obrigatórias:** `logger.info("mensagem", "contexto")` — ex: `"whatsapp"`, `"git"`, `"agent"`, `"task"`, `"auth"`
- **Request logging:** middleware logando `method`, `path`, `statusCode`, `duration(ms)` para toda requisição
- **Task lifecycle logging:** toda transição de estado logada em `task_logs` (audit trail completo)
- **Agent execution logging:** início, progresso, tool calls, erros, conclusão — tudo logado com `taskId`
- **Git operations logging:** cada push, pull, commit, branch operation com resultado e duração
- **Integration logging:** status de conexão, mensagens recebidas/enviadas, erros de integração
- **EventBus tracing:** eventos emitidos com tipo, payload resumido, timestamp
- **Error logging enriquecido:** stack trace + contexto (taskId, agentId, projectId) em toda exceção
- **Métricas:** duração de task, tokens consumidos, custo — armazenados por task para analytics
- **Padrão:** se um fluxo não tem log, é um bug — todo ponto de decisão deve ter pelo menos `debug`

### 9. Princípios do Design System
**Componentes consistentes, reutilizáveis e documentados.**

- **Icon set:** exclusivamente Lucide React — 1 set para toda a aplicação
- **Cores semânticas:** variáveis CSS mapeadas (`--color-primary`, `--color-success`, `--color-danger`, etc.)
- **Componentes base reutilizáveis:** Card, Badge, Button, Dialog, Toast, ConfirmDialog, Tooltip, Skeleton
- **Layout padrão:** Sidebar 240px (colapsável) + Header 56px + Content (flex-1) + ChatPanel 380px (toggle)
- **Organização de componentes:** `apps/web/src/components/<domain>/<component>.tsx`
  - `components/layout/` — sidebar, header, main-layout
  - `components/tasks/` — task-card, task-detail, create-task-dialog
  - `components/agents/` — agent-card, agent-config
  - `components/integrations/` — whatsapp-config, telegram-config
  - `components/analytics/` — charts, cost-dashboard
  - `components/common/` — componentes genéricos reutilizáveis
- **Tailwind classes consistentes:** mesmas classes para mesmos padrões (ex: `rounded-xl border border-stroke bg-neutral-bg2 p-6` para cards)
- **Estado visual:** loading (skeleton/spinner), empty state (ícone + mensagem), error state (vermelho + retry)
- **Transições:** `transition-colors` em hover/focus, `animate-spin` em loading, `animate-pulse` em skeleton
- **Tamanhos de texto padronizados:** `text-[24px]` stats, `text-[14px]` body, `text-[13px]` secondary, `text-[12px]` labels, `text-[11px]` captions

### 10. Construção Por Fases e SubFases
**Desenvolvimento incremental, planejado e verificável.**

- Cada feature é uma **Fase numerada** (ex: Fase 18)
- Fases complexas divididas em **SubFases com letras** (ex: 18A, 18B, 18C)
- **Antes de implementar:** plano documentado em `docs/DEVELOPMENT_PLAN.md` com:
  - Objetivo e justificativa
  - Arquivos a criar e modificar
  - Interfaces/tipos novos
  - Critérios de verificação
- **Cada SubFase termina com:**
  - `pnpm build` passando sem erros
  - Funcionalidade testável
  - Sem regressões em features existentes
- **Incrementos pequenos:** cada SubFase = 1-3 horas de trabalho, deployável independentemente
- **Ordem de dependência:** sempre implementar backend antes de frontend, types antes de tudo

### 11. Alterações Documentadas no Arquivo CHANGELOG.md
**Toda alteração deve ser rastreável no histórico.**

- **Formato:** `## [x.y.z] - YYYY-MM-DD` seguido de `### Fase N: Título`
- **SubFases:** `#### Fase NA: Subtítulo` com seções Added/Changed/Fixed/Security
- **Conteúdo obrigatório:**
  - Arquivos criados (com path completo)
  - Arquivos modificados (com descrição da mudança)
  - Dependências adicionadas/removidas
  - Breaking changes (se houver)
  - Detalhes técnicos relevantes para debug futuro
- **Versionamento semântico:**
  - Major (x.0.0): breaking changes, multi-tenant, redesign
  - Minor (0.x.0): nova feature/fase completa
  - Patch (0.0.x): bug fixes, hardening, polish
- **Quando atualizar:** ao final de cada SubFase, antes do commit

### 12. Build Funcional e Código Limpo
**A aplicação deve compilar sem erros e estar livre de código morto.**

- **`pnpm build` deve passar** após qualquer alteração — sem exceções
- **Zero imports não utilizados:** remover antes de cada commit
- **Zero variáveis unused:** TypeScript strict mode (`noUnusedLocals`, `noUnusedParameters`)
- **Zero `any` desnecessário:** usar tipos específicos; `unknown` quando incerto, generics quando flexível
- **Zero warnings do compilador:** tratar warnings como erros
- **Formatação consistente:** indentação 2 espaços, semicolons obrigatórios, trailing commas
- **Dead code removal:** código comentado deve ser removido (git tem o histórico)
- **Verificação pré-commit:** build + lint + type-check devem passar localmente antes de push

---

## Regras do Agente

### 1. Comandos Longos
- Se um comando demorar mais de 2 minutos, cancelar ou converter para background task
- Usar `timeout` em operações de rede/git
- Preferir execuções paralelas quando possível

### 2. Abordagem Alternativa
- Se uma solução falhar 2x, pesquisar na internet por alternativas
- Não ficar preso em uma abordagem — pivotar rapidamente
- Consultar documentação oficial quando APIs mudam

### 3. Economia de Token
- Foco na implementação — menos resumos, mais código
- Não repetir código já lido — referenciar por arquivo:linha
- Respostas concisas e diretas
- Não gerar documentação desnecessária (só quando pedido)
- Agrupar operações similares em blocos

### 4. Quando Usar Multi-Agents
- Considerar spawnar um Multi-Agents quando:
  - Trabalho paralelo em frontend + backend (ex: nova feature full-stack)
  - Code review com múltiplos focos (segurança, performance, testes)
  - Tarefas independentes que não escrevem no mesmo arquivo
  - Refactoring grande que afeta múltiplos packages
- Não usar Multi-Agents para tarefas simples, single-file, ou sequenciais
