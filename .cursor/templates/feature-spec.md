# [slug-da-feature] — Especificação de feature

> Template: `.cursor/templates/feature-spec.md`
> Destino: `.cursor/docs/specs/<slug>.md` (versionado no Git)
> Requer **aprovação explícita** antes de gerar testes ou código.

| Campo | Valor |
|-------|-------|
| **Ticket** | FEATURE-XXX |
| **Status** | rascunho \| em revisão \| **aprovado** \| implementado |
| **Autor** | |
| **Última atualização** | YYYY-MM-DD |

---

## 1. Contexto

- Problema de negócio:
- Objetivo:
- Fora de escopo:

## 2. Capacidades (nível 1)

Comportamentos em linguagem natural — sem detalhes de implementação.

1.
2.

## 3. Componentes (nível 2)

| Módulo / camada | Ação (criar / alterar / remover) | Notas |
|-----------------|----------------------------------|-------|
| | | |

## 4. Interações (nível 3)

### Fluxo principal

```text
[ator] → [componente A] → [componente B] → [resultado]
```

### Eventos e exceções relevantes

-

## 5. Critérios de aceite (BDD)

```gherkin
Cenário: caminho feliz
  Dado ...
  Quando ...
  Então ...
```

## 6. Aprovação

- [ ] Revisado por produto / negócio
- [ ] Revisado por engenharia
- [ ] **Aprovado para fase técnica** (gerar `<slug>-technical.md` e testes)

---

## Histórico

| Data | Autor | Alteração |
|------|-------|-----------|
| | | Versão inicial |
