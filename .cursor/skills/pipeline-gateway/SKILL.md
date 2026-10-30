---
name: pipeline-gateway
description: "Gateway de recursos para descoberta dinâmica de APIs e schemas via MCP."
disable-model-invocation: true
---

# Pipeline Gateway

Roteia contexto para APIs externas com validação de schema antes de persistir dados.

## Sequência Obrigatória

1. **Validar contexto** — confirmar intenção e permissões do usuário
2. **Descobrir schema** — listar tools MCP disponíveis e ler JSON Schema
3. **Validar payload** — checar campos obrigatórios e tipos antes de enviar
4. **Executar** — chamar tool MCP com payload validado
5. **Registrar** — anotar resultado no scratchpad se tarefa longa

## Servidores MCP

Consulte `mcp-integration` para diretrizes. Servidores configuráveis em `.cursor/mcp.json.example`:

- `context7` — documentação de bibliotecas
- `mcp-github` — PRs, issues, CI
- `atlassian` — Jira e Confluence via Remote MCP (quando habilitado)

## Regras

- Nunca executar operações destrutivas (DELETE, DROP, force push) sem aprovação explícita
- Cachear resultados imutáveis no contexto da conversa
- Preferir leitura antes de escrita em sistemas externos
