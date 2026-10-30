#!/bin/bash
# Executa lint e format conforme a stack do arquivo editado

FILE="$1"

run_node_lint() {
  local target="$1"
  if [ -n "$target" ]; then
    npx eslint "$target" --fix 2>/dev/null || true
    npx prettier --write "$target" 2>/dev/null || true
  else
    npx eslint . --fix 2>/dev/null || true
    npx prettier --write . 2>/dev/null || true
  fi
}

run_go_lint() {
  local target="$1"
  if command -v golangci-lint >/dev/null 2>&1; then
    if [ -n "$target" ]; then
      golangci-lint run --fix "$target" 2>/dev/null || true
    else
      golangci-lint run --fix 2>/dev/null || true
    fi
  fi
  if [ -n "$target" ]; then
    gofmt -w "$target" 2>/dev/null || true
  else
    gofmt -w . 2>/dev/null || true
  fi
}

if [ -z "$FILE" ]; then
  if [ -f "package.json" ]; then
    run_node_lint ""
  elif [ -f "go.mod" ]; then
    run_go_lint ""
  fi
  exit 0
fi

case "$FILE" in
  *.ts|*.tsx|*.js|*.jsx)
    run_node_lint "$FILE"
    ;;
  *.go)
    run_go_lint "$FILE"
    ;;
  *)
    exit 0
    ;;
esac

exit 0
