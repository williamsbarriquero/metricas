# Monta -ldflags com JIRA_URL, JIRA_EMAIL e JIRA_TOKEN para gravar no executável.
# Prioridade: variável de ambiente já exportada, depois o arquivo .env.
# Define LDFLAGS_JIRA. Não imprime os valores.
# shellcheck shell=bash

_valor_dotenv() {
  local chave="$1" arquivo="$2"
  [[ -f "$arquivo" ]] || return 0
  awk -v chave="$chave" '
    BEGIN { valor = "" }
    {
      line = $0
      gsub(/\r$/, "", line)
      gsub(/^[ \t]+|[ \t]+$/, "", line)
      if (line == "" || line ~ /^#/) next
      sub(/^export[ \t]+/, "", line)
      eq = index(line, "=")
      if (eq == 0) next
      k = substr(line, 1, eq - 1)
      gsub(/[ \t]+$/, "", k)
      if (k != chave) next
      v = substr(line, eq + 1)
      gsub(/^[ \t]+|[ \t]+$/, "", v)
      n = length(v)
      if (n >= 2 && (substr(v, 1, 1) == "\"" && substr(v, n, 1) == "\"" || substr(v, 1, 1) == "\047" && substr(v, n, 1) == "\047")) {
        v = substr(v, 2, n - 2)
      }
      valor = v
    }
    END { printf "%s", valor }
  ' "$arquivo"
}

_jira_ldflags_seguro() {
  local nome="$1" valor="$2"
  case "$valor" in
    *[[:space:]]*|*[\"\'\$\`\\]*)
      echo "Não foi possível gravar ${nome} no executável: o valor contém espaço ou caractere que quebra o -ldflags." >&2
      return 1
      ;;
  esac
}

# ldflags_jira ARQUIVO_ENV
ldflags_jira() {
  local arquivo="$1"
  local url="${JIRA_URL-}"
  local email="${JIRA_EMAIL-}"
  local token="${JIRA_TOKEN-}"
  [[ -n "$url" ]] || url="$(_valor_dotenv JIRA_URL "$arquivo")"
  [[ -n "$email" ]] || email="$(_valor_dotenv JIRA_EMAIL "$arquivo")"
  [[ -n "$token" ]] || token="$(_valor_dotenv JIRA_TOKEN "$arquivo")"

  _jira_ldflags_seguro JIRA_URL "$url" || return 1
  _jira_ldflags_seguro JIRA_EMAIL "$email" || return 1
  _jira_ldflags_seguro JIRA_TOKEN "$token" || return 1

  local pkg="metricas-coordenadores/internal/config"
  LDFLAGS_JIRA=""
  [[ -n "$url" ]] && LDFLAGS_JIRA+=" -X ${pkg}.embutidoJiraURL=${url}"
  [[ -n "$email" ]] && LDFLAGS_JIRA+=" -X ${pkg}.embutidoJiraEmail=${email}"
  [[ -n "$token" ]] && LDFLAGS_JIRA+=" -X ${pkg}.embutidoJiraToken=${token}"
}
