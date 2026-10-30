#!/bin/bash
# RPE Harness - Update Script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
# shellcheck source=lib/stack-manifest.sh
source "$SCRIPT_DIR/lib/stack-manifest.sh"

ALL_STACKS_FLAG=false
FLAG_JAVA=false
FLAG_GO=false
FLAG_NODE=false
FLAG_REACT=false
STACK_LIST=""
TARGET_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --all-stacks)
      ALL_STACKS_FLAG=true
      shift
      ;;
    --java)
      FLAG_JAVA=true
      shift
      ;;
    --go)
      FLAG_GO=true
      shift
      ;;
    --node)
      FLAG_NODE=true
      shift
      ;;
    --react)
      FLAG_REACT=true
      shift
      ;;
    --stack)
      if [ -z "${2:-}" ]; then
        echo "Erro: --stack requer uma lista (ex: java,react)" >&2
        exit 1
      fi
      STACK_LIST="$2"
      shift 2
      ;;
    -h|--help)
      echo "Uso: $0 [opções] [target-dir]"
      echo "  Repassa flags de stack para install.sh."
      echo "  Sem flags de stack, lê .cursor/harness-stacks.json do projeto alvo."
      exit 0
      ;;
    *)
      if [ -z "$TARGET_DIR" ]; then
        TARGET_DIR="$1"
      else
        echo "Argumento inesperado: $1" >&2
        exit 1
      fi
      shift
      ;;
  esac
done

TARGET_DIR="${TARGET_DIR:-.}"

echo "🔄 Atualizando RPE Harness em $TARGET_DIR..."

if [ -d "$ROOT_DIR/.git" ]; then
  echo "Fazendo pull das últimas atualizações do kit fonte..."
  git -C "$ROOT_DIR" pull origin main 2>/dev/null || git -C "$ROOT_DIR" pull 2>/dev/null || echo "Aviso: pull não executado."
fi

INSTALL_ARGS=()

if [ "$ALL_STACKS_FLAG" = true ]; then
  INSTALL_ARGS+=(--all-stacks)
elif [ "$FLAG_JAVA" = true ] || [ "$FLAG_GO" = true ] || [ "$FLAG_NODE" = true ] || [ "$FLAG_REACT" = true ] || [ -n "$STACK_LIST" ]; then
  [ "$FLAG_JAVA" = true ] && INSTALL_ARGS+=(--java)
  [ "$FLAG_GO" = true ] && INSTALL_ARGS+=(--go)
  [ "$FLAG_NODE" = true ] && INSTALL_ARGS+=(--node)
  [ "$FLAG_REACT" = true ] && INSTALL_ARGS+=(--react)
  if [ -n "$STACK_LIST" ]; then
    INSTALL_ARGS+=(--stack "$STACK_LIST")
  fi
else
  HARNESS_STACKS_FILE="$TARGET_DIR/.cursor/harness-stacks.json"
  if read_harness_stacks_json "$HARNESS_STACKS_FILE" 2>/dev/null; then
    echo "ℹ️  Reaplicando stacks de harness-stacks.json: ${SELECTED_STACKS[*]}"
    for stack in "${SELECTED_STACKS[@]}"; do
      INSTALL_ARGS+=(--"$stack")
    done
  else
    echo "ℹ️  harness-stacks.json ausente — update instalará todas as stacks"
  fi
fi

bash "$ROOT_DIR/scripts/install.sh" "${INSTALL_ARGS[@]}" "$TARGET_DIR"

echo "✅ Harness atualizado. mcp.json local foi preservado (não sobrescrito)."
