#!/bin/bash
# RPE Harness - Stack manifest (SSOT de artefatos por stack)

ALL_STACKS=(java go node react)

HARNESS_CORE_RULES=(
  rpe-identity.mdc
  interaction-standards.mdc
  coding-standards.mdc
  intent-routing.mdc
  safety-guardrails.mdc
  stack-baseline.mdc
  tdd-workflow.mdc
  code-review.mdc
  git-conventions.mdc
  ultrawork.mdc
  testing-standards.mdc
  qa-standards.mdc
  database-migrations.mdc
  observability.mdc
)

STACK_RULES_java=(java-standards.mdc)
STACK_RULES_go=(go-standards.mdc)
STACK_RULES_node=(node-standards.mdc typescript.mdc)
STACK_RULES_react=(frontend-standards.mdc typescript.mdc)

STACK_SKILLS_java=(java-mastery)
STACK_SKILLS_go=(go-mastery)
STACK_SKILLS_node=(node-mastery)
STACK_SKILLS_react=(frontend-mastery)

STACK_KNOWLEDGE_java=(stacks/java.md)
STACK_KNOWLEDGE_go=(stacks/go.md)
STACK_KNOWLEDGE_node=(stacks/node.md)
STACK_KNOWLEDGE_react=(stacks/frontend.md)

UNIVERSAL_SKILLS=(
  api-contract-testing
  accessibility-visual-testing
  bdd-gherkin
  design-patterns
  e2e-testing
  hexagonal-architecture
  lsp-integration
  mcp-integration
  performance-testing
  pipeline-gateway
  session-persistence
  solid-principles
  tdd-grinder
)

is_valid_stack() {
  local stack="$1"
  case "$stack" in
    java|go|node|react) return 0 ;;
    *) return 1 ;;
  esac
}

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  echo "$value"
}

add_stack_flag() {
  local stack
  stack="$(trim "$(echo "$1" | tr '[:upper:]' '[:lower:]')")"
  if ! is_valid_stack "$stack"; then
    echo "Stack inválida: $stack (use: java, go, node, react)" >&2
    return 1
  fi
  for existing in "${SELECTED_STACKS[@]}"; do
    if [ "$existing" = "$stack" ]; then
      return 0
    fi
  done
  SELECTED_STACKS+=("$stack")
}

expand_stack_list() {
  local list="$1"
  IFS=',' read -ra PARTS <<< "$list"
  for part in "${PARTS[@]}"; do
    add_stack_flag "$part" || return 1
  done
}

resolve_stacks_from_flags() {
  SELECTED_STACKS=()
  STACK_MODE="all"

  if [ "$ALL_STACKS_FLAG" = true ]; then
    SELECTED_STACKS=("${ALL_STACKS[@]}")
    STACK_MODE="all"
    return 0
  fi

  if [ "$FLAG_JAVA" = true ]; then add_stack_flag java; fi
  if [ "$FLAG_GO" = true ]; then add_stack_flag go; fi
  if [ "$FLAG_NODE" = true ]; then add_stack_flag node; fi
  if [ "$FLAG_REACT" = true ]; then add_stack_flag react; fi

  if [ -n "$STACK_LIST" ]; then
    expand_stack_list "$STACK_LIST" || return 1
  fi

  if [ "${#SELECTED_STACKS[@]}" -eq 0 ]; then
    SELECTED_STACKS=("${ALL_STACKS[@]}")
    STACK_MODE="all"
    echo "ℹ️  Nenhuma stack informada — instalando todas (java, go, node, react)."
    echo "   Use --java, --go, --node, --react ou --stack java,react para instalação seletiva."
    return 0
  fi

  STACK_MODE="explicit"
}

get_stack_rules() {
  local stack="$1"
  case "$stack" in
    java) printf '%s\n' "${STACK_RULES_java[@]}" ;;
    go) printf '%s\n' "${STACK_RULES_go[@]}" ;;
    node) printf '%s\n' "${STACK_RULES_node[@]}" ;;
    react) printf '%s\n' "${STACK_RULES_react[@]}" ;;
  esac
}

get_stack_skills() {
  local stack="$1"
  case "$stack" in
    java) [ "${#STACK_SKILLS_java[@]}" -gt 0 ] && printf '%s\n' "${STACK_SKILLS_java[@]}" ;;
    go) [ "${#STACK_SKILLS_go[@]}" -gt 0 ] && printf '%s\n' "${STACK_SKILLS_go[@]}" ;;
    node) [ "${#STACK_SKILLS_node[@]}" -gt 0 ] && printf '%s\n' "${STACK_SKILLS_node[@]}" ;;
    react) [ "${#STACK_SKILLS_react[@]}" -gt 0 ] && printf '%s\n' "${STACK_SKILLS_react[@]}" ;;
  esac
}

get_stack_knowledge() {
  local stack="$1"
  case "$stack" in
    java) printf '%s\n' "${STACK_KNOWLEDGE_java[@]}" ;;
    go) printf '%s\n' "${STACK_KNOWLEDGE_go[@]}" ;;
    node) printf '%s\n' "${STACK_KNOWLEDGE_node[@]}" ;;
    react) printf '%s\n' "${STACK_KNOWLEDGE_react[@]}" ;;
  esac
}

array_contains() {
  local needle="$1"
  shift
  local item
  for item in "$@"; do
    if [ "$item" = "$needle" ]; then
      return 0
    fi
  done
  return 1
}

resolve_install_rules() {
  INSTALL_RULES=("${HARNESS_CORE_RULES[@]}")
  local stack rule

  for stack in "${SELECTED_STACKS[@]}"; do
    while IFS= read -r rule; do
      [ -z "$rule" ] && continue
      if ! array_contains "$rule" "${INSTALL_RULES[@]}"; then
        INSTALL_RULES+=("$rule")
      fi
    done < <(get_stack_rules "$stack")
  done
}

resolve_install_skills() {
  INSTALL_SKILLS=("${UNIVERSAL_SKILLS[@]}")
  local stack skill

  for stack in "${SELECTED_STACKS[@]}"; do
    while IFS= read -r skill; do
      [ -z "$skill" ] && continue
      if ! array_contains "$skill" "${INSTALL_SKILLS[@]}"; then
        INSTALL_SKILLS+=("$skill")
      fi
    done < <(get_stack_skills "$stack")
  done
}

resolve_install_knowledge() {
  INSTALL_KNOWLEDGE=(README.md)
  local stack file

  for stack in "${SELECTED_STACKS[@]}"; do
    while IFS= read -r file; do
      [ -z "$file" ] && continue
      if ! array_contains "$file" "${INSTALL_KNOWLEDGE[@]}"; then
        INSTALL_KNOWLEDGE+=("$file")
      fi
    done < <(get_stack_knowledge "$stack")
  done
}

resolve_install_sets() {
  resolve_install_rules
  resolve_install_skills
  resolve_install_knowledge
}

stacks_to_json_array() {
  local json="["
  local first=true
  for stack in "${SELECTED_STACKS[@]}"; do
    if [ "$first" = true ]; then
      first=false
    else
      json+=","
    fi
    json+="\"$stack\""
  done
  json+="]"
  echo "$json"
}

write_harness_stacks_json() {
  local target_dir="$1"
  local json_file="$target_dir/.cursor/harness-stacks.json"
  local stacks_json
  stacks_json="$(stacks_to_json_array)"
  local installed_at
  installed_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u)"

  mkdir -p "$target_dir/.cursor"
  cat > "$json_file" <<EOF
{
  "version": 1,
  "installedAt": "$installed_at",
  "mode": "$STACK_MODE",
  "stacks": $stacks_json
}
EOF
}

read_harness_stacks_json() {
  local json_file="$1"
  SELECTED_STACKS=()
  STACK_MODE="all"

  if [ ! -f "$json_file" ]; then
    return 1
  fi

  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 necessário para ler $json_file" >&2
    return 1
  fi

  local parsed
  parsed="$(python3 - "$json_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    data = json.load(handle)

mode = data.get("mode", "all")
stacks = data.get("stacks", [])
print(mode)
print(",".join(stacks))
PY
)"

  STACK_MODE="$(echo "$parsed" | sed -n '1p')"
  local stacks_csv
  stacks_csv="$(echo "$parsed" | sed -n '2p')"

  if [ -z "$stacks_csv" ]; then
    SELECTED_STACKS=("${ALL_STACKS[@]}")
    return 0
  fi

  IFS=',' read -ra SELECTED_STACKS <<< "$stacks_csv"
  return 0
}
