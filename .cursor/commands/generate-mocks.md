# Geração de Mocks de API

Execute as seguintes ações para gerar dublês de teste a partir do schema OpenAPI, Swagger ou payload JSON enviado:

1. Identifique a estrutura de campos, tipos e payloads de requisição/resposta.
2. Pergunte ao usuário qual o framework de teste desejado (ex: MSW no TypeScript, WireMock no Java, Go mocks nativos). Se não especificado, use o padrão mais comum para a linguagem do projeto.
3. Produza o código de mock completo e configurável, incluindo respostas de sucesso (200/201) e simulações de erro comuns (400, 401, 500).
