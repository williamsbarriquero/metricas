#!/bin/bash
# Bloqueia comandos shell destrutivos ou de alto risco

COMMAND="$1"

if [ -z "$COMMAND" ]; then
  exit 0
fi

BLOCKED_PATTERNS=(
  'rm\s+-rf\s+/'
  'rm\s+-rf\s+~'
  'curl\s+.*\|\s*bash'
  'wget\s+.*\|\s*sh'
  'git\s+push\s+.*--force'
  'git\s+push\s+-f'
  'DROP\s+DATABASE'
  'DROP\s+TABLE'
  'TRUNCATE\s+TABLE'
)

for pattern in "${BLOCKED_PATTERNS[@]}"; do
  if echo "$COMMAND" | grep -qiE "$pattern"; then
    echo "❌ BLOCKED: Comando de alto risco detectado: $COMMAND"
    exit 1
  fi
done

exit 0
