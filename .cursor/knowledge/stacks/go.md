# Go — SSOT RPE

Conteúdo canônico para implementação e revisão Go. Referenciado por `go-mastery` e `go-standards.mdc`.

## Idiomático

- Formatar com `gofmt` ou `goimports`.
- `MixedCaps` / `mixedCaps` — sem underscores.
- Nomes curtos; interfaces terminam em `-er` (`Reader`, `Writer`).
- Simplicidade sobre one-liners.

## Estrutura de projeto

Layout Standard Go quando apropriado:

- `cmd/` — aplicações principais.
- `internal/` — código privado (imposto pelo compilador).
- `pkg/` — bibliotecas reutilizáveis externamente.
- `api/` — OpenAPI, JSON Schema, protos.

## Tratamento de erros

- Erros são valores — nunca ignorar com `_`.
- Evitar `panic`/`recover` exceto falhas irrecuperáveis na inicialização.
- Wrapping: `fmt.Errorf("...: %w", err)`; inspecionar com `errors.Is` / `errors.As`.

## Concorrência

- Goroutines para trabalho independente.
- Channels para comunicação segura entre goroutines.
- `context.Context` como primeiro argumento em I/O e operações longas.
- `sync.Mutex` / `sync.RWMutex` quando channels complicam demais.
- `sync.WaitGroup` para aguardar conjuntos de goroutines.
- Interfaces pequenas definidas pelo consumidor.

## Ponteiros vs valores

- Passar por valor por padrão.
- Ponteiro quando modificar receiver ou struct grande.
- Evitar ponteiros para interfaces.

## Verificação

Gate deste repositório, na raiz do módulo. `golangci-lint` não faz parte do projeto: não há `.golangci.yml`, e o README do módulo usa `go vet`.

- `gofmt -w .`
- `go vet ./...`
- `go test ./...`
