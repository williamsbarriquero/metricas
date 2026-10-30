# Scaffold K6

Crie script de teste de performance/carga com K6 para o endpoint ou fluxo solicitado.

## Passos

1. Consultar skill `performance-testing`.
2. Definir tipo de teste: load, stress, spike ou soak.
3. Configurar VUs, duration e thresholds (ex: `http_req_duration: ['p(95)<500']`).
4. Criar script em `tests/performance/` ou `k6/` conforme convenção do projeto.
5. Incluir cenários de autenticação e dados de teste se necessário.

## Exemplo de Threshold

```javascript
export const options = {
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};
```

## Output

Script k6 executável com instruções de execução (`k6 run script.js`).
