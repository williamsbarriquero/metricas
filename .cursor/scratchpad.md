# Scratchpad — Sessão de Trabalho

> Template: `.cursor/scratchpad.template.md`
> Não substitui a spec. Specs: `.cursor/docs/specs/paleta-cores-por-bu.md` + `paleta-cores-por-bu-technical.md`.

## Tarefa

- **Objetivo**: Plano TDD da feature paleta de cores por BU; atualizar §5 da spec técnica; registrar estado da sessão.
- **Intent**: test
- **Agente ativo**: rpe-test-lead (+ `@test-plan.md`)
- **Ticket / slug**: FEATURE-PALETA-BU / `paleta-cores-por-bu`

## Estado Atual

- **Fase**: testes (plano TDD definido; implementação de produto já existe; **gaps P1/P2** no mapeamento de testes)
- **Última ação**: Gerado plano P0→P1→P2; §5 de `paleta-cores-por-bu-technical.md` reescrita com ordem, critérios, estado coberto/gap; scratchpad atualizado.
- **Blockers**: nenhum

## Specs

| Arquivo | Status |
|---------|--------|
| `.cursor/docs/specs/paleta-cores-por-bu.md` | implementado |
| `.cursor/docs/specs/paleta-cores-por-bu-technical.md` | aprovado; §5 atualizada 2026-10-29 |

## Plano TDD (resumo executável)

### P0 — domínio + `/api/espacos` (já coberto em grande parte)

1. `TestCorBasePorBU` — cores-marca
2. `TestOrdemBUs` — ordem canônica
3. `TestEspacoTemBU` — mapa 15 chaves
4. `TestPaletaPastelEDistinta` — pastel / distintos / ≠ bruto
5. `TestPaletaGolden` — snapshot
6. `TestMesmaBUMesmaPaleta` — HPAY≡SDA; FIDC/SCG mesma base
7. `TestVisualTodos` / `TestVisualDesconhecido`
8. `TestAPISaudeEEspacos` — `bu`/`cor`/`paleta` no JSON

**Arquivo principal:** `internal/espacos/espacos_test.go` + `internal/web/web_test.go`  
**Gate:** `go test ./internal/espacos/ ./internal/web/ -count=1`

### P1 — gaps a fechar com `@rpe-tester.md`

| # | Critério | Arquivo |
|---|----------|---------|
| 10 | `/api/metricas` Espaço HPAY com visual | `internal/spaserver/server_test.go` |
| 11 | `/api/metricas` `todos` com visual Todos | idem |
| 12 | `/api/glossario` campos `bu`/`bu_nome`/`cor` | idem |
| 13 | Ranking SVG `fill` quando `Cor` setada | `internal/web/web_test.go` |
| 14 | (opcional) HTML optgroup / `--chart-1` smoke | `internal/web/web_test.go` |

**Gate:** `go test ./internal/espacos/ ./internal/web/ ./internal/spaserver/ -count=1`

### P2 — front helper (opcional neste ciclo)

- `frontend/src/lib/paleta.test.ts` — `agruparPorBU`, `estiloPaleta`
- Gate: `cd frontend && npm test`

### Aceite visual (não automatizado neste ciclo)

Todos pastel ≠ `#2a56a5`; MAPP≠PO; HPAY≡SDA; listas por BU; comparativo por cor-marca; `/print`.

## Testes

- **Comando executado**: (nesta sessão de plano) nenhum novo ciclo RED/GREEN
- **Exit code**: n/a (plano)
- **ALL_TESTS_PASSED**: false  
  Motivo: gaps P1 (#10–#13) ainda não cobertos por asserts dedicados; P0 já existia da implementação.

## Handoff

1. `@rpe-tester.md` — fechar gaps P1 na ordem 10→13 (14 opcional).
2. `@rpe-tester.md` — P2 se quiser cobertura do helper TS.
3. `@rpe-sdet.md` — só se CI visual/E2E for exigido.
4. Não alterar cores-marca de negócio sem atualizar feature spec primeiro.

## Checkpoints

| Timestamp | Ação | Resultado |
|-----------|------|-----------|
| 2026-10-29 | Feature + technical specs criadas | ok |
| 2026-10-29 | Implementação paleta pastel | código + P0 tests |
| 2026-10-29 | Plano TDD + §5 technical + scratchpad | ok |

## Notas

- Spec técnica é SSOT de contratos; este scratchpad só rastreia sessão.
- Crédito e Processamento: mesma `cor`/`paleta`, slugs distintos — coberto em P0 #6.
- Regenerar golden de `Paleta` só junto com mudança consciente da fórmula.
