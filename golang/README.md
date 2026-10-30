# Métricas dos Coordenadores — aplicação única em Go

Backend (coleta no Jira, banco SQLite, cálculo das métricas T1/T2/T3), API e **páginas HTML renderizadas pelo próprio Go** em **um só executável**. Templates, CSS, JS e o fuso horário vão embutidos no binário (`go:embed`); o SQLite é Go puro (`modernc.org/sqlite`), então não precisa de CGO, Node, Python nem de outro servidor.

Este entrypoint (`go run ..` a partir daqui) é o painel HTML local. Continua suportado. A imagem e o pod usam `cmd/spa`, não este fluxo.

## Rodar

```bash
cd spa/golang
cp .env.example .env         # preencha JIRA_EMAIL, JIRA_TOKEN e, para o Slide, GOOGLE_CLIENT_ID/SECRET
go run ..                    # abre http://127.0.0.1:8080
```

O código Go está na pasta de cima (`spa/go.mod`). Rodar daqui mantém este `.env` e o banco em `data/`.

Sem credenciais dá para explorar com dados fictícios:

```bash
go run .. -demo              # usa data/demo.db, recriado a cada execução
```

Flags: `-addr host:porta` · `-db caminho.db` · `-demo` · `-nao-abrir` (não abre o navegador) · `-versao`.
Variáveis (ou `.env`, procurado na pasta atual e na do executável; variável de ambiente vence): `JIRA_URL`, `JIRA_EMAIL`, `JIRA_TOKEN`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `DB_PATH` (padrão `data/metricas.db`), `ADDR` (padrão `127.0.0.1:8080`), `JIRA_CONCORRENCIA` (padrão 12).

Reinicie o programa depois de alterar o `.env`. Os tokens do Jira e do Google ficam só no servidor (o da conta Google, no SQLite) e nunca vão para o navegador.

## Gerar o executável

```bash
cd spa/golang
go build -ldflags "-s -w -X main.versao=1.0.0" -o metricas ..          # ~14 MB

# outras plataformas (CGO desligado)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o metricas.exe ..
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -o metricas-mac ..
```

Para distribuir, basta o binário + um `.env` ao lado dele. O banco é criado em `data/` na primeira execução.

## Primeiro uso

1. **Configurações**: período (de 3 até 12 meses), Espaços, nomes das contas de esforço e o Slide modelo (ID ou URL). No card **Google**, conecte a conta (OAuth, cliente tipo Desktop, APIs Drive e Slides ativas no Cloud Console). Redirect: `http://127.0.0.1:{porta}/google/callback`.
2. **Equipe**: na primeira execução (banco novo) já vem cadastrada a lista padrão de 62 pessoas em 5 coordenadores (`internal/equipe/padrao.txt`, nomes como no Jira, todas com "Incluir" ligado). O cabeçalho `##` é o coordenador do bloco e só organiza a tela; não filtra as horas. Para alterar a lista padrão, edite esse arquivo e recompile; na tela, "Completar com a lista padrão" adiciona quem estiver faltando e atualiza o coordenador de quem já existe, sem mexer em Incluir, Account ID, espaços ou observação. "Resolver Account IDs" preenche os IDs.
3. **Sincronização**: "Atualizar tudo". Roda em segundo plano, com progresso na tela. Depois de atualizar o programa para a métrica de Estoque, rode de novo a coleta de **Incidentes** (os dados antigos não têm o superset).
4. **Dashboard / Métricas / Dados**: escolha o Espaço e o período. **Dados** (`/dados/historias` etc.) lista as linhas coletadas, com filtro, paginação e CSV. Cada tabela do dashboard também exporta CSV (`;`, decimal com vírgula, abre direto no Excel).
5. **MapaSlide**: conferir os marcadores `{{MARCADOR}}` e **Gerar Slide** (faz uma *cópia* do modelo e preenche os textos). `{{HPAY_T1}}` / `{{HPAY_T2}}` / `{{HPAY_T3}}` colam um print do cartão do dashboard (gráfico e tabela) na caixa do marcador. `{{HPAY_T1_GRAFICO}}` cola só o gráfico. A geração precisa do Chrome ou do Chromium instalado (`CHROME_PATH` se o binário não estiver no PATH). Os gráficos da planilha são removidos. O modelo nunca é alterado. Em `-demo` o botão só pré-visualiza, sem chamar o Google nem o Chrome.

## Regras de negócio

Idênticas às do Apps Script / versão Python:

- **T1** — histórias resolvidas no mês (tipos História, Story ou Problema; status Concluido, Concluído ou Done; sem parent SQUAD-3632; resumo sem QA) e Test Bugs resolvidos no mês, sem filtro de Equipe, com quebra por Fluxo (labels `Fluxo_principal/alternativo/exceção`).
- **T2** — Sustentação, Evolução e Overhead (todas as outras contas, sem Recesso e sem "(Sem conta)"); percentuais sobre o total sem Recesso. A consulta só traz horas de quem está na Equipe. Referência: Timesheet 34680.
- **T3** — criados (29828), resolvidos (29812) e **Estoque**, de todo o Espaço, sem filtrar o Responsável pela Equipe. Cards que estavam abertos no último instante daquele mês, mesmo que tenham sido fechados depois (`criado_em <= fim do mês` e `resolvido_em` vazio ou depois do fim do mês; status cancelados ficam de fora). Vazão = Resolvidos ÷ Criados × 100. O total do Estoque (**TOT**) é o do **último mês**, não a soma. "Todos os Espaços" soma Criados/Resolvidos; o Estoque total também é o snapshot do último mês.
- Horas: modo `bulk` (`worklog/updated` + `worklog/list`, margem de 30 dias antes do início) com queda automática para `por_issue`. A consulta restringe `worklogAuthor` à Equipe. Tirar alguém da lista vale na hora; incluir alguém novo exige sincronizar de novo.

## Concorrência (goroutines e canais)

Tudo está em `internal/conc` (worker pool, fan-out/fan-in e execução paralela com cancelamento e captura de pânico) e é usado assim:

| Onde | O que roda em paralelo |
| --- | --- |
| `coleta.Runner` | Os blocos do "Atualizar tudo" (Histórias, Test Bugs, Incidentes, Horas) rodam juntos, cada um numa goroutine. Todas publicam andamento e resultado num **canal de eventos**; uma única goroutine agregadora lê o canal e grava o progresso. Se um bloco falha, os outros terminam e a mensagem diz o que deu certo e o que falhou. |
| Horas (modo `bulk`) | **Pipeline** em 3 estágios ligados por canais: produtor lê as páginas de ids (`/worklog/updated`) → N trabalhadores detalham cada lote (`/worklog/list`) enquanto a próxima página ainda está sendo lida → coletor reordena. A busca das issues roda ao mesmo tempo. |
| Horas (modo `por_issue`) | Pool de goroutines, uma chamada por issue. |
| Incidentes | Criados, resolvidos e o superset de estoque buscados ao mesmo tempo. |
| Equipe | "Resolver Account IDs" consulta o Jira de várias pessoas ao mesmo tempo. |
| Dashboard | Leituras do banco (histórias, test bugs, incidentes, worklogs, equipe) e cálculos T1/T2/T3 em paralelo. |

Para não estourar o limite do Jira, o cliente tem um **semáforo global** (canal com `JIRA_CONCORRENCIA` vagas, padrão 12): a soma de todas as requisições simultâneas, de todos os blocos, nunca passa disso. 429/5xx continuam com retry e `Retry-After`.

## Rotas

| Rota | O que é |
| --- | --- |
| `/` | Dashboard (`?espaco=&inicio=&fim=&aba=comparativo`) |
| `/equipe` · `/configuracoes` · `/sincronizacao` · `/glossario` · `/metricas` · `/mapa-slide` | Telas |
| `/dados/{historias,testbugs,criados,resolvidos,estoque,horas}` | Linhas coletadas (filtro, paginação, CSV) |
| `/google/conectar` · `/google/callback` · `POST /google/desconectar` | OAuth Google |
| `POST /slide/gerar` · `/api/slide/status` | Geração do Slide (cópia + marcadores + print do cartão) |
| `/csv/{t1,t2,t3,comparativo}` · `/csv/dados/{aba}` | Exportação CSV |
| `/api/health` · `/api/metricas` · `/api/espacos` · `/api/sync/status` | JSON |

Requisições que alteram dados (POST) só são aceitas com `Origin` igual ao host do servidor. O processo não tem login de usuário e isso permanece assim. Na máquina ele escuta em `127.0.0.1`. Na imagem, `0.0.0.0:8080` só existe para o container receber o Service. Não há manifesto de Ingress neste repositório; o Service não deve ficar público sem autenticação na borda do cluster.

## Qualidade

```bash
cd spa
go vet ./... && go test ./...        # use CGO_ENABLED=1 go test -race ./... para o detector de corridas
```

Os testes usam um Jira falso (`internal/jiratest`, `httptest`) e conferem as métricas contra valores calculados à mão com as fórmulas do Apps Script; também validam as JQLs geradas, a paginação, o retry em 429, a queda do modo bulk, a sincronização ponta a ponta e a renderização de todas as páginas.

## Estrutura

Caminhos abaixo de `spa/` (ao lado do `go.mod`):

```
main.go                 ponto de entrada (flags, servidor, abre o navegador)
internal/config         .env e variáveis
internal/params         parâmetros, período, JQL
internal/jira           cliente REST (paginação, retry, worklogs em lote)
internal/store          SQLite (schema, CRUD, sincronizações)
internal/coleta         coleta por bloco + runner em segundo plano
internal/equipe         regras de pertencimento à Equipe
internal/metrics        cálculo de T1/T2/T3 (Estoque = snapshot; TOT = último mês)
internal/mapaslide      marcadores {{ESP_T3_E_MES_n}} / TOT
internal/google         OAuth, Drive e Slides
internal/slide          cópia do modelo, replaceAllText, {{ESP_Tn}} → print do cartão
internal/graficopng     PNG de T1/T2/T3 (T3 inclui a série Estoque)
internal/web            handlers, templates, gráficos SVG, CSV, API, estáticos
internal/demo           dados fictícios
internal/jiratest       Jira falso para testes
internal/googletest     Google falso (OAuth/Drive/Slides) para testes
```
