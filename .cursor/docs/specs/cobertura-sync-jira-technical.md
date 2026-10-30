# cobertura-sync-jira — Especificação técnica

> Template: `.cursor/templates/technical-spec.md`
> Destino: `.cursor/docs/specs/cobertura-sync-jira-technical.md`
> Alimenta diretamente `@rpe-tester.md` e `@test-plan.md` (nível 4 — contratos).

| Campo | Valor |
|-------|-------|
| **Feature spec** | `.cursor/docs/specs/cobertura-sync-jira.md` (**aprovada**) |
| **Ticket** | FEATURE-COBERTURA-SYNC-JIRA |
| **Status** | **aprovado** |
| **Última atualização** | 2026-10-29 |

---

## 1. Decisões arquiteturais

| Decisão | Alternativas | Motivo |
|---------|--------------|--------|
| Cobertura por **série** (não só por bloco) | Uma linha por bloco `incidentes` | Criados/resolvidos/estoque têm geometria e JQL distintos; lacunas independentes |
| Intervalos meio-abertos `[inicio, fimExcl)` | Inclusivo nos dois lados | Alinha a `params.Periodo` / `JQLIntervalo` e evita sobreposição de um dia |
| Impressão JQL com **datas sentinela** | Hash de config; versão de schema | Detecta mudança real do predicado; datas sentinela isolam o “miolo” estável do intervalo |
| Não inferir cobertura das linhas SQLite existentes | Backfill a partir de min/max de datas | Evita falsa cobertura (linhas incompletas / syncs antigas parciais); 1ª sync pós-feature = full |
| Upsert (`Mesclar*`) no incremental; `Substituir*` em forçar / impressão / sem equipe | Sempre substituir tabela | Reduz tráfego Jira e preserva meses já válidos; força caminho limpo quando o predicado mudou |
| Transação única: dados + cobertura | Cobertura after-the-fact | Falha Jira ou falha de write não deixa cobertura “mentirosa” |
| Lacunas de horas via `worklogDate` + worklogs de issues | Reusar `/worklog/updated` | Updated é cursor global; inadequado para janelas históricas pontuais |
| Estoque: JQL complementar (prefixo/sufixo/buraco) | Reaplicar `JQLIncidentesEstoque` no período pedido inteiro | Superconjunto depende de `created`/`resolutiondate` relativos ao intervalo; lacunas precisam predicados dedicados |
| Flag `forcar` no `POST /api/sync` (e form HTML) | Só invalidar cobertura manualmente no SQLite | UX explícita na tela Sincronização; confirmação evita acidente |
| Monólito modular (coleta + store + API) | Hexagonal formal | Conforme `agents.md`; menor superfície |
| `MaxMeses = 12` e fórmulas T1/T2/T3 intactos | Aumentar janela; recalcular métricas | Fora de escopo da feature |

### Séries canônicas

| Série (`coleta_cobertura.serie`) | Bloco runner | Campo / natureza | Persistência alvo |
|----------------------------------|--------------|------------------|-------------------|
| `historias` | `historias` | `resolved` | histórias (upsert/subst.) |
| `testbugs` | `testbugs` | `created` | test bugs |
| `incidentes_criados` | `incidentes` | `created` | incidentes tipo criado |
| `incidentes_resolvidos` | `incidentes` | `resolved` | incidentes tipo resolvido |
| `incidentes_estoque` | `incidentes` | superconjunto estoque | incidentes tipo estoque |
| `horas` | `horas` | `worklogDate` | worklogs |

### Geometria de lacunas (contrato)

- Entrada: cobertura atual (união de intervalos disjuntos ordenados) + pedido `[P0, P1)`.
- Saída: lista de lacunas disjuntas ⊆ pedido.
- Pedido ⊆ cobertura → lista vazia.
- Exemplo aprovado: cobertura `[2026-01-01, 2026-05-01)` + pedido `[2025-12-01, 2026-05-31)` → `[2025-12-01, 2026-01-01)`, `[2026-05-01, 2026-05-31)`.
- Após sucesso: cobertura := união(cobertura, pedido) **exceto** se `forcar` ou impressão invalidou → cobertura := pedido.

### Impressão JQL (contrato)

- Calcular o JQL da série com datas sentinela fixas (ex.: início/fim fictícios estáveis).
- Persistida em `coleta_cobertura.impressao` (texto).
- Se impressão_atual ≠ impressão_gravada → tratar como cobertura vazia para aquela série nesta execução; ao sucesso, gravar impressão_atual + intervalo pedido.

### Estoque — JQL complementar (conceitual)

- **Prefixo** (lacuna antes da cobertura): predicado que obtém o “delta” à esquerda do superconjunto já conhecido.
- **Sufixo** (lacuna depois): delta à direita.
- **Buraco** (lacuna interna): predicado do intervalo furado sem rebaixar o que já está coberto.
- Detalhe exato das cláusulas fica na implementação em `internal/coleta/blocos.go`, desde que preserve equivalência com o estoque mensal das métricas.

### Horas — caminhos

| Estado | Estratégia |
|--------|------------|
| Sem cobertura / forçar / impressão inválida | Bulk atual (`/worklog/updated` + list) **ou** `por_issue` conforme `horasModo` — comportamento legado do período inteiro |
| Lacunas incrementais | Por lacuna: JQL `worklogDate` da lacuna + `WorklogsDeIssues` (não `/worklog/updated`) |

---

## 2. Contratos — interfaces e ports (nível 4)

```text
// Domínio cobertura (conceitual — pacote coleta ou store)

Serie = "historias" | "testbugs" | "incidentes_criados"
      | "incidentes_resolvidos" | "incidentes_estoque" | "horas"

Intervalo { Inicio time; FimExcl time }   // [Inicio, FimExcl)

CoberturaSerie {
  Serie
  Impressao string
  Intervalos []Intervalo   // ou representação compacta inicio/fim_excl se 1 segmento
}

Lacunas(cobertura []Intervalo, pedido Intervalo) → []Intervalo
Uniao(a []Intervalo, b Intervalo) → []Intervalo
ImpressaoJQL(serie, cfg, equipe?) → string   // datas sentinela

// Política por execução
PlanoSyncSerie {
  Modo: Incremental | SubstituirCompleto
  Lacunas []Intervalo          // vazio se contido
  Motivo: Contido | Lacunas | ImpressaoMudou | Forcar | SemEquipe | PrimeiraSync
}

// Persistência (assinaturas conceituais)
MesclarHistorias(ctx, rows) / MesclarTestBugs / MesclarIncidentes / MesclarWorklogs
SubstituirHistorias(ctx, rows) / ...   // já existentes; reusar quando Modo=SubstituirCompleto
LerCobertura(ctx, serie) → CoberturaSerie
GravarCoberturaTx(tx, serie, impressao, intervalos)  // mesma tx dos dados

// Runner
Iniciar(ctx, bloco string, opts { Forcar bool }) → (runID, error)

// API
POST /api/sync  body { bloco: string, forcar?: bool }
```

---

## 3. DTOs e modelos de domínio

| Nome | Campos | Validações |
|------|--------|------------|
| `SyncIniciarIn` | `bloco` (string), `forcar` (bool, default false) | `bloco` ∈ {tudo, historias, testbugs, incidentes, horas}; `forcar` opcional |
| `SyncRun` (existente) | id, bloco, status, progresso, etapa, mensagem, período… | Sem breaking change obrigatório; etapa pode exibir “Sincronia completa” |
| `CoberturaLinha` | `serie`, `impressao`, `inicio`, `fim_excl` | `inicio < fim_excl`; datas ISO `YYYY-MM-DD`; `serie` canônica |
| `IntervaloPedido` | derivado de `params.CalcularPeriodo` | Respeita `MaxMeses` (12); meio-aberto via `FimExcl` |
| Form HTML sync | `bloco`, `forcar` (checkbox/hidden) | Mesmas regras; confirmação client-side antes do submit quando forçar |

---

## 4. APIs / eventos / persistência

| Operação | Entrada | Saída | Erros HTTP / códigos |
|----------|---------|-------|----------------------|
| `POST /api/sync` | JSON `{ "bloco": "tudo"\|…, "forcar": false\|true }` | `200` + `SyncRun` | `409` em andamento / Jira não configurado; `422` bloco inválido; demais erros de negócio alinhados ao atual |
| `GET /api/sync/status` | — | status + histórico | Inalterado |
| `POST /sincronizacao` (HTML) | form `bloco`, `forcar` | redirect flash | Mesma semântica do API |
| Sync em background | opts.Forcar | progresso / etapa | Falha Jira → run `erro`; cobertura não avança |

### Schema / migração (se aplicável)

```sql
-- esboço
CREATE TABLE IF NOT EXISTS coleta_cobertura (
  serie     TEXT NOT NULL,
  impressao TEXT NOT NULL,
  inicio    TEXT NOT NULL,  -- YYYY-MM-DD
  fim_excl  TEXT NOT NULL,  -- YYYY-MM-DD (exclusive)
  PRIMARY KEY (serie, inicio, fim_excl)
  -- alternativa aceitável: PK(serie) se a implementação colapsar sempre a um único segmento
);

-- Índices / evolução: se múltiplos segmentos por série forem necessários,
-- PK composta (serie, inicio) e rotina de merge ao gravar.
```

### Regras de persistência

1. **Incremental + impressão ok**: `Mesclar*` nas entidades tocadas pelas lacunas; unir intervalos na cobertura.
2. **Forçar / impressão mudou / sem equipe (quando aplicável)**: `Substituir*` do bloco/série; cobertura := apenas intervalo pedido (apaga segmentos anteriores da série).
3. **Commit**: dados + cobertura na **mesma** transação.
4. **Rollback / erro Jira**: não persiste nova cobertura; não “fecha” lacuna.

### Arquivos-chave (implementação)

- `internal/coleta/blocos.go`, `internal/coleta/runner.go`
- `internal/store/store.go`, `internal/store/modelos.go`
- `internal/spaserver/api.go`
- `internal/web/paginas.go`, `internal/web/templates/sincronizacao.html`
- `frontend/src/pages/sincronizacao.tsx`, `frontend/src/lib/api.ts`
- `frontend/src/components/ui/switch.tsx`, `frontend/src/components/ui/dialog.tsx`
- `internal/params` (`JQLIntervalo`, período) — métricas já filtram por mês

---

## 5. Mapeamento para testes

| Contrato | Tipo de teste | Prioridade | Arquivo previsto |
|----------|---------------|------------|------------------|
| Lacunas: jan–abr depois dez–maio → só pontas | unitário (domínio lacunas) + integração coleta | P0 | `internal/coleta/cobertura_test.go` (ou `coleta_test.go`) |
| Período contido → zero Jira | integração com `jiratest` fake | P0 | `internal/coleta/coleta_test.go` |
| Mudança filtro/equipe → impressão muda → full fetch | integração | P0 | `internal/coleta/coleta_test.go` |
| `forcar=true` substitui e redefine cobertura | integração | P0 | `internal/coleta/coleta_test.go` |
| `forcar=false` incremental | integração | P0 | `internal/coleta/coleta_test.go` |
| Horas lacuna: **não** chama `GET /worklog/updated` | integração (contar `fk.Chamou`) | P0 | `internal/coleta/coleta_test.go` |
| Falha Jira não grava cobertura | integração | P0 | `internal/coleta/coleta_test.go` / store |
| Schema `coleta_cobertura` + Mesclar upsert | unitário store | P1 | `internal/store/*_test.go` |
| `POST /api/sync` aceita `forcar` | integração HTTP | P1 | `internal/spaserver/server_test.go` |
| SPA: switch default off + Dialog antes de forçar | E2E / componente (se houver harness) | P2 | `frontend` (teste de página ou smoke manual checklist) |
| HTML: campo forçar + confirmação | integração web / checklist | P2 | `internal/web/web_test.go` ou manual |
| Métricas T1/T2/T3 inalteradas (regressão) | unitário metrics existente | P1 | suíte `internal/metrics` (sem mudança esperada) |

---

## 6. Edge cases técnicos

- Cobertura vazia + linhas antigas nas tabelas → **ignorar** linhas para cobertura; full fetch do pedido.
- Múltiplas lacunas na mesma sync → N consultas Jira (uma por lacuna) ou batch explícito; resultado mesclado antes do commit único.
- Bloco `tudo` + `forcar=true` → aplica substituição/cobertura-reset em **todas** as séries dos blocos executados.
- Bloco `incidentes` + incremental → três planos de série independentes; falha em uma série não deve marcar cobertura das outras como sucesso (granularidade por série na tx ou falha do bloco inteiro — preferir **bloco falha ⇒ nenhuma das três coberturas daquele bloco avança** se a coleta atual é atômica por bloco).
- Equipe vazia / modo que zera JQL de pessoas → alinhado ao `Substituir*` atual (limpa tabela); cobertura não deve afirmar dados de equipe.
- Mudança só de período (mesma impressão) → lacunas; mudança de projetos/tipos/status/equipe → impressão.
- Relógio / fuso: datas de cobertura em calendário `YYYY-MM-DD` iguais às strings de `Periodo.InicioStr` / `FimExclStr`.
- Corrida: uma sync por vez (`ErrEmAndamento`); sem merge concorrente de cobertura.
- Demo / Jira não configurado → sync não inicia; cobertura intacta.
- UI: switch off por padrão; habilitar forçar exige Dialog; cancelar Dialog não dispara POST.

---

## Histórico

| Data | Autor | Alteração |
|------|-------|-----------|
| 2026-10-29 | RPE / arquitetura | Derivado de `cobertura-sync-jira.md` — design aprovado |
