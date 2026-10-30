#!/usr/bin/env bash
# Sobe o Go na porta do .env e o Vite com recarga. Dois processos, só nesta máquina.
set -euo pipefail
RAIZ="$(cd "$(dirname "$0")" && pwd)"

if [[ ! -f "$RAIZ/.env" ]]; then
  cp "$RAIZ/.env.example" "$RAIZ/.env"
  echo "Criei spa/.env. A porta local é 8090."
fi

# Vite 8 pede Node 20.19+ ou 22.12+. O shell às vezes deixa um Node 16 na frente.
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

cd "$RAIZ"
# shellcheck source=scripts/lib/ldflags-jira.sh
source "$RAIZ/scripts/lib/ldflags-jira.sh"
ldflags_jira "$RAIZ/.env"
go build -ldflags "${LDFLAGS_JIRA# }" -o "$RAIZ/metricas-spa" ./cmd/spa

cd "$RAIZ"
# O print do slide abre o dashboard que o Vite está servindo, não o HTML legado.
export SLIDE_ORIGEM="${SLIDE_ORIGEM:-http://127.0.0.1:5173}"
./metricas-spa -nao-abrir "$@" &
PID=$!
trap 'kill "$PID" 2>/dev/null || true' EXIT
sleep 0.4
if ! kill -0 "$PID" 2>/dev/null; then
  wait "$PID"
  exit 1
fi

cd "$RAIZ/frontend"
if [[ ! -d node_modules ]]; then
  npm install
fi
npm run dev
