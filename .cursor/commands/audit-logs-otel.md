# Auditoria de Logs e Rastreabilidade (OpenTelemetry & LGPD)

Execute as seguintes ações para auditar logs e spans nos arquivos de código selecionados:

1. Varra o código procurando por chamadas de escrita de logs (ex: `log.info`, `console.log`, `logger.error`).
2. Avalie as seguintes conformidades:
   - **Rastreabilidade**: Há propagação de trace context ou IDs de transação no escopo do log?
   - **LGPD/Sanitização**: Há risco de vazar dados pessoais (PII) nos logs? (ex: senhas, CPFs, e-mails, tokens). Se sim, indique o mascaramento ou remoção.
   - **Nível de Severidade**: Os níveis de log (debug, info, warn, error) são usados de forma adequada ou há excesso de ruído?
3. Gere um relatório estruturado apontando vulnerabilidades ou desvios encontrados e forneça correções pragmáticas.
