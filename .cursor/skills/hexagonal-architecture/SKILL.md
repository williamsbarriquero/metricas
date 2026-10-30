---
name: hexagonal-architecture
description: Biblioteca de conhecimento sobre arquitetura hexagonal (Ports & Adapters).
disable-model-invocation: true
---

# Arquitetura Hexagonal (Ports & Adapters)

## Princípios Centrais

A Arquitetura Hexagonal, ou arquitetura Ports and Adapters, busca criar componentes de aplicação fracamente acoplados que possam ser conectados facilmente ao ambiente de software. Torna os componentes substituíveis em qualquer nível e facilita a automação de testes.

1. **Regra de Dependência**: A lógica central do domínio não depende de nada externo a ela. Todas as dependências apontam para dentro, em direção ao domínio.
2. **Separação de Responsabilidades**: Mantenha a lógica de negócio completamente separada de sistemas externos (bancos de dados, UI, APIs, message brokers).
3. **Ports**: Interfaces definidas pelo núcleo da aplicação que especificam como ela se comunica com o mundo externo.
   - **Ports de entrada/driving/primary**: Como atores externos interagem com a aplicação (ex.: interfaces de Use Case).
   - **Ports de saída/driven/secondary**: Como a aplicação interage com atores externos (ex.: interfaces de Repository).
4. **Adapters**: Implementações que interagem com o mundo externo e se conectam aos Ports.
   - **Adapters primários**: Implementam ports de entrada (ex.: REST Controllers, CLI Commands, GraphQL Resolvers). Eles conduzem a aplicação.
   - **Adapters secundários**: Implementam ports de saída (ex.: Database Repositories, HTTP Clients para serviços externos, Event Publishers). Eles são conduzidos pela aplicação.

## Boas Práticas

- **Isolamento do domínio**: Garanta que entidades de domínio e use cases não tenham conhecimento de frameworks, bancos de dados ou camadas de transporte.
- **Injeção de dependência**: Use DI para conectar adapters aos ports no ponto de entrada da aplicação (ex.: função main ou configuração do container DI).
- **Testabilidade**: O núcleo da aplicação deve ser testável sem dependências externas, mockando ou stubando os ports de saída.
- **Mapeamento**: Evite vazar entidades de domínio para o exterior. Mapeie DTOs externos para entidades de domínio nos adapters primários e entidades de domínio para modelos de banco ou payloads externos nos adapters secundários.
