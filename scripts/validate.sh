#!/bin/bash
# RPE Harness - Validator

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/stack-manifest.sh
source "$SCRIPT_DIR/lib/stack-manifest.sh"

echo "🔍 Validando integridade do RPE Harness..."

ERRORS=0
WARNINGS=0

check_dir() {
  if [ ! -d "$1" ]; then
    echo "❌ Diretório ausente: $1"
    ERRORS=$((ERRORS+1))
  else
    echo "✅ Diretório encontrado: $1"
  fi
}

check_file() {
  if [ ! -f "$1" ]; then
    echo "❌ Arquivo ausente: $1"
    ERRORS=$((ERRORS+1))
  else
    echo "✅ Arquivo encontrado: $1"
  fi
}

HARNESS_STACKS_FILE=".cursor/harness-stacks.json"
IS_KIT_SOURCE=false

if [ ! -f "$HARNESS_STACKS_FILE" ]; then
  IS_KIT_SOURCE=true
  SELECTED_STACKS=("${ALL_STACKS[@]}")
  echo "ℹ️  harness-stacks.json ausente — validação completa (modo kit fonte)"
else
  read_harness_stacks_json "$HARNESS_STACKS_FILE"
  echo "ℹ️  Stacks instaladas: ${SELECTED_STACKS[*]} (mode: $STACK_MODE)"
fi

resolve_install_sets

check_dir ".cursor/rules"
check_dir ".cursor/agents"
check_dir ".cursor/hooks"
check_dir ".cursor/skills"
check_dir ".cursor/knowledge"
check_dir ".cursor/commands"

for rule in "${INSTALL_RULES[@]}"; do
  check_file ".cursor/rules/$rule"
done

for skill in "${INSTALL_SKILLS[@]}"; do
  check_dir ".cursor/skills/$skill"
done

for file in "${INSTALL_KNOWLEDGE[@]}"; do
  check_file ".cursor/knowledge/$file"
done

check_file ".cursor/hooks.json"
check_file "agents.md"
check_file "scripts/validate.sh"
check_file "scripts/lib/stack-manifest.sh"
check_file ".cursor/scratchpad.template.md"
check_file ".cursor/hooks/continuations/grind-loop.ts"
check_dir ".cursor/docs/specs"
check_file ".cursor/docs/specs/README.md"

# No kit fonte os templates ficam em templates/. O install copia para .cursor/templates/ no projeto alvo.
if [ "$IS_KIT_SOURCE" = true ]; then
  check_file "templates/feature-spec.md"
  check_file "templates/technical-spec.md"
  check_file "templates/definition-of-done.md"
else
  check_dir ".cursor/templates"
  check_file ".cursor/templates/feature-spec.md"
  check_file ".cursor/templates/technical-spec.md"
  check_file ".cursor/templates/definition-of-done.md"
fi

if [ "$IS_KIT_SOURCE" = true ]; then
  check_file ".cursor/mcp.json.example"
fi

AGENT_COUNT=$(find .cursor/agents -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
if [ "$AGENT_COUNT" -lt 10 ]; then
  echo "⚠️ Esperados ≥10 agentes, encontrados: $AGENT_COUNT"
  WARNINGS=$((WARNINGS+1))
else
  echo "✅ Agentes: $AGENT_COUNT"
fi

COMMAND_COUNT=$(find .cursor/commands -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
echo "✅ Commands: $COMMAND_COUNT"

for script in .cursor/hooks/guards/*.sh .cursor/hooks/transforms/*.sh; do
  if [ -f "$script" ] && [ ! -x "$script" ]; then
    echo "⚠️ Script sem permissão de execução: $script"
    WARNINGS=$((WARNINGS+1))
  fi
done

if command -v bun >/dev/null 2>&1; then
  echo "✅ Bun disponível para grind-loop.ts"
else
  echo "⚠️ Bun não encontrado — grind-loop.ts pode falhar"
  echo "   Instale via: bash scripts/install.sh . hooks (ou curl -fsSL https://bun.sh/install | bash)"
  WARNINGS=$((WARNINGS+1))
fi

if grep -q 'rules/core/' .cursor/agents/*.md 2>/dev/null; then
  echo "❌ Referências obsoletas a rules/core/ encontradas nos agentes"
  ERRORS=$((ERRORS+1))
fi

MCP_FILE=".cursor/mcp.json"
MCP_EXAMPLE=".cursor/mcp.json.example"

if [ -f "$MCP_FILE" ]; then
  if python3 -c "import json; json.load(open('$MCP_FILE'))" 2>/dev/null; then
    echo "✅ mcp.json é JSON válido"
  else
    echo "❌ mcp.json inválido"
    ERRORS=$((ERRORS+1))
  fi
elif [ -f "$MCP_EXAMPLE" ]; then
  if python3 -c "import json; json.load(open('$MCP_EXAMPLE'))" 2>/dev/null; then
    echo "✅ mcp.json.example é JSON válido (mcp.json local ausente — esperado)"
  else
    echo "❌ mcp.json.example inválido"
    ERRORS=$((ERRORS+1))
  fi
elif [ "$IS_KIT_SOURCE" = false ]; then
  echo "ℹ️  mcp.json e mcp.json.example ausentes (opcional no projeto alvo)"
fi

if [ -f "$HARNESS_STACKS_FILE" ]; then
  if python3 -c "import json; json.load(open('$HARNESS_STACKS_FILE'))" 2>/dev/null; then
    echo "✅ harness-stacks.json é JSON válido"
  else
    echo "❌ harness-stacks.json inválido"
    ERRORS=$((ERRORS+1))
  fi
fi

if [ $ERRORS -gt 0 ]; then
  echo "⚠️ $ERRORS erro(s) e $WARNINGS aviso(s) encontrados."
  exit 1
else
  echo "✨ O RPE Harness está íntegro ($WARNINGS aviso(s))."
  exit 0
fi
