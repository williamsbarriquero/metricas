# Métricas dos Coordenadores — React e Go num único processo

Front em React, Tailwind e shadcn/ui. O mesmo binário Go serve essas telas e a API. CSV e o login do Google continuam em rotas do Go. No pod do EKS roda só esse processo: o Node entra apenas na hora de compilar a imagem.

O `go.mod` fica nesta pasta, junto de `main.go`, `cmd/` e `internal/`. O app só em HTML sobe com `go run ..` a partir de `golang/` (porta 8080, banco em `golang/data`). Esse entrypoint continua suportado para uso local e não entra na imagem. O React usa `data/` e, por padrão, a porta **8090**.

## Rodar na máquina

Um processo só (o que vai para o EKS):

```bash
cd spa
cp .env.example .env          # JIRA_EMAIL e JIRA_TOKEN, se for coletar
./run.sh                      # abre http://127.0.0.1:8090
./run.sh -demo                # dados fictícios, sem Jira
```

Desenvolvimento, com recarga do React:

```bash
cd spa
./dev.sh                      # API em :8090, Vite em http://localhost:5173
./dev.sh -demo
```

`./dev.sh` e `./run.sh` criam o `.env` na primeira vez. Reinicie o servidor depois de mudar o `.env`.

## Deploy

Na raiz do repositório:

```bash
docker build -f spa/Dockerfile -t metricas-spa .
docker run --rm -p 8080:8080 -v metricas-data:/app/data -e JIRA_EMAIL -e JIRA_TOKEN metricas-spa
```

A imagem final não tem Node. `ADDR` na imagem é `0.0.0.0:8080` para o container receber o Service. O processo não autentica usuários e não vai ganhar login: na máquina a escuta padrão é `127.0.0.1`; quem publicar o pod mantém o Service fora da internet pública, com autenticação na borda do cluster. Não há manifesto de Ingress neste repositório.

O print do cartão no Slide abre o dashboard React (`/print`) num Chrome da máquina e fotografa esse cartão; a imagem não traz o Chrome, para continuar pequena. Em `./dev.sh` o Chrome usa o Vite (`SLIDE_ORIGEM`, padrão `http://127.0.0.1:5173`). No processo único, usa o próprio painel.

## O que é cada tela

| Caminho | Quem desenha |
| --- | --- |
| `/`, `/metricas`, `/mapa-slide`, `/dados`, `/equipe`, `/configuracoes`, `/sincronizacao`, `/glossario` | React (shadcn) |
| `/csv`, `/google` | Download e login OAuth no Go |
