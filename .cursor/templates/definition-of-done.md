# Definition of Done — [slug-da-feature]

Checklist antes de abrir o PR. Complementa `agents.md` e `stack-baseline.mdc`.

## Especificação

- [ ] `.cursor/docs/specs/<slug>.md` existe e está **aprovada**
- [ ] `.cursor/docs/specs/<slug>-technical.md` reflete os contratos implementados
- [ ] Spec atualizada se houve drift durante a implementação

## Testes

- [ ] Testes RED confirmados antes da implementação
- [ ] Suíte relevante passa (exit 0)
- [ ] Edge cases da spec cobertos

## Qualidade

- [ ] `@validate-stack.md` passou (lint, format, type-check, testes)
- [ ] `@rpe-reviewer.md` + `@review-pr.md` executados
- [ ] Sem segredos hardcoded; input validado onde aplicável

## Entrega

- [ ] PR vinculado ao ticket Jira (se aplicável)
- [ ] Descrição gerada com `@generate-pr-description.md`
