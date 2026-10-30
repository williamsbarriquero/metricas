#!/bin/bash
# Bloqueia operações MCP destrutivas (SQL, deletes em massa)

PAYLOAD="$1"

if [ -z "$PAYLOAD" ]; then
  exit 0
fi

BLOCKED_PATTERNS=(
  'DROP\s+(DATABASE|TABLE|SCHEMA)'
  'TRUNCATE\s+TABLE'
  'DELETE\s+FROM\s+\w+\s*;'
  'ALTER\s+TABLE\s+\w+\s+DROP'
)

for pattern in "${BLOCKED_PATTERNS[@]}"; do
  if echo "$PAYLOAD" | grep -qiE "$pattern"; then
    echo "❌ BLOCKED: Operação MCP destrutiva detectada."
    exit 1
  fi
done

exit 0
