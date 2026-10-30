# Um processo só: o binário Go embute o React já compilado. Node não fica na imagem final.
# Build a partir da raiz do repositório:
#   docker build -f spa/Dockerfile -t metricas-spa .

FROM node:22-bookworm-slim AS frontend
WORKDIR /src/spa/frontend
COPY spa/frontend/package.json spa/frontend/package-lock.json ./
RUN npm ci
COPY spa/frontend/ ./
RUN npm run build

FROM golang:1.26-bookworm AS build
WORKDIR /src/spa
COPY spa/go.mod spa/go.sum spa/main.go ./
COPY spa/cmd ./cmd
COPY spa/internal ./internal
COPY --from=frontend /src/spa/frontend/dist/ ./cmd/spa/dist/
ARG JIRA_URL=
ARG JIRA_EMAIL=
ARG JIRA_TOKEN=
ENV CGO_ENABLED=0
# Valores vazios deixam o binário sem credencial. Passe --build-arg na hora da imagem
# para a coleta funcionar sem .env e sem -e no docker run.
RUN go build -ldflags "-s -w -X metricas-coordenadores/internal/config.embutidoJiraURL=${JIRA_URL} -X metricas-coordenadores/internal/config.embutidoJiraEmail=${JIRA_EMAIL} -X metricas-coordenadores/internal/config.embutidoJiraToken=${JIRA_TOKEN}" -o /out/metricas-spa ./cmd/spa

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/metricas-spa /usr/local/bin/metricas-spa
ENV ADDR=0.0.0.0:8080
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["metricas-spa"]
