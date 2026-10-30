---
name: mcp-integration
description: Guia de integração e uso de ferramentas Model Context Protocol (MCP) no RPE Harness.
disable-model-invocation: true
---

# Integração e Ferramentas MCP

O Model Context Protocol (MCP) estende as capacidades dos agentes de IA com acesso a documentação, GitHub e Jira.

## Diretrizes de Uso

1. **Listar tools**: Consultar schema/descriptor antes de invocar qualquer tool MCP.
2. **Respeitar schemas**: Validar payload contra JSON Schema da tool.
3. **Não-destrutividade**: Nunca executar DELETE, DROP, force push ou operações irreversíveis sem aprovação explícita.
4. **Cache**: Evitar chamadas repetidas no mesmo turno para dados imutáveis.

## Servidores Configuráveis

Template em `.cursor/mcp.json.example`. Variáveis via ambiente (`${env:VAR}`).

### context7

- **Objetivo**: Documentação atualizada de bibliotecas e frameworks.
- **Quando usar**: Dúvidas sobre API, configuração ou migração de dependências.

### mcp-github

- **Objetivo**: PRs, issues, checks de CI.
- **Quando usar**: Criar PR, ler reviews, diagnosticar falhas de pipeline.

### atlassian (opcional)

- **Objetivo**: Jira e Confluence via Atlassian Remote MCP (OAuth).
- **Quando usar**: `@rpe-atlassian.md` para ler/atualizar histórias, páginas e RFCs.
- **Configuração**: `mcp-remote` → `https://mcp.atlassian.com/v1/mcp` (ver `.cursor/mcp.json.example`).

### mcp-postgres (desabilitado por default)

- **Risco**: Acesso SQL direto. Habilitar somente com read-only e whitelist de schema.
- **Guard**: Hook `mcp-guard.sh` bloqueia DROP/TRUNCATE/DELETE em massa.

## Agentes por Servidor

| Servidor | Agente principal |
|----------|------------------|
| context7 | Todos (documentação) |
| mcp-github | developer, reviewer, infra |
| atlassian | atlassian |
| mcp-postgres | dba (somente leitura) |
