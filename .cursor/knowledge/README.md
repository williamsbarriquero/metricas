# Knowledge Hub — SSOT de Conteúdo por Stack

Este diretório centraliza o **conteúdo técnico** que antes estava duplicado entre rules, skills e commands.

## Camadas (responsabilidade única)

| Camada | Pasta | Papel | Tamanho típico |
|--------|-------|-------|----------------|
| **Knowledge** | `.cursor/knowledge/stacks/` | SSOT — padrões, convenções, ecossistema | Completo |
| **Skill** | `.cursor/skills/*-mastery/` | Índice de ativação + quando usar | Curto (~30 linhas) |
| **Rule contextual** | `.cursor/rules/*-standards.mdc` | Globs + ponte para skill/knowledge + verificação | Curto (~25 linhas) |
| **Rule global** | `.cursor/rules/*.mdc` | Harness (identidade, segurança, DoD) — **nunca** conteúdo de stack | Médio |
| **Command** | `.cursor/commands/*.md` | Workflow acionável — referencia rules/skills, não duplica conteúdo | Curto |

## Fluxo de ativação

```text
Arquivo aberto (*.go)
    → go-standards.mdc (glob)
        → skill go-mastery (índice)
            → knowledge/stacks/go.md (SSOT)

Arquivo aberto (*.tsx)
    → frontend-standards.mdc (glob)
        → skill frontend-mastery (índice)
            → knowledge/stacks/frontend.md (SSOT)
```

## Regras de manutenção

1. **Edite o knowledge** quando mudar padrões de uma stack.
2. **Não duplique** conteúdo técnico em rules `alwaysApply: true` — isso polui todas as sessões.
3. Skills e rules contextuais **apontam** para o knowledge; não copiam parágrafos.
4. Commands referenciam `stack-baseline.mdc` para verificação — não repetem tabelas de comandos.

## Stacks disponíveis

Instaladas neste repositório (`go` e `react`):

- [stacks/go.md](stacks/go.md)
- [stacks/frontend.md](stacks/frontend.md)
