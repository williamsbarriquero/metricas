# Especificações de features

Diretório **SSOT** para specs versionadas no Git. Criado pelo `install.sh`.

## Convenção de arquivos

Por feature, use **dois arquivos**:

| Arquivo | Conteúdo | Quando |
|---------|----------|--------|
| `<slug>.md` | Comportamentos, componentes, critérios de aceite (níveis 1–3) | Após design no Plan Mode — **requer aprovação** |
| `<slug>-technical.md` | Interfaces, DTOs, schemas, contratos (nível 4) | Após aprovação da feature spec — alimenta testes RED |

**Exemplo:** `forgot-password.md` + `forgot-password-technical.md`

## Templates

Copiados para `.cursor/templates/` na instalação:

- `feature-spec.md`
- `technical-spec.md`

## Fluxo resumido

1. Design — `@rpe-architect.md` + `@plan-architecture.md` (Plan Mode)
2. Gerar specs — Act Mode, templates acima → gravar nesta pasta
3. Testes — `@rpe-tester.md` + `@.cursor/docs/specs/<slug>-technical.md`
4. Implementação — `@rpe-developer.md` conforme spec técnica
5. Revisão — `@rpe-reviewer.md` + `@.cursor/docs/specs/<slug>.md`

**Regra:** se o código divergir da intenção, **atualize a spec primeiro** e regenere — não corrija código manualmente contornando o fluxo.

Publicação opcional no Confluence: `@rpe-atlassian.md` (espelho para o time).
