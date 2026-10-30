---
name: session-persistence
description: "Persistência de sessão para retomada de trabalho após interrupção via scratchpad."
disable-model-invocation: true
---

# Session Persistence

Persiste estado da tarefa em `.cursor/scratchpad.md` para retomada após interrupção, crash ou iterações do grind-loop.

## Quando Usar

- Tarefas multi-step (`deep`, `ultrawork`)
- Sessões TDD com múltiplas iterações
- Handoff entre agentes (architect → developer → tester)

## Schema do Scratchpad

Copie `.cursor/scratchpad.template.md` para `.cursor/scratchpad.md` no início da tarefa.

Seções obrigatórias:

| Seção | Conteúdo |
|-------|----------|
| Tarefa | Objetivo, intent, agente ativo |
| Estado Atual | Fase, última ação, blockers |
| Testes | Comando, exit code, `ALL_TESTS_PASSED` |
| Checkpoints | Tabela timestamp/ação/resultado |

## Checkpoints

Registre a cada marco significativo:

```markdown
| 2026-06-19 10:30 | Plano aprovado | OK |
| 2026-06-19 11:00 | Testes RED escritos | FAIL esperado |
| 2026-06-19 11:15 | GREEN alcançado | exit 0 |
```

## Retomada

Ao retomar sessão:

1. Ler `.cursor/scratchpad.md`
2. Identificar fase e blockers
3. Continuar do último checkpoint sem repetir trabalho concluído
