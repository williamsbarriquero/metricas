# paleta-cores-por-bu — Especificação técnica

> Template: `.cursor/templates/technical-spec.md`
> Destino: `.cursor/docs/specs/paleta-cores-por-bu-technical.md`
> Alimenta diretamente `@rpe-tester.md` e `@test-plan.md` (nível 4 — contratos).

| Campo | Valor |
|-------|-------|
| **Feature spec** | `.cursor/docs/specs/paleta-cores-por-bu.md` (aprovada / implementada) |
| **Ticket** | FEATURE-PALETA-BU |
| **Status** | **aprovado** |
| **Última atualização** | 2026-10-29 |

---

## 1. Decisões arquiteturais

| Decisão | Alternativas | Motivo |
|---------|--------------|--------|
| Fonte única de BU/cor no pacote de glossário Go (`internal/espacos`) | Duplicar mapa no front; config em banco | Front não inventa cor; um SSOT evita drift entre React, HTML e APIs |
| Cor-marca como matiz; paleta pastel derivada (OKLCH) | Usar hex bruto como chart-1; HSL; CSS `color-mix` só no front | Marcas neon (Integra, Conecta, Todos) saturavam os gráficos; pastéis com L alta e C baixa ficam legíveis nos 3 charts |
| Cinco tons → `--chart-1`…`--chart-5` (papéis atuais das séries) | Remapear séries; paleta por gráfico | Menor superfície de mudança; secoes React já leem as variáveis |
| Identidade visual no adaptador JSON, não em `metrics.Bloco` | Embutir cor no domínio de cálculo | Cálculo permanece puro; cor é preocupação de apresentação |
| “Todos” com cor-marca `#2a56a5`, sem slug de BU | Oitava BU; herdar tema | Visão agregada distinta, sem poluir o catálogo de BUs |
| Campos JSON aditivos | Breaking change de schema | Clientes antigos ignoram campos novos |
| HTML e React em manutenção | Só React | Deploy = SPA embutida; HTML local e print HTML sem origem ainda usam CSS vars |
| `graficopng` fora de escopo | Colorir PNG estático | Slide em produção fotografa `/print` React |

### Papéis da paleta (contrato visual)

| Índice | Variável CSS | Papel | Uso típico |
|--------|--------------|-------|------------|
| 0 | `--chart-1` | Primária pastel | Histórias (T1), Evolução (T2) |
| 1 | `--chart-2` | Clara | Sustentação (T2) |
| 2 | `--chart-3` | Pálida / muted | Overhead (T2), Estoque (T3) |
| 3 | `--chart-4` | Accent mais profunda | Test Bugs (T1), Criados (T3) |
| 4 | `--chart-5` | Intermediária | Resolvidos (T3) |

### Regras da função de paleta (contrato de domínio)

- Entrada: hex `#rrggbb` da cor-marca.
- Saída: exatamente 5 hex distintos.
- Índice 0 **não** é obrigatoriamente igual à cor-marca (e, para marcas saturadas, **não deve** ser).
- Matiz derivado da marca; luminosidade na faixa pastel; croma reduzida especialmente em verdes e magentas.
- Pequenos desvios de matiz entre os índices são permitidos para distinguir séries.
- Luminosidade dos tons deve permitir rótulo escuro legível nas fatias do T2 empilhado.
- Hex inválido: fallback determinístico (matiz do azul de tema).

### Ordem canônica de BUs (agrupamento de UI)

1. `conecta` — Conecta — `#f471ff`
2. `contabiliza` — Contabiliza — `#ff7d87`
3. `credito` — Crédito — `#ff6632`
4. `integra` — Integra — `#25e288`
5. `processamento` — Processamento — `#ff6632`
6. `valida` — Valida — `#6bd0fb`
7. `valoriza` — Valoriza — `#916eff`

A ordem de `Lista` de espaços (glossário Jira) **não** muda.

### Mapa Espaço → BU (contrato de domínio)

| Chave | BU |
|-------|-----|
| MAPP, ARF, SQUAD | conecta |
| CTE, RPEC | contabiliza |
| FIDC | credito |
| HPAY, SDA | integra |
| SCG, CAN, SBOF, BNPL, ECG | processamento |
| PO | valida |
| KLOK | valoriza |

---

## 2. Contratos — interfaces e ports (nível 4)

```text
// Domínio (ports conceituais — pacote espacos)

BUInfo { ID, Nome, Cor }
Espaco { Chave, Nome, BU }
Visual { BU, BUNome, Cor, Paleta[5] }

Paleta(corBase hex) → [5]hex
VisualDaChave(chave) → (Visual, ok)
  - chave == TODOS → ok, BU="", BUNome="Todos", Cor=CorTodos, Paleta=Paleta(CorTodos)
  - chave no glossário → ok + Visual da BU
  - chave desconhecida → ok=false

// Apresentação

AplicarPaleta(elementoUI, Visual.Paleta) → define --chart-1..5 no escopo do elemento
AgruparPorBU(itens com bu/bu_nome/cor) → grupos na ordem canônica
```

---

## 3. DTOs e modelos de domínio

| Nome | Campos | Validações |
|------|--------|------------|
| `BUInfo` | `id` (slug), `nome`, `cor` (#rrggbb) | slug ∈ conjunto canônico; cor hex normalizada |
| `Espaco` (glossário) | `chave`, `nome`, `bu` | `bu` obrigatório para itens de `Lista` |
| `Visual` | `bu`, `bu_nome`, `cor`, `paleta[5]` | `paleta` com 5 hex distintos quando `cor` preenchida |
| Item `GET /api/espacos` | `chave`, `nome`, `ordem`, `bu`, `bu_nome`, `cor`, `paleta` | campos de visual preenchidos para o glossário |
| Bloco métricas (JSON SPA) | campos existentes + `bu`, `bu_nome`, `cor?`, `paleta?` | Todos: `bu=""`, `bu_nome="Todos"`, `cor="#2a56a5"`, `paleta` presente |
| Item glossário JSON | + `bu`, `bu_nome`, `cor` | alinhado ao domínio |

`metrics.Bloco` (domínio de cálculo) **não** carrega cor; o adaptador HTTP consulta `VisualDaChave(chave)`.

---

## 4. APIs / eventos / persistência

| Operação | Entrada | Saída | Erros HTTP / códigos |
|----------|---------|-------|----------------------|
| `GET /api/espacos` | — | Array de espaços com `bu`, `bu_nome`, `cor`, `paleta`, `ordem` | 200 |
| `GET /api/metricas` | `inicio?`, `fim?` | `espacos[]` e `todos` com campos de visual aditivos | 200; 422 datas inválidas (inalterado) |
| `GET /api/glossario` | — | `espacos[]` com `bu`, `bu_nome`, `cor` + tiers | 200 |

### Schema / migração (se aplicável)

```text
-- Sem migração de banco.
-- Glossário e cores são constantes de código (SSOT em internal/espacos).
```

### Superfícies de UI (contrato de comportamento)

| Superfície | Comportamento |
|------------|---------------|
| Dashboard React — visão Espaço | Wrapper com `--chart-1..5`; título com swatch (`cor`) + `bu_nome` |
| Dashboard React — select | Grupos por BU (`SelectGroup` / label) |
| Dashboard React — comparativo ranking | Fill por `cor` da BU do espaço |
| `/print` React | Mesmo wrapper de paleta do bloco |
| Seletor de espaços | Seções por BU; badge ativo pode usar `cor` |
| Glossário React | Coluna BU com swatch |
| Dashboard HTML | `<optgroup>` por BU; estilo inline da paleta no miolo |
| Cartão HTML print | Estilo inline da paleta em `#print` |
| Ranking HTML | Fill por cor-marca do espaço |

---

## 5. Mapeamento para testes

Plano TDD derivado de `@rpe-test-lead.md` / `@test-plan.md` (2026-10-29). Ordem de execução: **P0 → P1 → P2**. Handoff: `@rpe-tester.md` (unitário/integração); E2E visual → `@rpe-sdet.md` / aceite manual. **Não mockar** `espacos` nos testes de domínio; HTTP usa o harness existente (`novoApp` / `spaserver`).

### 5.1 Ordem P0 (domínio + contrato HTTP mínimo)

| # | Contrato | Tipo | Complexidade | Arquivo | Critério verificável | Estado |
|---|----------|------|--------------|---------|----------------------|--------|
| 1 | Cor-marca de cada BU + `CorTodos` | unitário | baixa | `internal/espacos/espacos_test.go` | Cada `BUs[i].Cor` e `CorTodos` batem com a tabela de negócio | coberto (`TestCorBasePorBU`) |
| 2 | Ordem canônica de BUs | unitário | baixa | idem | IDs na ordem conecta→…→valoriza | coberto (`TestOrdemBUs`) |
| 3 | Mapa Espaço → BU (15 chaves) | unitário | baixa | idem | Cada chave de `Lista` tem o `BU` esperado; len(Lista)=len(mapa) | coberto (`TestEspacoTemBU`) |
| 4 | `Paleta`: pastel + 5 distintos + ≠ hex bruto | unitário | média | idem | 5 hex únicos; L∈[0.52,0.90]; `Paleta(base)[0] != base` para as 7 marcas | coberto (`TestPaletaPastelEDistinta`) |
| 5 | `Paleta` golden (determinismo) | unitário | baixa | idem | Snapshot fixo por cor-marca; mudança de fórmula atualiza golden na mesma PR | coberto (`TestPaletaGolden`) |
| 6 | Mesma BU / mesma base → mesma paleta | unitário | baixa | idem | HPAY≡SDA; FIDC.Cor≡SCG.Cor e paletas iguais; slugs BU distintos | coberto (`TestMesmaBUMesmaPaleta`) |
| 7 | `VisualDaChave(TODOS)` | unitário | baixa | idem | ok; `bu=""`; `bu_nome=Todos`; `cor=#2a56a5`; `paleta[0]≠#2a56a5` | coberto (`TestVisualTodos`) |
| 8 | `VisualDaChave` desconhecida | unitário | baixa | idem | `ok=false` para chave fora do glossário | coberto (`TestVisualDesconhecido`) |
| 9 | `GET /api/espacos` campos visuais | integração HTTP | baixa | `internal/web/web_test.go` | 200; corpo contém `bu`/`cor`/`paleta` (ex.: HPAY → integra / `#25e288`) | coberto parcial (`TestAPISaudeEEspacos`) |

**Gate P0:** `go test ./internal/espacos/ ./internal/web/ -count=1` → exit `0`.

### 5.2 Ordem P1 (adaptadores SPA + HTML)

| # | Contrato | Tipo | Complexidade | Arquivo | Critério verificável | Estado |
|---|----------|------|--------------|---------|----------------------|--------|
| 10 | `GET /api/metricas`: bloco Espaço com visual | integração HTTP | média | `internal/spaserver/server_test.go` (ou `metricas_test` dedicado) | Em `espacos[]`, item HPAY tem `bu=integra`, `cor=#25e288`, `paleta` len=5, `paleta[0]≠cor` | **gap** |
| 11 | `GET /api/metricas`: bloco `todos` | integração HTTP | média | idem | `todos.bu=""`, `bu_nome=Todos`, `cor=#2a56a5`, `paleta` len=5 e pastel | **gap** |
| 12 | `GET /api/glossario`: BU por espaço | integração HTTP | baixa | `internal/spaserver/server_test.go` | Cada item de `espacos` tem `bu`, `bu_nome`, `cor` alinhados ao domínio | **gap** |
| 13 | Ranking SVG usa `fill` da cor do item | unitário | baixa | `internal/web/web_test.go` | `Ranking` com `Cor` hex emite `fill="#…"`; sem `Cor` mantém `class="s1"` | **gap** (só smoke SVG hoje) |
| 14 | Dashboard HTML: optgroup / paleta (smoke) | integração HTTP | média | `internal/web/web_test.go` | `GET /?espaco=HPAY` contém `optgroup` com label de BU e/ou style com `--chart-1` | **gap** (opcional se flaky em template) |

**Gate P1:** `go test ./internal/espacos/ ./internal/web/ ./internal/spaserver/ -count=1` → exit `0`.

### 5.3 Ordem P2 (front — helper puro; fora do caminho crítico Go)

| # | Contrato | Tipo | Complexidade | Arquivo | Critério verificável | Estado |
|---|----------|------|--------------|---------|----------------------|--------|
| 15 | `agruparPorBU` ordem canônica | unitário | baixa | `frontend/src/lib/paleta.test.ts` | Grupos na ordem ORDEM_BU; itens sem bu em “Outros” | **gap** |
| 16 | `estiloPaleta` / ausência de paleta | unitário | baixa | idem | Com 5 hex → CSS vars `--chart-1..5`; sem paleta → `undefined` | **gap** |

**Gate P2:** em `frontend/`: `npm test` → exit `0`.

### 5.4 Fora do plano automatizado (aceite visual / SDET)

Não bloqueiam o RED/GREEN do domínio. Checklist:

1. Todos: charts pastéis azulados, não `#2a56a5` puro.
2. MAPP vs PO: paletas distintas.
3. HPAY vs SDA: mesma paleta; Integra sem verde neon.
4. Select / seletor / glossário agrupados ou com coluna BU.
5. Comparativo: barras da mesma BU com a mesma cor-marca.
6. `/print` e cartão HTML (se usado) herdam a paleta.

E2E browser (Playwright/etc.): **não** neste ciclo — handoff `@rpe-sdet.md` se o time exigir regressão visual CI.

### 5.5 Dependências e mocks

| Camada | Dependência | Mock? |
|--------|-------------|-------|
| `espacos` | nenhuma | não |
| `web` `/api/espacos` | store + glossário | harness `novoApp` (já existe) |
| `spaserver` metricas/glossario | store + dados demo/coleta | harness `server_test` (já existe); assert só nos campos JSON novos |
| Front `paleta.ts` | nenhuma (puro) | não |

### 5.6 Comandos de fechamento (DoD)

```text
go test ./internal/espacos/ ./internal/web/ ./internal/spaserver/ -count=1
# se P2 tocado:
cd frontend && npm test && npm run lint && npx tsc -b
```

Scratchpad: ao concluir, `ALL_TESTS_PASSED: true|false`.

---

## 6. Edge cases técnicos

- Hex de marca inválido ou malformado → fallback determinístico; não panic.
- Crédito e Processamento: mesma `cor` e mesma `paleta`; `bu` / `bu_nome` diferentes.
- Tema escuro: variáveis inline no container da visão sobrescrevem `--chart-*` do tema apenas naquele escopo.
- Espaço extra fora do glossário nas métricas: sem `paleta` (omitido); UI usa tema.
- Ordem de `Lista` vs ordem de agrupamento: UI agrupa por `BUs`; glossário canônico permanece na ordem Apps Script.
- Ajuste fino visual: alterar apenas parâmetros da derivação pastel (L/C/desvio de hue), não as cores-marca de negócio — salvo decisão explícita de produto.
- Regeneração de golden da paleta: mudança consciente na fórmula exige atualizar snapshot de teste na mesma mudança.

---

## Histórico

| Data | Autor | Alteração |
|------|-------|-----------|
| 2026-10-29 | RPE | Derivado de `paleta-cores-por-bu.md`; inclui iteração pastel (cor-marca ≠ chart-1) |
| 2026-10-29 | RPE / test-lead | §5: plano TDD P0→P1→P2, critérios verificáveis, estado coberto/gap |
