# Napkin

## Corrections
| Date | Source | What Went Wrong | What To Do Instead |
|------|--------|----------------|-------------------|
| 2026-02-20 | self | Modal $effect chamava showModal() antes do bind:this resolver | Usar tick() para esperar DOM atualizar antes de showModal() |
| 2026-02-20 | self | Chamou logout(auth.user.id) mas logout() não aceita args | Verificar assinatura da função antes de chamar |
| 2026-02-20 | self | Toggle não tem prop onchange, usou bind:checked | Checar Props interface do componente antes de passar props |

## User Preferences
- Comunica em português
- Espera implementação autônoma, sem perguntas desnecessárias
- Prefere commits e push sem hesitação quando solicitado

## Patterns That Work
- Subagent-driven development para tarefas paralelas independentes
- Git worktree para isolamento de branches de feature
- go build ./... + go vet ./... + go test -short ./... como verificação final

## Patterns That Don't Work
- Background agents em worktrees podem não escrever arquivos no path correto
- -race flag não funciona no Windows sem CGO_ENABLED=1

## Domain Notes
- Wails v2: main.go DEVE estar na raiz, bindings em frontend/wailsjs/
- Svelte 5 runes: $state, $props, $bindable, $derived, $effect
- TailwindCSS v4: NUNCA adicionar * { padding: 0; margin: 0 } no CSS global
- Rodar app sempre via `wails dev`, nunca `npm run dev` sozinho
