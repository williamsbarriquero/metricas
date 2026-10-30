---
name: accessibility-visual-testing
description: Acessibilidade digital (WCAG) e testes de regressão visual.
disable-model-invocation: true
---
# Acessibilidade (a11y) e Regressão Visual

## Acessibilidade Digital (WCAG 2.1)

- **Leitores de Tela**: Garantir o uso correto de tags semânticas e atributos ARIA (`aria-expanded`, `aria-label`).
- **Navegação via Teclado**: Todo componente interativo deve ser focável e navegável via `Tab` e executável via `Enter`/`Space`.
- **Contraste**: Relação de contraste de texto mínima de 4.5:1 (ou 3:1 para textos grandes).
- **Validação Automatizada**: Integração do `axe-core` nos testes Cypress/Playwright para capturar falhas de a11y comuns em tempo de compilação.

## Regressão Visual

- Comparação pixel a pixel de capturas de tela (screenshots) contra snapshots de referência.
- Configure margens de tolerância de cor e desalinhamento pequenas (`mismatchThreshold`) para evitar falsos positivos de fontes/renderização do SO.
