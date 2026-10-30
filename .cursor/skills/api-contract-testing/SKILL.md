---
name: api-contract-testing
description: Testes de API funcionais, validações de contrato e mockagem avançada.
disable-model-invocation: true
---

# Testes de API e de Contrato

## Testes de Contrato

- Garantir que Producer e Consumer concordem sobre formatos de payload.
- Validar JSON Schema em todas as respostas de API.
- Pact para contratos consumer-driven entre serviços.

## Mockagem

- **MSW** (frontend): interceptação de rede no browser/Node.
- **WireMock** (backend): stubs HTTP com cenários de timeout, 500 e payloads inválidos.

## Estrutura de Pastas Sugerida

```
tests/
  contract/
    consumer.pact.spec.ts
    provider.verify.ts
  mocks/
    handlers.ts        # MSW handlers
    wiremock/
      mappings/        # WireMock JSON mappings
```

## Exemplo MSW

```typescript
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.get('/api/users/:id', ({ params }) => {
    return HttpResponse.json({ id: params.id, name: 'Test User' });
  }),
];
```

## Integração CI

- Rodar `pact:verify` no pipeline antes de deploy.
- Falhar build se contrato quebrado.
