# Revisão de Migração

Revise a migração de banco de dados solicitada contra as boas práticas RPE.

## Checklist

Aplicar `database-migrations.mdc`:

- [ ] Possui script de rollback explícito
- [ ] Usa expand-and-contract para alterações breaking
- [ ] Foreign keys possuem índice
- [ ] Evita locks longos (sem ALTER bloqueante em tabelas grandes sem estratégia)
- [ ] Nomes descritivos e versionados
- [ ] Testada em ambiente de staging

## Passos

1. Ler arquivos de migração no diff ou path indicado.
2. Avaliar impacto em dados existentes e downtime.
3. Gerar relatório com status Aprovado/Alterações/Rejeitado.
4. Sugerir correções se necessário.

## Output

Relatório seguindo `interaction-standards.mdc`.
