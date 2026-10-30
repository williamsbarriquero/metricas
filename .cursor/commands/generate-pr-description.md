# Geração de Descrição de Pull Request (PR)

Execute as seguintes ações para gerar uma descrição detalhada e padronizada de PR com base no git diff ou nos arquivos selecionados:

1. Analise o diff e identifique todas as mudanças introduzidas no código.
2. Formate o output com as seguintes seções estruturadas:
   - **## 📝 Resumo**: Explicação suscinta do que foi alterado.
   - **## 💡 Motivação**: Qual problema de negócio ou bug esta alteração resolve.
   - **## 🧪 Como foi Testado**: Descreva os testes manuais executados ou suítes de testes automatizados adicionadas.
   - **## 🛡️ Impactos colaterais (Segurança/QA/SRE)**: Se há mudanças em rotas públicas, permissões de banco ou variáveis de ambiente.
   - **## ✅ Checklist**: Um checklist interativo contendo:
     - [ ] Código passa nos linters locales.
     - [ ] Testes unitários/E2E cobrem as alterações.
     - [ ] Segredos/credenciais foram verificados (sem vazar).
     - [ ] Documentação atualizada (se necessário).
