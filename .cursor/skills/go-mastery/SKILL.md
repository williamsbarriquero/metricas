---
name: go-mastery
description: Engenharia idiomática em Go. SSOT em knowledge/stacks/go.md.
disable-model-invocation: true
---

# Go Mastery

## Quando usar

- Implementação ou revisão de código `.go`
- Decisões de layout, concorrência e tratamento de erros
- Acionada por `go-standards.mdc`, `@rpe-developer.md` ou intent `infra`

## SSOT (leitura obrigatória)

Antes de implementar ou revisar Go, leia integralmente:

`.cursor/knowledge/stacks/go.md`

## Verificação

Conforme `stack-baseline.mdc` → seção Go. Na raiz do módulo: `gofmt -w .`, `go vet ./...`, `go test ./...`.
