# Validação da Stack

Execute os passos a seguir para validar a saúde da stack e reportar status consolidado.

## Passos

1. Este repositório tem duas stacks de produto: Go e Frontend (React/Vite). Java não se aplica.
2. Executar os comandos de `stack-baseline.mdc` e `agents.md`:

| Stack | Onde | Lint | Type-check | Testes |
|-------|------|------|------------|--------|
| Go | raiz do `go.mod` | `go vet ./...` | coberto por `go vet` / `go test` | `go test ./...` |
| Frontend | `frontend/` | `npm run lint` | `npx tsc -b` | `npm test` |

Formatação Go: `gofmt -w .`. Formatação do front: `npm run fmt:check` (oxfmt).

`./scripts/validate.sh` confere a instalação do harness (pastas `.cursor/`). Não entra no gate do produto.

3. Reportar status consolidado por comando (✅ exit 0 / ❌ exit ≠ 0).
4. Se algum check falhar, listar erros e sugerir correções.

## Output

Tabela com comando, exit code e status final.
