---
name: tdd-grinder
description: "Orquestrador de ciclo TDD com verificação automática de testes e integração com grind-loop."
disable-model-invocation: true
---

# TDD Grinder

Orquestra o ciclo Red-Green-Refactor com persistência em `.cursor/scratchpad.md` e validação pelo hook `grind-loop.ts`.

## Ciclo Obrigatório

1. **RED**: Escrever teste que falha para o comportamento desejado.
2. **GREEN**: Implementar código mínimo para o teste passar.
3. **REFACTOR**: Melhorar código mantendo testes verdes.
4. **VERIFY**: Executar comando de testes da stack (ver `AGENTS.md` ou `stack-baseline.mdc`).
5. **RECORD**: Atualizar scratchpad com exit code e `ALL_TESTS_PASSED: true|false`.

## Scratchpad

Use `.cursor/scratchpad.md` (template: `.cursor/scratchpad.template.md`). Campos obrigatórios:

```markdown
## Testes
- **Comando executado**: (o da stack em stack-baseline.mdc)
- **Exit code**: 0
- **ALL_TESTS_PASSED**: true
```

Só marque `ALL_TESTS_PASSED: true` após exit code `0` do comando de testes.

## Integração grind-loop

O hook `.cursor/hooks/continuations/grind-loop.ts` intercepta o evento `stop` e:

- Lê `ALL_TESTS_PASSED` no scratchpad
- Reexecuta até 5 iterações se testes não passaram
- Escala para análise profunda na 3ª iteração sem sucesso

## Comandos por Stack

| Stack | Onde | Teste | Lint | Type-check |
|-------|------|-------|------|------------|
| Go | raiz do `go.mod` | `go test ./...` | `go vet ./...` | — |
| Frontend | `frontend/` | `npm test` | `npm run lint` e `npm run fmt:check` | `npx tsc -b` |

## Agentes Relacionados

- `@rpe-test-lead.md` — plano de testes (não implementa)
- `@rpe-tester.md` — implementa e executa testes unitários/integração
- `@rpe-developer.md` — testes mínimos de produção junto ao código
