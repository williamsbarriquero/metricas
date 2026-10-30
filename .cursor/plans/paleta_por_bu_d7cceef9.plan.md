---
name: Paleta por BU
overview: Cada Espaço passa a usar a paleta da sua BU, e a visão Todos usa a base azul
todos:
  - id: dominio-bu
    content: Catalogar BUs, vincular cada Espaço, definir a base
    status: completed
  - id: api
    content: Expor bu, bu_nome, cor e paleta em GET /api/espacos e no bloco de GET /api/metricas
    status: completed
  - id: react-dashboard
    content: Aplicar a paleta nos gráficos do dashboard e do /print, agrupar o select e colorir o ranking por BU
    status: completed
  - id: react-listas
    content: Agrupar glossário e seletor de espaços por BU
    status: completed
  - id: html
    content: optgroup no select HTML, variáveis CSS no dashboard e no cartão de print, ranking por cor da BU
    status: completed
  - id: verificar
    content: Rodar testes Go e front e conferir no browser dois espaços de BUs diferentes e dois da mesma BU
    status: completed
isProject: true
---

# Paleta de cores por BU

## Contexto

Os gráficos T1, T2 e T3 usam cinco cores fixas do tema (`--chart-1` … `--chart-5`) em [frontend/src/index.css](frontend/src/index.css) e [internal/web/static/app.css](internal/web/static/app.css). Não há BU no domínio: [internal/espacos/espacos.go](internal/espacos/espacos.go) só guarda chave e nome, na ordem do Apps Script.

Os dois painéis continuam em manutenção ([agents.md](agents.md)): o deploy é o React (`cmd/spa`); o HTML em `internal/web` é o uso local. O print do slide, com origem definida, fotografa `/print` (React). Sem origem, usa o cartão HTML.

## Decisões

- Fonte única: catálogo de BUs e o vínculo de cada Espaço em `espacos.go`. O front não inventa cor.
- Cor base gravada como hex. Os outros quatro tons são calculados em Go a partir dessa cor (mesmo matiz OKLCH, luminosidade e croma diferentes), para as séries continuarem distinguíveis.
- Papéis dos gráficos não mudam. Só o valor das cinco variáveis muda:
  - `--chart-1` = cor base (Histórias, Evolução)
  - `--chart-2` = tom mais claro (Sustentação)
  - `--chart-3` = tom mais pálido (Overhead, Estoque)
  - `--chart-4` = tom mais escuro (Test Bugs, Criados)
  - `--chart-5` = tom intermediário (Resolvidos)
- `TODOS` usa a mesma regra de paleta, com cor base azul `#2a56a5` (não é uma BU). Chave fora do glossário mantém a paleta atual do tema.
- A ordem de `Lista` não muda (marcadores e Apps Script). O agrupamento é só de apresentação, na ordem das BUs abaixo.
- Crédito e Processamento usam o mesmo hex (`#ff6632`), como informado. Os dashboards dessas duas BUs ficam iguais.
- `graficopng` fica de fora: o slide em produção fotografa o React, e o fallback local é o HTML.

## Mapa Espaço → BU

- Conecta `#f471ff`: MAPP, ARF, SQUAD
- Contabiliza `#ff7d87`: CTE, RPEC
- Crédito `#ff6632`: FIDC
- Integra `#25e288`: HPAY, SDA
- Processamento `#ff6632`: SCG, CAN, SBOF, BNPL, ECG
- Valida `#6bd0fb`: PO
- Valoriza `#916eff`: KLOK
- Todos (visão agregada, não é BU) `#2a56a5`

## Contrato

`GET /api/espacos` (HTML, usado pelo seletor React) e o bloco de `GET /api/metricas` (SPA) ganham campos aditivos:

- `bu` (slug: `conecta`, `contabiliza`, `credito`, `integra`, `processamento`, `valida`, `valoriza`)
- `bu_nome`
- `cor` (hex da base)
- `paleta` (cinco hex, na ordem chart-1 … chart-5)

O bloco `TODOS` recebe `bu` vazio, `bu_nome` "Todos", `cor` `#2a56a5` e `paleta` derivada dessa base. Chave desconhecida devolve `bu` vazio e sem paleta (o front cai no tema).

O domínio de métricas (`metrics.Bloco`) não guarda cor. O adaptador em [internal/spaserver/metricas.go](internal/spaserver/metricas.go) consulta `espacos` ao montar o JSON.

## Aplicação visual

- React: um wrapper em volta de T1/T2/T3 (dashboard e `/print`) define `--chart-1` … `--chart-5` inline. As séries em [frontend/src/components/dashboard/secoes.tsx](frontend/src/components/dashboard/secoes.tsx) já leem essas variáveis. O título do espaço ganha o nome da BU e um swatch da cor base.
- Comparativo (React e HTML): cada barra do ranking usa a cor base da BU daquele Espaço. Espaços da mesma BU ficam da mesma cor.
- Listas agrupadas por BU, com rótulo da BU:
  - select do dashboard React (`SelectGroup` / `SelectLabel`, que já existem)
  - `<optgroup>` no select do dashboard HTML
  - tabela do glossário React (coluna BU)
  - badges de [frontend/src/components/seletor-espacos.tsx](frontend/src/components/seletor-espacos.tsx)
- HTML: o miolo do dashboard e o `#print` do cartão recebem as mesmas variáveis inline, para as classes `.s1`–`.s5` em [internal/web/graficos.go](internal/web/graficos.go) herdarem a paleta sem reescrever o SVG. Tema escuro não substitui cor de marca definida no elemento.

## Tons

Função `Paleta(corBase) [5]string` em `espacos`:

- índice 0 = hex informado, sem alteração
- demais índices: mesmo hue OKLCH, com passos fixos de luminosidade (mais claro, pálido, mais escuro, intermediário) e croma reduzido nos tons claros
- luminosidade limitada para o rótulo escuro do T2 (`oklch(0.2 0 0)`) continuar legível nas três fatias empilhadas
- teste trava o hex de cada BU (golden) e exige os cinco tons diferentes entre si
- HPAY e SDA devolvem a mesma paleta; FIDC e SCG também (mesma base)
- `Paleta` de `TODOS` parte de `#2a56a5`; o índice 0 desse resultado é exatamente esse hex

## Fora de escopo

- Reordenar `espacos.Lista`
- Repintar KPIs, sidebar ou o tema global
- Tratar Todos como uma oitava BU no glossário ou no agrupamento das listas
- `internal/graficopng`

## Riscos

- Tons muito próximos se a cor base já for extrema. O teste de distinção e o clamp de luminosidade cobrem isso; se um BU ficar ruim no olho, ajusta-se só o passo, não o hex da base.
- Crédito e Processamento não se separam no comparativo. É o dado informado.
- Campo novo no JSON é aditivo; clientes antigos ignoram.

## Verificação

- `go test ./internal/espacos/ ./internal/web/ ./internal/spaserver/`
- Em `frontend/`: `npm test`, `npm run lint`, `npm run fmt:check`, `npx tsc -b`
- No browser: abrir o dashboard em Todos e conferir a base azul `#2a56a5`; trocar MAPP (Conecta) e PO (Valida) e conferir gráficos diferentes; abrir SDA e HPAY e conferir a mesma paleta; ver o select e o glossário agrupados; repetir Todos e um espaço no painel HTML local e no `/print`.
