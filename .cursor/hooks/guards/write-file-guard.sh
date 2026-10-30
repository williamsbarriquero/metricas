#!/bin/bash
# Guard hook: protege arquivos críticos contra edição acidental

PROTECTED_BASENAMES=(
  ".env"
  ".env.local"
  ".env.production"
  "package-lock.json"
  "yarn.lock"
  "pnpm-lock.yaml"
  "docker-compose.yml"
  "docker-compose.prod.yml"
)

PROTECTED_PATTERNS=(
  "/.github/workflows/"
  "/Dockerfile"
)

FILE_TO_EDIT="$1"

if [ -z "$FILE_TO_EDIT" ]; then
  exit 0
fi

BASENAME=$(basename "$FILE_TO_EDIT")

for protected in "${PROTECTED_BASENAMES[@]}"; do
  if [ "$BASENAME" = "$protected" ]; then
    echo "⚠️ WARNING: Modificação bloqueada em arquivo protegido: $FILE_TO_EDIT"
    mkdir -p .cursor/hooks/guards/logs
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ): Blocked edit on $FILE_TO_EDIT" >> .cursor/hooks/guards/logs/modifications.log
    exit 1
  fi
done

for pattern in "${PROTECTED_PATTERNS[@]}"; do
  if [[ "$FILE_TO_EDIT" == *"$pattern"* ]]; then
    echo "⚠️ WARNING: Modificação bloqueada em arquivo protegido: $FILE_TO_EDIT"
    mkdir -p .cursor/hooks/guards/logs
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ): Blocked edit on $FILE_TO_EDIT" >> .cursor/hooks/guards/logs/modifications.log
    exit 1
  fi
done

if [[ "$BASENAME" == "package.json" ]] && [[ "$FILE_TO_EDIT" != *"node_modules"* ]]; then
  echo "⚠️ WARNING: Modificação de package.json detectada: $FILE_TO_EDIT"
  echo "Confirme se a alteração de dependências é intencional."
  mkdir -p .cursor/hooks/guards/logs
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ): Warned on package.json edit: $FILE_TO_EDIT" >> .cursor/hooks/guards/logs/modifications.log
  exit 0
fi

exit 0
