---
name: bdd-gherkin
description: Guia avançado de BDD e escrita de cenários em Gherkin.
disable-model-invocation: true
---
# BDD e Gherkin (Ports & Adapters de Comportamento)

## Princípios do BDD

- **3 Amigos**: Colaboração constante entre Negócio (PO/PM), Desenvolvimento e QA para alinhar a compreensão antes do código.
- **Linguagem Ubíqua**: Uso de uma linguagem comum que todos os membros do time entendem.

## Escrita Gherkin Avançada

- **Dado (Given)**: Configura o estado inicial do sistema. Evite ações nesta etapa.
- **Quando (When)**: Descreve a ação chave ou evento de negócio disparado pelo usuário.
- **Então (Then)**: Descreve o resultado observado ou efeito colateral. Deve ser assertivo.
- **Esquema do Cenário (Scenario Outline)**: Use para testar o mesmo cenário com múltiplos conjuntos de dados usando uma tabela de `Exemplos` (Examples).
- **Anti-patterns**:
  - Misturar comportamento com implementação (ex: "Dado que eu clico no botão #submit-btn e digito no input @email").
  - Cenários excessivamente longos com múltiplos "Quando" e "Então".
