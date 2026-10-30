---
name: lsp-integration
description: Guia de uso dos recursos nativos do Language Server Protocol (LSP) para navegação e depuração estática na IDE.
disable-model-invocation: true
---

# Engenharia de Contexto via LSP (Language Server Protocol)

O LSP permite que o agente consulte o compilador da IDE diretamente para navegar de forma extremamente precisa em bases de código complexas e ler avisos/erros de compilação sem precisar rodar suítes de testes pesadas todas as vezes.

## Diretrizes de Navegação e Análise

Ao analisar o codebase, o agente deve priorizar as seguintes interações nativas do LSP (quando disponíveis na IDE ou via ferramentas acopladas):

1. **Go to Definition (Ir para Definição)**:
   * **Objetivo**: Descobrir a implementação exata de uma função, classe, struct ou tipo.
   * **Quando usar**: Sempre que encontrar um símbolo desconhecido no código. Evite fazer buscas de texto simples se o atalho do LSP conseguir te levar diretamente à declaração original.

2. **Find References (Achar Referências)**:
   * **Objetivo**: Listar todas as linhas em que uma variável, classe ou método é utilizado no workspace.
   * **Quando usar**: Antes de fazer qualquer refatoração de assinatura de método ou alteração de tipo estrutural para prever o impacto colateral (breaking changes) em outros pacotes.

3. **LSP Diagnostics (Erros de Compilação & Linter)**:
   * **Objetivo**: Ler e analisar avisos de tipos, problemas de importação, variáveis não utilizadas ou erros de sintaxe gerados em tempo real pela IDE.
   * **Quando usar**: Trate estes diagnósticos como a primeira linha de defesa. Antes de rodar testes demorados, corrija todos os avisos do compilador e de tipagem acusados no arquivo atual.
