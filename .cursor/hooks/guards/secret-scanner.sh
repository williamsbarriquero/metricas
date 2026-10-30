#!/bin/bash
# Varre working tree e staged files por padrões de credenciais

set -euo pipefail

echo "Scanning for secrets..."

PATTERN='(password|secret|api[_-]?key|token|AKIA[A-Z0-9]{16})\s*[:=]\s*['\''"][^'\''"]+['\''"]'

MATCHES=""

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  STAGED=$(git diff --cached -i -E 2>/dev/null | grep -E "$PATTERN" || true)
  UNSTAGED=$(git diff -i -E 2>/dev/null | grep -E "^\+.*" | grep -E "$PATTERN" || true)
  MATCHES="${STAGED}${UNSTAGED}"
else
  MATCHES=$(grep -rEi "$PATTERN" --include="*.ts" --include="*.js" --include="*.go" --include="*.java" --include="*.env*" --include="*.yml" --include="*.yaml" --include="*.json" . 2>/dev/null | grep -v "node_modules" | grep -v ".git" || true)
fi

if [ -n "$MATCHES" ]; then
  echo "❌ ERROR: Potential secrets detected!"
  echo "$MATCHES"
  exit 1
fi

echo "✅ No secrets found."
exit 0
