# Scaffold E2E

Crie estrutura de testes E2E para o fluxo solicitado.

## Passos

1. Identificar framework do projeto (Playwright ou Cypress).
2. Consultar skill `e2e-testing` para anti-flakiness e seletores resilientes.
3. Criar estrutura Page Object Model (POM):
   - Pages para cada tela/endpoint UI
   - Fixtures para setup/teardown de dados
   - Testes independentes e idempotentes
4. Usar `data-testid` ou roles ARIA em vez de seletores frágeis.
5. Integrar com CI se `package.json` ou workflow existir.

## Output

Arquivos de teste E2E, pages e fixtures criados com instruções de execução.
