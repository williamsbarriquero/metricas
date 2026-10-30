# Modo UltraWork

Ative o modo de automação total para implementar, testar e auto-corrigir sem interrupção.

## Pré-requisitos

- **Bun** no PATH — instalado automaticamente por `scripts/install.sh` (módulo `hooks`); o hook `.cursor/hooks/continuations/grind-loop.ts` executa via `bun run`
- **Scratchpad**: copiar `.cursor/scratchpad.template.md` → `.cursor/scratchpad.md` (gitignored)
- **Hooks**: `.cursor/hooks.json` com evento `stop` registrado

## Passos

1. Copiar `.cursor/scratchpad.template.md` para `.cursor/scratchpad.md` e preencher objetivo, intent `ultrawork` e agente ativo.
2. Classificar intent como `ultrawork` e carregar regra `ultrawork.mdc`.
3. **Auto-planejamento**: se tarefa complexa, `@plan-architecture.md` e/ou `@test-plan.md` antes de implementar.
4. Executar ciclo TDD: RED (`@rpe-tester.md`) → GREEN (`@rpe-developer.md`) → REFACTOR → verificação.
5. Executar comandos da stack conforme `stack-baseline.mdc` e `AGENTS.md` (via `@validate-stack.md` no fechamento).
6. Registrar no scratchpad após cada execução de testes: comando, exit code e `ALL_TESTS_PASSED: true|false`.
7. Consultar diagnósticos LSP antes de encerrar — zero erros nos arquivos alterados.
8. O hook `grind-loop.ts` reexecuta até 5 iterações se testes falharem ou scratchpad incompleto.

## Regras

- `ultrawork.mdc` — fluxo autônomo e critério de parada
- `stack-baseline.mdc` — Definition of Done por stack
- `tdd-workflow.mdc` — ciclo Red-Green-Refactor
- `testing-standards.mdc` — padrões ao editar `*.test.*`
- `safety-guardrails.mdc` — proibições de segurança

## Skills

- `tdd-grinder` — orquestração TDD + integração com grind-loop
- `session-persistence` — checkpoints e retomada via scratchpad
- `lsp-integration` — validação estática antes de concluir

## Comandos auxiliares

| Comando | Quando usar |
|---------|-------------|
| `@plan-architecture.md` | Tarefa complexa ou novo módulo |
| `@test-plan.md` | Definir ordem e prioridade dos testes P0 |
| `@validate-stack.md` | Fechamento DoD (lint, type-check, testes) |
| `@review-pr.md` | Revisão final antes do PR |

## Agentes sugeridos

| Agente | Fase |
|--------|------|
| `@rpe-architect.md` | Planejamento (read-only) |
| `@rpe-test-lead.md` | Plano TDD técnico (read-only) |
| `@rpe-tester.md` | RED — testes unitários/integração |
| `@rpe-developer.md` | GREEN/REFACTOR — código de produção |
| `@rpe-reviewer.md` | Revisão final |

## Critério de parada

- Todos os checks passam com exit code `0` **e** scratchpad contém `ALL_TESTS_PASSED: true`, OU
- Limite de 5 iterações do `grind-loop.ts` atingido (escalar para o usuário).

## Output

Seguir `interaction-standards.mdc`. Ao encerrar, reportar:

1. Tabela de verificação (comando | exit code | status)
2. Estado final do scratchpad (`ALL_TESTS_PASSED`, fase, blockers)
3. Iterações consumidas do grind-loop (se aplicável)
