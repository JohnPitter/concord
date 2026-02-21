# Napkin

## Corrections
| Date | Source | What Went Wrong | What To Do Instead |
|------|--------|----------------|-------------------|
| 2026-02-20 | self | Modal $effect chamava showModal() antes do bind:this resolver | Usar tick() para esperar DOM atualizar antes de showModal() |
| 2026-02-20 | self | Chamou logout(auth.user.id) mas logout() não aceita args | Verificar assinatura da função antes de chamar |
| 2026-02-20 | self | Toggle não tem prop onchange, usou bind:checked | Checar Props interface do componente antes de passar props |
| 2026-02-21 | self | VAD com setInterval+getByteFrequencyData não era reativo — speakers só atualizava no polling de 2s | Usar requestAnimationFrame + getByteTimeDomainData (RMS) e enriquecer speakers no getter do store (reativo) |
| 2026-02-21 | self | toggleScreenSharing era só boolean toggle, não usava getDisplayMedia | Usar navigator.mediaDevices.getDisplayMedia() com picker nativo do browser |
| 2026-02-21 | user | Colocou @ antes do username no voice channel | Usuário não quer @, apenas o nick do GitHub |
| 2026-02-21 | self | sidebarMembers mapeava role 'member' para undefined, causando comparação falha no MemberSidebar | Sempre mapear todas as roles explicitamente (incluindo 'Member') |
| 2026-02-21 | self | CI falha por main.go embed de frontend/dist que não existe no CI | Criar stub: `mkdir -p frontend/dist && touch frontend/dist/index.html` antes de go vet/build |
| 2026-02-21 | self | golangci-lint latest (v1.64.8) buildado com Go 1.24, incompatível com go.mod Go 1.25 | Usar continue-on-error até versão compatível; go-version-file: go.mod em vez de hardcoded |
| 2026-02-21 | self | svelte-check --ignore só funciona com --no-tsconfig, não com --tsconfig | Instalar @typescript-eslint/types como devDep resolve erros em node_modules/esrap |

## User Preferences
- Comunica em português
- Espera implementação autônoma, sem perguntas desnecessárias
- Prefere commits e push sem hesitação quando solicitado

## Patterns That Work
- Subagent-driven development para tarefas paralelas independentes
- Git worktree para isolamento de branches de feature
- go build ./... + go vet ./... + go test -short ./... como verificação final
- `npx svelte-check --threshold error` para verificar TypeScript no frontend rapidamente
- Para novos props em Svelte 5: adicionar ao $props() destructuring E ao type annotation
- VAD (Voice Activity Detection): requestAnimationFrame + getByteTimeDomainData + RMS > 0.02, enriquecido no getter do store para reatividade
- Screen sharing: getDisplayMedia() com listener no 'ended' event do track para auto-stop
- Mode switch: SEMPRE chamar leaveVoice(), resetChat(), resetVoice() antes de resetMode() para limpeza completa
- Logos SVG: usar import + <img> em componentes normais; inline SVG só quando precisa de cores dinâmicas (ex: ServerSidebar)
- Tradução: API MyMemory (api.mymemory.translated.net) é gratuita sem key, basta langpair=src|tgt

## Patterns That Work (CI)
- `go-version-file: go.mod` em vez de hardcoded — auto-match
- Stub de `frontend/dist` no CI para satisfazer `//go:embed`
- `continue-on-error: true` em jobs de lint que dependem de tool version
- `--threshold error` no svelte-check para ignorar warnings

## Patterns That Don't Work
- Background agents em worktrees podem não escrever arquivos no path correto
- -race flag não funciona no Windows sem CGO_ENABLED=1
- svelte-check `--ignore` com `--tsconfig` — causa erro fatal

## Domain Notes
- Wails v2: main.go DEVE estar na raiz, bindings em frontend/wailsjs/
- Svelte 5 runes: $state, $props, $bindable, $derived, $effect
- TailwindCSS v4: NUNCA adicionar * { padding: 0; margin: 0 } no CSS global
- Rodar app sempre via `wails dev`, nunca `npm run dev` sozinho
- Design System Void: accent é VERDE (#16a34a), NÃO roxo — contraste ao Discord
- O @ do usuario refere-se ao username do GitHub (OAuth), não ao display_name
