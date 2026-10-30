---
name: performance-testing
description: Testes de carga, estresse e definição de limites de performance com K6.
disable-model-invocation: true
---

# Testes de Performance e Carga

## Conceitos K6

- **VUs**: Usuários virtuais executando cenários simultaneamente.
- **Thresholds**: Limites automáticos de aceitação (ex: p95 < 500ms).
- **Stages**: Ramp-up, plateau e ramp-down de carga.

## Tipos de Teste

| Tipo | Objetivo |
|------|----------|
| Load | Comportamento sob carga normal |
| Stress | Encontrar limite do sistema |
| Spike | Resiliência a picos repentinos |
| Soak | Vazamento de memória em carga prolongada |

## Exemplo Completo

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 20 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const res = http.get('https://api.example.com/health');
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(1);
}
```

## Comando

`k6 run tests/performance/load-test.js`

Comando de scaffold: `@scaffold-k6.md`
