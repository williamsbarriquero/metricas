---
name: e2e-testing
description: Práticas avançadas para testes de interface E2E com Playwright e Cypress.
disable-model-invocation: true
---
# Automação de Testes E2E (Modern Web)

## Estratégias de Estabilidade (Anti-Flakiness)

- **Auto-waiting**: Confie na espera inteligente do framework pelos elementos ficarem visíveis e interativos. Evite `page.waitForTimeout(5000)`.
- **Isolamento de Estado**: Sempre crie ou resete o estado da aplicação (via chamadas de API rápidas no setup) antes do teste de UI para não depender do resultado do teste anterior.
- **Page Object Model (POM)**: Agrupe seletores e ações de cada página em classes dedicadas para facilitar a manutenção.

## Seletores Resilientes

- Priorize seletores por acessibilidade (`getByRole`, `getByLabel`, `getByText`).
- Utilize atributos específicos de teste (`data-testid`, `data-qa`) quando os seletores semânticos não forem suficientes.
- Evite XPath absoluto (`/html/body/div[1]/form/input`).
