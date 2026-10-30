# Auditoria de Segurança

Execute varredura de segurança no código modificado ou no escopo indicado.

## Passos

1. Ler diff do branch ou arquivos indicados.
2. Aplicar `safety-guardrails.mdc` e checklist OWASP Top 10:
   - Credenciais hardcoded (regex: password/secret/api_key/token)
   - SQL injection (queries sem parametrização)
   - XSS (output não sanitizado)
   - Dependências vulneráveis (`npm audit` se Node)
3. Verificar se hook `secret-scanner.sh` passaria nos arquivos analisados.
4. Para deep dive AppSec (SLSA, IaC, K8s), escalar ao `@rpe-security.md`.

## Output

Relatório com vulnerabilidades por severidade (Critical/Major/Minor) e correções sugeridas.
