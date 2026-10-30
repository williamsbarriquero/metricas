# paleta-cores-por-bu — Especificação de feature

> Template: `.cursor/templates/feature-spec.md`
> Destino: `.cursor/docs/specs/paleta-cores-por-bu.md` (versionado no Git)
> Requer **aprovação explícita** antes de gerar testes ou código.

| Campo | Valor |
|-------|-------|
| **Ticket** | FEATURE-PALETA-BU |
| **Status** | **implementado** |
| **Autor** | RPE / arquitetura |
| **Última atualização** | 2026-10-29 |

---

## 1. Contexto

- Problema de negócio: todos os dashboards de Espaço usavam a mesma paleta global de gráficos (`--chart-1` … `--chart-5`). Não dava para reconhecer a BU pelo visual, e espaços da mesma BU não compartilhavam identidade. Listas de espaços apareciam planas, sem agrupamento por Business Unit.
- Objetivo: cada Espaço usa a paleta da sua BU; espaços da mesma BU compartilham a mesma paleta; a visão agregada “Todos” tem identidade própria (azul de marca); listas e glossário separam os espaços por BU; as cores dos gráficos são pastéis agradáveis derivados da cor-marca, sem usar o hex bruto da marca como cor principal dos charts.
- Fora de escopo:
  - Reordenar o glossário canônico de espaços (marcadores / Apps Script).
  - Repintar KPIs, sidebar ou o tema global da aplicação.
  - Tratar “Todos” como uma oitava BU no glossário ou no agrupamento das listas.
  - Alterar o gerador PNG legado (`graficopng`); o slide em produção fotografa o React (`/print`).

## 2. Capacidades (nível 1)

Comportamentos em linguagem natural — sem detalhes de implementação.

1. O sistema conhece as BUs oficiais (Conecta, Contabiliza, Crédito, Integra, Processamento, Valida, Valoriza), cada uma com uma cor-marca (hex) e um nome de exibição.
2. Cada Espaço do glossário pertence a exatamente uma BU; espaços da mesma BU compartilham a mesma identidade visual nos dashboards.
3. A visão “Todos os Espaços” não é uma BU; usa cor-marca azul `#2a56a5` e a mesma regra de geração de paleta pastel.
4. A cor-marca é referência de matiz: o sistema gera cinco tons pastéis harmônicos (não o hex bruto) para as séries T1/T2/T3.
5. Os papéis das séries nos gráficos permanecem: primária, clara, pálida/muted, accent profunda, intermediária — mapeados às variáveis de chart existentes.
6. Ao abrir a visão de um Espaço (ou Todos), os três gráficos (Histórias/Test Bugs, Horas, Vazão) usam a paleta daquela BU (ou de Todos).
7. No comparativo entre espaços, cada barra do ranking usa a cor-marca da BU daquele Espaço (espaços da mesma BU ficam com a mesma cor de barra).
8. Listas de espaços (select do dashboard, seletor de configuração/equipe, glossário) apresentam agrupamento ou coluna por BU, na ordem canônica das BUs.
9. O título da visão do Espaço identifica a BU (nome + indicação visual da cor-marca).
10. Espaço fora do glossário não recebe paleta de BU; o dashboard cai na paleta do tema.
11. Crédito e Processamento compartilham a mesma cor-marca (`#ff6632`); os dashboards dessas BUs ficam visualmente iguais nos charts, mas continuam BUs distintas no agrupamento.
12. O cartão de impressão / print do slide herda a mesma paleta do Espaço fotografado.

## 3. Componentes (nível 2)

| Módulo / camada | Ação (criar / alterar / remover) | Notas |
|-----------------|----------------------------------|-------|
| Glossário de Espaços (domínio) | Alterar | Vincular cada espaço a uma BU; catalogar BUs e cores-marca |
| Geração de paleta (domínio) | Criar | Derivar 5 pastéis a partir do matiz da cor-marca |
| API de espaços | Alterar | Expor BU, nome da BU, cor-marca e paleta |
| API de métricas (bloco) | Alterar | Incluir identidade visual no payload do bloco (Espaço e Todos) |
| API / tela de glossário | Alterar | Expor e exibir BU por espaço |
| Dashboard React (visão Espaço) | Alterar | Aplicar paleta aos gráficos; título com BU |
| Dashboard React (comparativo) | Alterar | Ranking colorido por BU; select agrupado |
| Print React (`/print`) | Alterar | Mesma paleta do Espaço |
| Seletor de espaços (React) | Alterar | Agrupar badges por BU |
| Dashboard HTML (local) | Alterar | optgroup por BU; CSS vars no miolo; ranking por cor |
| Cartão HTML de print | Alterar | CSS vars da paleta no `#print` |
| `graficopng` | — | Fora de escopo |

## 4. Interações (nível 3)

### Fluxo principal

```text
[coordenador] → [seleciona Espaço ou Todos no dashboard]
  → [API de métricas devolve bloco com bu/cor/paleta]
  → [UI aplica paleta pastel aos gráficos T1/T2/T3]
  → [título mostra BU + swatch da cor-marca]
```

### Fluxo comparativo

```text
[coordenador] → [aba Comparativo]
  → [ranking por indicador]
  → [cada barra usa a cor-marca da BU do Espaço]
```

### Fluxo de listagem

```text
[coordenador] → [select / seletor / glossário]
  → [itens agrupados ou rotulados por BU na ordem canônica]
```

### Eventos e exceções relevantes

- Chave desconhecida: sem `bu`/`paleta`; gráficos usam o tema padrão.
- “Todos”: `bu` vazio, `bu_nome` “Todos”, cor-marca `#2a56a5`, paleta pastel derivada (não o azul bruto nos charts).
- Tema claro/escuro: a paleta de marca aplicada no elemento da visão prevalece sobre as `--chart-*` do tema para aquela visão.
- Crédito vs Processamento: mesma cor-marca e mesma paleta de charts; labels de BU distintos nas listas.

## 5. Critérios de aceite (BDD)

```gherkin
Cenário: Espaços da mesma BU compartilham a paleta
  Dado que HPAY e SDA pertencem à BU Integra
  Quando o coordenador abre o dashboard de HPAY e depois o de SDA
  Então os gráficos T1, T2 e T3 usam a mesma paleta pastel em ambos

Cenário: BUs diferentes têm paletas distintas
  Dado que MAPP é Conecta e PO é Valida
  Quando o coordenador compara os dashboards de MAPP e PO
  Então as paletas dos gráficos são visualmente distintas

Cenário: Visão Todos usa azul de marca sem saturar os charts
  Dado que a cor-marca de Todos é #2a56a5
  Quando o coordenador abre a visão Todos
  Então os gráficos usam pastéis derivados desse azul
  E a cor principal dos charts não é o hex bruto #2a56a5

Cenário: Listas agrupadas por BU
  Dado o glossário de espaços cadastrado
  Quando o coordenador abre o select do dashboard ou o seletor de espaços
  Então os itens aparecem agrupados por BU na ordem Conecta, Contabiliza, Crédito, Integra, Processamento, Valida, Valoriza

Cenário: Comparativo colorido por BU
  Dado vários espaços com dados no período
  Quando o coordenador abre o ranking do comparativo
  Então cada barra usa a cor-marca da BU do respectivo Espaço
  E espaços da mesma BU têm a mesma cor de barra

Cenário: Print herda a paleta
  Dado um Espaço com BU mapeada
  Quando o cartão é aberto em /print (ou equivalente HTML)
  Então os gráficos do cartão usam a mesma paleta pastel da visão do Espaço

Cenário: Espaço fora do glossário
  Dado uma chave de projeto inexistente no glossário
  Quando o dashboard tenta aplicar identidade visual
  Então não há paleta de BU e o tema padrão permanece
```

## 6. Aprovação

- [x] Revisado por produto / negócio
- [x] Revisado por engenharia
- [x] **Aprovado para fase técnica** (gerar `paleta-cores-por-bu-technical.md` e testes)

### Mapa de cor-marca (referência de negócio)

| BU / visão | Cor-marca | Espaços |
|------------|-----------|---------|
| Conecta | `#f471ff` | MAPP, ARF, SQUAD |
| Contabiliza | `#ff7d87` | CTE, RPEC |
| Crédito | `#ff6632` | FIDC |
| Integra | `#25e288` | HPAY, SDA |
| Processamento | `#ff6632` | SCG, CAN, SBOF, BNPL, ECG |
| Valida | `#6bd0fb` | PO |
| Valoriza | `#916eff` | KLOK |
| Todos (não é BU) | `#2a56a5` | visão agregada |

### Diretriz de design (iteração)

- A cor-marca **não** deve ser usada literalmente como cor principal dos gráficos quando for neon ou escura demais.
- A paleta dos charts deve ser pastel, legível com rótulos escuros no T2 empilhado, e com distinção suficiente entre as cinco séries nos três gráficos.
- Verdes (Integra) e magentas (Conecta) pedem saturação ainda mais baixa.

---

## Histórico

| Data | Autor | Alteração |
|------|-------|-----------|
| 2026-10-29 | RPE | Versão inicial a partir do plano e da iteração de design pastel |
