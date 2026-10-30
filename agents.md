# Operational Policies for Coding Agents

Baseline do **RPE Harness** neste repositório (Métricas dos Coordenadores). Stacks instaladas em [`.cursor/harness-stacks.json`](.cursor/harness-stacks.json): `go` e `react`.

| Documento | Conteúdo |
|-----------|----------|
| [`.cursor/knowledge/README.md`](.cursor/knowledge/README.md) | Modelo rules → skills → knowledge |
| [`.cursor/knowledge/stacks/`](.cursor/knowledge/stacks/) | SSOT técnico instalado: `go.md`, `frontend.md` |
| [`.cursor/commands/validate-stack.md`](.cursor/commands/validate-stack.md) | Fechamento de verificação (os mesmos comandos desta página) |

---

## Stack do Projeto

Manifests: [`go.mod`](go.mod) (`module metricas-coordenadores`, `go 1.26.0`) e [`frontend/package.json`](frontend/package.json). Registro do harness: [`.cursor/harness-stacks.json`](.cursor/harness-stacks.json).

| Item | Go | Frontend |
|------|----|----------|
| **Stack principal** | API, coleta Jira, métricas T1/T2/T3, SQLite, OAuth Google, CSV e painel HTML | SPA React 19 + TypeScript + Vite |
| **Runtime / build** | Go 1.26. Imagem: `golang:1.26-bookworm`, `CGO_ENABLED=0`, binário `./cmd/spa`. SQLite puro (`modernc.org/sqlite`) | Node 20.19+ ou 22.12+ só na compilação (`node:22-bookworm-slim`). `npm run build` = `tsc -b && vite build`. A imagem final não leva Node |
| **Test runner** | `go test` (stdlib). Doubles em `internal/jiratest` e `internal/googletest` | Vitest + Testing Library (`npm test` em `frontend/`) |
| **Knowledge SSOT** | [`.cursor/knowledge/stacks/go.md`](.cursor/knowledge/stacks/go.md) | [`.cursor/knowledge/stacks/frontend.md`](.cursor/knowledge/stacks/frontend.md) |
| **Skill sob demanda** | `@go-mastery` | `@frontend-mastery` |

`@java-mastery` e `@node-mastery` não estão instalados. O SSOT de frontend descreve esta SPA (Vite, React Router, TanStack Query, shadcn/ui, Base UI).

**Entrypoints (decisão).** Os dois permanecem em manutenção. O pod, a imagem e `./run.sh` usam só `cmd/spa`. O painel HTML é o uso local. O deploy continua um processo: o binário embute `frontend/dist` (`go:embed`) e serve a API. Node existe só no stage de build da imagem.

**HTTP (decisão).** O roteamento fica no `net/http.ServeMux` do Go 1.26 (método, `{param}` e `PathValue`). chi não é dependência: as rotas são planas e o middleware de origem e de recover envolve o mux inteiro. Não há subárvore autenticada nem versão de API que peça `Route` ou `Group`.

- [`cmd/spa`](cmd/spa) — artefato de `./run.sh`, `./dev.sh`, Docker e EKS. Embute o `dist` do React (`go:embed`) e serve a API. Local: `127.0.0.1:8090` (`./dev.sh` sobe o Vite em `:5173`). Na imagem, `ADDR=0.0.0.0:8080`.
- [`main.go`](main.go) — painel HTML local. A partir de `golang/`: `go run ..` (porta 8080, banco em `golang/data`). Não entra na imagem. CSV e OAuth continuam no Go nos dois modos. Regra de negócio em `internal/` vale para os dois.

---

## Arquitetura

Monólito modular em um processo. O cálculo das métricas fica no Go; o React só consome `/api`. CSV (`/csv`) e OAuth Google (`/google`) continuam no servidor Go. O painel HTML (`main.go`) e o SPA (`cmd/spa`) compartilham `internal/`. Mudança de regra de negócio vale para os dois. O pod roda só `cmd/spa`.

Camadas (pacotes em `internal/`, layout Standard Go; não é hexagonal formal):

| Camada | Pacotes | Papel |
|--------|---------|--------|
| Entrada HTTP | `spaserver`, `web` | `spaserver`: JSON + arquivos do Vite e fallback SPA. `web`: páginas HTML, CSV, OAuth e impressão do cartão |
| Aplicação | `coleta`, `slide`, `metrics` | Sincronização com o Jira, geração do Slide, agregação T1/T2/T3 |
| Regras | `equipe`, `params`, `mapaslide`, `espacos`, `tempo`, `formato` | Pertencimento à equipe, período, JQL, marcadores `{{…}}` |
| Adapters | `jira`, `google`, `store`, `graficopng` | REST Jira (retry, semáforo), Drive/Slides, SQLite, PNG do cartão |
| Suporte | `config`, `conc`, `demo` | `.env`, worker pool / fan-out, dados fictícios (`-demo`) |
| Testes | `jiratest`, `googletest` | Jira e Google falsos (`httptest`) |

Frontend (`frontend/src/`): páginas em `pages/`, UI em `components/` (shadcn + Base UI), cliente HTTP em `lib/api.ts` (`fetch` em `/api`). Estado de servidor com TanStack Query. Rotas em `App.tsx` (React Router 7): `/`, `/metricas`, `/mapa-slide`, `/dados/:aba`, `/equipe`, `/configuracoes`, `/sincronizacao`, `/glossario`, `/print`. No dev, o Vite faz proxy de `/api`, `/csv`, `/google` e `/static` para `127.0.0.1:8090`.

Concorrência da coleta: `internal/conc` (pool, fan-out/fan-in, cancelamento). Um semáforo global limita as chamadas simultâneas ao Jira (`JIRA_CONCORRENCIA`, padrão 12).

**Artefatos do harness (não duplicar aqui o conteúdo técnico de stack):**

- Regras globais: `.cursor/rules/rpe-identity.mdc`, `stack-baseline.mdc`, `intent-routing.mdc`, `safety-guardrails.mdc`
- Regras contextuais deste install: `go-standards.mdc`, `frontend-standards.mdc`, `typescript.mdc`, `tdd-workflow.mdc`
- Agentes: `.cursor/agents/rpe-*.md` — invocar com `@` conforme a fase (design, TDD, implementação, revisão)
- Conteúdo de stack: editar somente `.cursor/knowledge/stacks/go.md` ou `frontend.md`

---

## Restrições

Limites que agentes não devem violar.

**Negócio**

- Fórmulas T1, T2 e T3 permanecem equivalentes às do Apps Script. Os testes em `internal/metrics` conferem valores calculados à mão. Mudou a fórmula, atualize o teste no mesmo cambio.
- T1: histórias resolvidas no mês (tipos História, Story ou Problema; status Concluido, Concluído ou Done; sem parent SQUAD-3632; resumo sem QA) e Test Bugs resolvidos no mês, sem filtro de Equipe; quebra por fluxo (labels). O limite de `resolved` é o dia 1 do mês seguinte à data fim. Os gráficos só batem com o Jira depois de uma nova sincronização.
- T2: sustentação, evolução e overhead sobre horas da equipe (referência Timesheet 34680). Percentuais sobre o total sem recesso. Recesso e "(Sem conta)" não entram como overhead.
- T3: criados (29828), resolvidos (29812) e estoque de todo o espaço, sem filtrar responsável pela equipe. Estoque é o snapshot do fim do mês (card aberto naquele instante). TOT do estoque é o último mês, não a soma. Vazão = resolvidos ÷ criados × 100.
- Período: no máximo 12 meses (`params.MaxMeses`). A tela pede início no 1º dia do mês (gráficos mês a mês) e sugere 3 a 6 meses. `CalcularPeriodo` rejeita fim anterior ao início e mais de 12 meses; não há piso de 3 meses no código.
- Lista padrão da equipe: `internal/equipe/padrao.txt` (62 pessoas em 5 coordenadores). O cabeçalho `##` define o coordenador do bloco e não entra na lista. O coordenador só organiza a tela Equipe; não filtra Horas, Histórias nem Test Bugs. Alterar o padrão exige recompilar. Incluir alguém novo exige sincronizar de novo; tirar alguém da lista vale na hora para as horas.
- `-demo` não chama Jira, Google nem Chrome. A geração de slide copia o modelo e preenche marcadores; o arquivo modelo não é alterado. O print do cartão precisa de Chrome/Chromium na máquina (`CHROME_PATH`); a imagem Docker não traz o navegador.
- Não recalcular T1/T2/T3 no browser.

**Segurança**

- Segredos só em `.env` ou no SQLite (`google_token`): `JIRA_TOKEN`, `GOOGLE_CLIENT_SECRET`, access/refresh token. Nunca hardcodar, commitar, logar ou enviar ao browser. Hooks `secret-scanner.sh` e `write-file-guard.sh`.
- Mutações HTTP (não GET/HEAD) exigem `Origin` igual ao host, ou loopback cruzado só entre Vite (`:5173`) e API local. Não afrouxar `origemPermitida`.
- **Autenticação (decisão).** O processo não autentica usuários e não vai ganhar login, sessão nem RBAC. Na máquina a escuta padrão é `127.0.0.1`. Na imagem, `0.0.0.0:8080` só existe para o container receber o Service. Não há Ingress, NetworkPolicy nem oauth2-proxy neste repositório (`metricas/` também não tem). Quem publicar o pod mantém o Service fora da internet pública, com autenticação na borda do cluster. Agente não cria Service público e não implementa login no Go nem no React.
- `CGO_ENABLED=0` no build de produção. Não introduzir SQLite que exija CGO.
- Input externo (config, equipe, período, JQL montada a partir de config) validado no servidor.

**Escopo**

- Node existe só para compilar o front. Não acrescentar runtime Node na imagem nem um segundo processo no pod.
- Não remover o painel HTML (`main.go`) ao evoluir o SPA, nem deixar `cmd/spa` de fora da imagem. Os dois entrypoints são suportados; só `cmd/spa` vai para o pod.
- Não introduzir Next.js, Zustand, ESLint, Prettier, Java nem chi. O linter do front é oxlint (type-aware, com `oxlint-tsgolint`); o formatador é oxfmt. Os dois são devDependency e não entram na imagem.
- Shell/MCP: guards em `.cursor/hooks/guards/` bloqueiam comandos destrutivos.
- Conteúdo de stack: não duplicar regras de Go/React em rules globais; o SSOT é o knowledge.

---

## Verificação por Stack

Antes de dar a tarefa por concluída, rode os checks da stack que o cambio tocou. São os mesmos de `stack-baseline.mdc` e de `@validate-stack.md`.

### Go

Na raiz do módulo (pasta do `go.mod`):

* Formatação: `gofmt -w .`
* Lint: `go vet ./...`
* Testes: `go test ./...`

**Lint Go (decisão).** O gate é `go vet`. `golangci-lint` fica de fora: não há `.golangci.yml` e o README do módulo já usa `go vet`. O detector de corrida (`CGO_ENABLED=1 go test -race ./...`) é opcional; a imagem usa `CGO_ENABLED=0`.

### Frontend (React / Vite)

Em `frontend/`:

* Lint: `npm run lint` (oxlint type-aware, [`.oxlintrc.json`](frontend/.oxlintrc.json), pacote `oxlint-tsgolint`)
* Formatação: `npm run fmt:check` (oxfmt, [`.oxfmtrc.json`](frontend/.oxfmtrc.json); aspas simples, sem ponto e vírgula)
* Type-check: `npx tsc -b` (`strict: true`; `tsconfig.app.json` com `noEmit`; o `build` encadeia `tsc -b && vite build`)
* Testes: `npm test` (`vitest run`)

ESLint e Prettier não fazem parte do projeto. `./scripts/validate.sh` confere a instalação do harness, não a saúde do produto. O `npm run build` que a imagem usa não chama oxlint nem oxfmt.

---

## Definition of Done (DoD)

Uma tarefa é considerada fisicamente concluída se e somente se todos os checkpoints de verificação aplicáveis à stack do projeto retornarem com código de status zero (0):

1. O código compila sem erros de tipagem/compilação (`go vet` no Go; `npx tsc -b` no front).
2. A suíte da stack tocada executa com zero falhas (`go test ./...` e, se o front mudou, `npm test` em `frontend/`).
3. Linter e formatador aplicáveis rodam com sucesso (`gofmt` e `go vet` no Go; `npm run lint` e `npm run fmt:check` no front).
4. O estado do Git é validado (se inicializado), e as modificações estão prontas para commit com mensagens convencionais (Conventional Commits) quando o usuário solicitar.

---

## Communication and Response Policy

Para garantir clareza, rastreabilidade e evitar poluição visual, todo agente deve responder ao usuário seguindo estritamente as diretrizes abaixo:

1. **Tom de Voz**: Profissional, técnico, direto e em Português (BR). Evite introduções e conclusões prolixas ou clichês (ex: *"Certamente, posso ajudar..."* ou *"Espero ter ajudado!"*).
2. **Links de Arquivos**: Sempre utilize links markdown absolutos ou relativos com o esquema `file:///` para apontar para arquivos e trechos de código (ex: `[agents.md](file:///caminho/para/agents.md)`). Não envolva o texto do link com crase.
3. **Template de Saída (Output Structure)**: Para tarefas complexas, divida sua resposta usando cabeçalhos de nível 2 na seguinte ordem:
   * `## 🔍 Análise`: Diagnóstico do problema e constraints identificados.
   * `## 🛠️ Proposta`: Arquivos que serão lidos, modificados, criados ou deletados.
   * `## 💻 Execução`: Implementação real do código ou saída de comandos.
   * `## ✅ Verificação`: Como a alteração foi validada localmente (testes, linters, etc.).
   * `## 📌 Status e Próximos Passos`: O estado final e ações requeridas do usuário.
