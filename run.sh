#!/usr/bin/env bash
# Compila o React dentro do binário Go e sobe um único processo.
set -euo pipefail
RAIZ="$(cd "$(dirname "$0")" && pwd)"

node_serve() {
  "$1" -e 'const [M,m]=process.versions.node.split(".").map(Number); process.exit((M===20 && m>=19) || M>=22 ? 0 : 1)'
}
if ! command -v node >/dev/null || ! node_serve node; then
  escolhido=""
  for candidato in "$HOME"/.nvm/versions/node/*/bin/node; do
    [[ -x "$candidato" ]] || continue
    if node_serve "$candidato"; then
      escolhido="$candidato"
    fi
  done
  if [[ -z "$escolhido" ]]; then
    echo "O Vite precisa do Node 20.19+ ou 22.12+. Atual: $(node -v 2>/dev/null || echo nenhum)." >&2
    echo "Com o nvm: nvm install 22" >&2
    exit 1
  fi
  echo "Node do shell é antigo ($(node -v 2>/dev/null || echo nenhum)); usando $escolhido."
  export PATH="$(dirname "$escolhido"):$PATH"
  hash -r
fi

cd "$RAIZ/frontend"
if [[ ! -d node_modules ]]; then
  npm install
fi
npm run build

rm -rf "$RAIZ/cmd/spa/dist"
mkdir -p "$RAIZ/cmd/spa/dist"
cp -a dist/. "$RAIZ/cmd/spa/dist/"

cd "$RAIZ"
# shellcheck source=scripts/lib/ldflags-jira.sh
source "$RAIZ/scripts/lib/ldflags-jira.sh"
ldflags_jira "$RAIZ/.env"
if [[ "$LDFLAGS_JIRA" != *embutidoJiraEmail* || "$LDFLAGS_JIRA" != *embutidoJiraToken* ]]; then
  echo "Aviso: JIRA_EMAIL ou JIRA_TOKEN ausentes. O executável sobe, mas a coleta falha até preencher o .env." >&2
fi
go build -ldflags "-s -w${LDFLAGS_JIRA}" -o "$RAIZ/metricas-spa" ./cmd/spa

if [[ ! -f "$RAIZ/.env" ]]; then
  cp "$RAIZ/.env.example" "$RAIZ/.env"
fi
cd "$RAIZ"
exec ./metricas-spa "$@"
