# cobertura-sync-jira — Especificação de feature

> Template: `.cursor/templates/feature-spec.md`
> Destino: `.cursor/docs/specs/cobertura-sync-jira.md` (versionado no Git)
> Requer **aprovação explícita** antes de gerar testes ou código.

| Campo | Valor |
|-------|-------|
| **Ticket** | FEATURE-COBERTURA-SYNC-JIRA |
| **Status** | **aprovado** |
| **Autor** | RPE / arquitetura |
| **Última atualização** | 2026-10-29 |

---

## 1. Contexto

- Problema de negócio: cada bloco de coleta monta JQL do período inteiro configurado e **substitui** a tabela local. Uma nova sync cujo período se sobrepõe a dados já obtidos **reconsulta meses já baixados** e bombardeia o Jira com tráfego desnecessário.
- Objetivo: lembrar, por tipo de consulta (**série**), quais intervalos já foram gravados com sucesso; no caminho padrão, **chamar o Jira só nas lacunas**. Persistir com **upsert** no SQLite (métricas já filtram por mês). Oferecer interruptor **"Forçar período completo"** na tela Sincronização para rebusca total quando houver ajuste no Jira (issues movidas, canceladas, campos alterados, etc.).
- Fora de escopo:
  - Não alterar fórmulas T1 / T2 / T3.
  - Não mudar `MaxMeses` (permanece 12).
  - Não criar job automático de refresh do mês corrente.
  - Sem hexagonal formal (monólito modular conforme `agents.md`).

---

## 2. Capacidades (nível 1)

Comportamentos em linguagem natural — sem detalhes de implementação.

1. O sistema registra **cobertura** por série de coleta: quais intervalos de tempo já foram obtidos do Jira e gravados com sucesso.
2. As séries de cobertura são: **historias** (campo `resolved`), **testbugs** (`created`), **incidentes_criados** (`created`), **incidentes_resolvidos** (`resolved`), **incidentes_estoque** (superconjunto de estoque), **horas** (`worklogDate`).
3. Intervalos de cobertura e de pedido são **meio-abertos** `[inicio, fimExcl)`.
4. No caminho padrão (`forcar = false`), o sistema calcula as **lacunas** entre o período pedido e a cobertura já registrada da série; só consulta o Jira nas lacunas.
5. Se o período pedido estiver **contido** na cobertura existente da série (e a impressão do JQL for a mesma), **não há chamada ao Jira** para aquela série.
6. Na **primeira sync** após a feature (cobertura vazia), o sistema baixa o período inteiro pedido. **Não** se infere cobertura a partir das linhas já existentes nas tabelas de dados.
7. Cada série guarda uma **impressão** do JQL (com datas sentinela). Se a impressão atual divergir da gravada (ex.: mudança de filtro, projetos, equipe ativa, status excluídos), a cobertura da série é **invalidada** e o período pedido é buscado por completo.
8. Persistência de entidades usa **upsert** no caminho incremental; **substituição total** do bloco ocorre quando não há equipe (quando o bloco depende dela), quando a impressão mudou, ou quando `forcar = true`.
9. Dados da série e atualização de cobertura ocorrem na **mesma transação**. Falha na chamada ao Jira **não** grava cobertura nova.
10. Bloco **Incidentes** mantém três séries independentes (criados, resolvidos, estoque). Lacunas de estoque usam JQL **complementar** (prefixo / sufixo / buraco) conforme a geometria da lacuna.
11. Bloco **Horas**: sem cobertura prévia, mantém o caminho bulk atual; com lacunas, consulta por `worklogDate` da lacuna + worklogs das issues encontradas. **Não** reutiliza `/worklog/updated` para preencher lacunas.
12. Na tela Sincronização (SPA React e painel HTML), o usuário dispõe de um switch **"Forçar período completo"** (desligado por padrão). Ao ativar e iniciar sync, um **dialog de confirmação** explica que os dados do bloco serão reescritos no intervalo.
13. Com `forcar = true`, o sistema substitui as séries do bloco no período pedido e a cobertura passa a ser **somente** aquele intervalo (não preserva cobertura anterior fora do pedido).
14. Durante sync forçada, a etapa de progresso comunica **"Sincronia completa"** (ou equivalente claro) para distinguir do caminho incremental.
15. Métricas e dashboards continuam filtrando por mês; não dependem de “última sync = período exibido”.

---

## 3. Componentes (nível 2)

| Módulo / camada | Ação (criar / alterar / remover) | Notas |
|-----------------|----------------------------------|-------|
| Domínio de cobertura (coleta) | Criar | Lacunas, mescla de intervalos, impressão JQL, regras forçar/invalidar |
| `internal/coleta/blocos.go` | Alterar | Historias, TestBugs, Incidentes (3 séries), Horas — sync por lacuna |
| `internal/coleta/runner.go` | Alterar | Propagar flag `forcar`; etapa “Sincronia completa” |
| `internal/store` (`store.go`, `modelos.go`) | Alterar | Tabela `coleta_cobertura`; Mesclar*/Substituir*; tx dados+cobertura |
| `internal/params` (`JQLIntervalo`, período) | Alterar / reusar | Intervalos meio-abertos; impressão com datas sentinela |
| `internal/spaserver/api.go` | Alterar | `POST /api/sync` aceita `forcar: true` |
| `internal/web/paginas.go` + `templates/sincronizacao.html` | Alterar | Switch + confirmação no painel HTML |
| `frontend/src/pages/sincronizacao.tsx` | Alterar | Switch + Dialog; envia `forcar` |
| `frontend/src/components/ui/switch.tsx`, `dialog.tsx` | Reusar | Controles já existentes no design system |
| Fórmulas T1/T2/T3 / `MaxMeses` | — | Fora de escopo |
| Job de refresh automático | — | Fora de escopo |

---

## 4. Interações (nível 3)

### Fluxo principal (incremental)

```text
[coordenador] → [Sincronização: Forçar OFF] → [POST sync bloco / tudo]
  → [Runner carrega cobertura + impressão por série]
  → [se impressão igual: calcula lacunas; senão: período inteiro]
  → [Jira só nas lacunas (ou zero chamadas se contido)]
  → [upsert dados + mescla cobertura na mesma transação]
  → [métricas leem SQLite filtrando por mês]
```

### Fluxo forçar período completo

```text
[coordenador] → [liga "Forçar período completo"] → [Dialog confirma]
  → [POST sync com forcar:true]
  → [substitui séries do bloco no período pedido]
  → [cobertura = somente aquele intervalo]
  → [etapa "Sincronia completa"]
```

### Fluxo invalidação por impressão

```text
[coordenador altera filtro / equipe / config JQL]
  → [próxima sync recalcula impressão da série]
  → [impressão ≠ gravada] → [invalida cobertura da série]
  → [busca período pedido inteiro] → [grava nova impressão + cobertura]
```

### Eventos e exceções relevantes

- Pedido contido na cobertura + impressão igual → zero Jira na série; sync conclui ok (pode registrar 0 linhas novas).
- Exemplo geométrico: cobertura `[2026-01-01, 2026-05-01)` + pedido `[2025-12-01, 2026-05-31)` → lacunas `[2025-12-01, 2026-01-01)` e `[2026-05-01, 2026-05-31)`.
- Falha parcial/total no Jira → não atualiza cobertura da série afetada; dados já commitados em syncs anteriores permanecem.
- Sync em andamento → comportamento atual (`ErrEmAndamento` / 409); flag `forcar` não altera exclusão mútua.
- Bloco sem equipe quando o bloco exige equipe → caminho de substituição / limpeza alinhado ao comportamento atual do bloco (não inventar cobertura falsa).
- Estoque: lacunas em prefixo, sufixo ou buraco interno geram JQL complementar distinto (não simplesmente `JQLIntervalo` de um campo único).
- Horas com lacunas: proibido usar `/worklog/updated` como atalho; usar JQL por `worklogDate` da lacuna + `WorklogsDeIssues` (ou equivalente por_issue).

---

## 5. Critérios de aceite (BDD)

```gherkin
Funcionalidade: Cobertura incremental na sync Jira

  Cenário: Expansão do período consulta só as lacunas
    Dado que a série "historias" tem cobertura [2026-01-01, 2026-05-01)
    E a impressão do JQL está inalterada
    Quando o coordenador sincroniza o período pedido [2025-12-01, 2026-05-31) com forcar=false
    Então o Jira é consultado apenas em [2025-12-01, 2026-01-01) e [2026-05-01, 2026-05-31)
    E a cobertura resultante da série cobre a união dos intervalos

  Cenário: Período contido não chama o Jira
    Dado cobertura [2026-01-01, 2026-05-01) na série
    E impressão inalterada
    Quando o pedido é [2026-02-01, 2026-04-01) com forcar=false
    Então não há chamada ao Jira para aquela série
    E a cobertura permanece a mesma

  Cenário: Primeira sync pós-feature baixa tudo
    Dado cobertura vazia para a série (sem inferência das linhas existentes)
    Quando o coordenador sincroniza qualquer período válido
    Então o Jira é consultado no período pedido inteiro
    E a cobertura passa a ser exatamente esse intervalo

  Cenário: Mudança de filtro ou equipe invalida a série
    Dado cobertura existente com impressão A
    Quando a configuração ou equipe altera a impressão do JQL para B
    E o coordenador sincroniza com forcar=false
    Então a cobertura anterior é invalidada
    E o Jira é consultado no período pedido inteiro
    E a nova impressão B é gravada com a nova cobertura

  Cenário: Forçar período completo substitui e redefine cobertura
    Dado cobertura anterior mais ampla ou distinta
    Quando o coordenador confirma "Forçar período completo" e inicia sync com forcar=true
    Então os dados das séries do bloco são substituídos no intervalo pedido
    E a cobertura de cada série afetada fica igual somente a esse intervalo
    E a etapa de progresso indica sincronia completa

  Cenário: Forçar desligado usa caminho incremental
    Dado cobertura parcial sobreposta ao pedido
    Quando forcar=false
    Então apenas lacunas são consultadas
    E não há substituição total desnecessária dos meses já cobertos

  Cenário: Horas em lacuna não usa worklog/updated
    Dado cobertura parcial na série "horas"
    Quando a sync preenche uma lacuna com forcar=false
    Então a coleta da lacuna usa JQL por worklogDate da lacuna e worklogs das issues
    E não chama GET /rest/api/3/worklog/updated para preencher a lacuna

  Cenário: Falha no Jira não grava cobertura
    Dado uma lacuna a preencher
    Quando a chamada ao Jira falha antes do commit
    Então a cobertura da série não é ampliada
    E a próxima sync ainda considera a lacuna pendente

  Cenário: Jan–abr depois dez–maio (plano de verificação)
    Dado sync bem-sucedida do pedido janeiro–abril (cobertura correspondente)
    Quando nova sync pede dezembro–maio com forcar=false e mesma impressão
    Então só as pontas (dezembro e maio) geram consulta Jira
```

---

## 6. Aprovação

- [x] Revisado por produto / negócio
- [x] Revisado por engenharia
- [x] **Aprovado para fase técnica** (gerar `cobertura-sync-jira-technical.md` e testes)

---

## Histórico

| Data | Autor | Alteração |
|------|-------|-----------|
| 2026-10-29 | RPE / arquitetura | Versão inicial — design aprovado |
