#!/bin/bash
# RPE Harness - Install Script

set -e

SKIP_BUN=false
ALL_STACKS_FLAG=false
FLAG_JAVA=false
FLAG_GO=false
FLAG_NODE=false
FLAG_REACT=false
STACK_LIST=""
TARGET_DIR=""
MODULES="all"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/stack-manifest.sh
source "$SCRIPT_DIR/lib/stack-manifest.sh"

usage() {
  echo "Uso: $0 [opções] <target-dir> [modules]"
  echo ""
  echo "Stacks (combináveis):"
  echo "  --java        Padrões Java (java-standards, java-mastery, knowledge/java)"
  echo "  --go          Padrões Go"
  echo "  --node        Padrões Node.js backend"
  echo "  --react       Frontend/Next.js (frontend-standards + frontend-mastery + knowledge/frontend)"
  echo "  --stack LIST  Atalho: --stack java,react"
  echo "  --all-stacks  Instala todas as stacks (padrão se nenhuma flag de stack)"
  echo ""
  echo "Outras opções:"
  echo "  --skip-bun    Não instala Bun (grind-loop / UltraWork requer Bun quando hooks incluídos)"
  echo ""
  echo "Módulos: all, rules, agents, commands, hooks, skills, knowledge, mcp, scripts"
  echo ""
  echo "Exemplos:"
  echo "  $0 --java /projeto"
  echo "  $0 --java --react /monorepo"
  echo "  $0 --stack java,react /projeto"
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-bun)
      SKIP_BUN=true
      shift
      ;;
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
        usage
      fi
      STACK_LIST="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    *)
      if [ -z "$TARGET_DIR" ]; then
        TARGET_DIR="$1"
      elif [ "$MODULES" = "all" ]; then
        MODULES="$1"
      else
        echo "Argumento inesperado: $1" >&2
        usage
      fi
      shift
      ;;
  esac
done

if [ -z "$TARGET_DIR" ]; then
  usage
fi

ROOT_DIR="$(dirname "$SCRIPT_DIR")"

resolve_stacks_from_flags
resolve_install_sets

echo "🚀 Instalando RPE Harness em $TARGET_DIR..."
echo "📦 Stacks: ${SELECTED_STACKS[*]} (mode: $STACK_MODE)"

ensure_bun() {
  if command -v bun >/dev/null 2>&1; then
    echo "✅ Bun já disponível: $(bun --version)"
    return 0
  fi

  echo "📦 Instalando Bun (necessário para grind-loop / UltraWork)..."
  if ! command -v curl >/dev/null 2>&1; then
    echo "⚠️ curl não encontrado — instale Bun manualmente: https://bun.sh"
    return 1
  fi

  curl -fsSL https://bun.sh/install | bash

  export BUN_INSTALL="${BUN_INSTALL:-$HOME/.bun}"
  export PATH="$BUN_INSTALL/bin:$PATH"

  if command -v bun >/dev/null 2>&1; then
    echo "✅ Bun instalado: $(bun --version)"
    return 0
  fi

  echo "⚠️ Bun instalado em $BUN_INSTALL/bin, mas não está no PATH desta sessão."
  echo "   Execute: export PATH=\"\$HOME/.bun/bin:\$PATH\""
  echo "   Ou reinicie o terminal."
  return 1
}

hooks_will_be_installed() {
  if [ "$MODULES" = "all" ]; then
    return 0
  fi

  IFS=',' read -ra MOD_LIST <<< "$MODULES"
  for mod in "${MOD_LIST[@]}"; do
    mod="$(trim "$mod")"
    if [ "$mod" = "hooks" ]; then
      return 0
    fi
  done
  return 1
}

install_rules() {
  mkdir -p "$TARGET_DIR/.cursor/rules"
  local rule
  for rule in "${INSTALL_RULES[@]}"; do
    cp "$ROOT_DIR/.cursor/rules/$rule" "$TARGET_DIR/.cursor/rules/"
  done
  echo "   Rules: ${#INSTALL_RULES[@]} arquivo(s)"
}

install_skills() {
  mkdir -p "$TARGET_DIR/.cursor/skills"
  local skill
  for skill in "${INSTALL_SKILLS[@]}"; do
    cp -r "$ROOT_DIR/.cursor/skills/$skill" "$TARGET_DIR/.cursor/skills/"
  done
  echo "   Skills: ${#INSTALL_SKILLS[@]} skill(s)"
}

install_knowledge() {
  mkdir -p "$TARGET_DIR/.cursor/knowledge/stacks"
  local file
  for file in "${INSTALL_KNOWLEDGE[@]}"; do
    local dest_dir
    dest_dir="$(dirname "$TARGET_DIR/.cursor/knowledge/$file")"
    mkdir -p "$dest_dir"
    cp "$ROOT_DIR/.cursor/knowledge/$file" "$TARGET_DIR/.cursor/knowledge/$file"
  done
  echo "   Knowledge: ${#INSTALL_KNOWLEDGE[@]} arquivo(s)"
}

install_specs_and_templates() {
  mkdir -p "$TARGET_DIR/.cursor/docs/specs"
  mkdir -p "$TARGET_DIR/.cursor/templates"
  cp "$ROOT_DIR/.cursor/docs/specs/README.md" "$TARGET_DIR/.cursor/docs/specs/README.md"
  cp \
    "$ROOT_DIR/templates/feature-spec.md" \
    "$ROOT_DIR/templates/technical-spec.md" \
    "$ROOT_DIR/templates/definition-of-done.md" \
    "$TARGET_DIR/.cursor/templates/"
  echo "   Specs: .cursor/docs/specs/ + templates (feature-spec, technical-spec)"
}

install_module() {
  local module="$1"
  case "$module" in
    rules)
      install_rules
      ;;
    agents)
      mkdir -p "$TARGET_DIR/.cursor/agents"
      cp -r "$ROOT_DIR/.cursor/agents/"* "$TARGET_DIR/.cursor/agents/"
      ;;
    commands)
      mkdir -p "$TARGET_DIR/.cursor/commands"
      cp -r "$ROOT_DIR/.cursor/commands/"* "$TARGET_DIR/.cursor/commands/"
      ;;
    hooks)
      mkdir -p "$TARGET_DIR/.cursor/hooks"
      cp -r "$ROOT_DIR/.cursor/hooks/"* "$TARGET_DIR/.cursor/hooks/"
      cp "$ROOT_DIR/.cursor/hooks.json" "$TARGET_DIR/.cursor/"
      chmod +x "$TARGET_DIR/.cursor/hooks/"*/*.sh 2>/dev/null || true
      ;;
    skills)
      install_skills
      ;;
    knowledge)
      install_knowledge
      ;;
    mcp)
      mkdir -p "$TARGET_DIR/.cursor"
      if [ ! -f "$TARGET_DIR/.cursor/mcp.json" ]; then
        cp "$ROOT_DIR/.cursor/mcp.json.example" "$TARGET_DIR/.cursor/mcp.json"
        echo "ℹ️  mcp.json criado a partir do template. Configure variáveis de ambiente."
      else
        echo "ℹ️  mcp.json já existe — não sobrescrito."
      fi
      ;;
    scripts)
      mkdir -p "$TARGET_DIR/scripts/lib"
      cp "$ROOT_DIR/scripts/install.sh" "$ROOT_DIR/scripts/validate.sh" "$ROOT_DIR/scripts/update.sh" "$TARGET_DIR/scripts/"
      cp "$ROOT_DIR/scripts/lib/stack-manifest.sh" "$TARGET_DIR/scripts/lib/"
      chmod +x "$TARGET_DIR/scripts/"*.sh
      ;;
  esac
}

if [ "$MODULES" = "all" ]; then
  for mod in rules agents commands hooks skills knowledge mcp scripts; do
    install_module "$mod"
  done
  cp "$ROOT_DIR/agents.md" "$TARGET_DIR/"
  mkdir -p "$TARGET_DIR/.cursor"
  cp "$ROOT_DIR/.cursor/scratchpad.template.md" "$TARGET_DIR/.cursor/scratchpad.template.md"
  if [ ! -f "$TARGET_DIR/.cursor/scratchpad.md" ]; then
    cp "$ROOT_DIR/.cursor/scratchpad.template.md" "$TARGET_DIR/.cursor/scratchpad.md"
  fi
  install_specs_and_templates
  write_harness_stacks_json "$TARGET_DIR"
else
  SHOULD_WRITE_STACKS=false
  IFS=',' read -ra MOD_LIST <<< "$MODULES"
  for mod in "${MOD_LIST[@]}"; do
    mod="$(trim "$mod")"
    install_module "$mod"
    case "$mod" in
      rules|skills|knowledge) SHOULD_WRITE_STACKS=true ;;
    esac
  done
  if [ "$SHOULD_WRITE_STACKS" = true ]; then
    write_harness_stacks_json "$TARGET_DIR"
  fi
fi

if hooks_will_be_installed; then
  if [ "$SKIP_BUN" = true ]; then
    echo "ℹ️  Instalação do Bun ignorada (--skip-bun). grind-loop requer Bun no PATH."
  else
    ensure_bun || echo "ℹ️  UltraWork (grind-loop) requer Bun no PATH — instale manualmente ou reexecute install.sh."
  fi
fi

echo "✅ Harness instalado com sucesso!"
