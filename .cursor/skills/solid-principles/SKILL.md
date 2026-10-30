---
name: solid-principles
description: Fundamentos práticos dos princípios SOLID de design orientado a objetos.
disable-model-invocation: true
---

# Princípios SOLID

## Visão Geral

SOLID é um acrônimo para cinco princípios de design que tornam software mais compreensível, flexível e sustentável.

## 1. Single Responsibility Principle (SRP)

*Uma classe deve ter um, e apenas um, motivo para mudar.*

- **Definição**: Cada módulo ou classe deve ter responsabilidade sobre uma única parte da funcionalidade fornecida pelo software.
- **Heurística**: Se você consegue pensar em mais de um motivo para alterar uma classe, ela tem mais de uma responsabilidade.
- **Exemplo de violação**: Uma classe que formata um relatório e também o imprime em um stream de saída.

## 2. Open/Closed Principle (OCP)

*Entidades de software devem ser abertas para extensão, mas fechadas para modificação.*

- **Definição**: Você deve poder estender o comportamento de uma classe sem modificá-la.
- **Heurística**: Use polimorfismo. Em vez de escrever `if/switch` baseados em tipos de objeto, crie uma abstração e implemente-a de formas diferentes.
- **Exemplo de violação**: Uma classe que calcula custos de frete e precisa ser modificada toda vez que um novo serviço de entrega é adicionado.

## 3. Liskov Substitution Principle (LSP)

*Objetos em um programa devem ser substituíveis por instâncias de seus subtipos sem alterar a correção do programa.*

- **Definição**: Subclasses devem ser substituíveis por suas classes base.
- **Heurística**: Se um método sobrescrito não faz nada ou lança exceção apenas por ser de uma subclasse, você provavelmente está violando o LSP.
- **Exemplo de violação**: Uma classe `Square` herdando de `Rectangle` onde definir a largura também altera a altura, quebrando expectativas de código que trabalha com `Rectangle`.

## 4. Interface Segregation Principle (ISP)

*Muitas interfaces específicas por cliente são melhores que uma interface generalista.*

- **Definição**: Clientes não devem ser forçados a depender de interfaces que não utilizam.
- **Heurística**: Interfaces gordas devem ser divididas em interfaces menores e altamente coesas.
- **Exemplo de violação**: Uma interface monolítica `Worker` com métodos `work()`, `eat()` e `sleep()` implementada por uma classe `Robot` (robôs não comem nem dormem).

## 5. Dependency Inversion Principle (DIP)

*Dependa de abstrações, não de implementações concretas.*

- **Definição**: Módulos de alto nível não devem depender de módulos de baixo nível. Ambos devem depender de abstrações. Abstrações não devem depender de detalhes. Detalhes devem depender de abstrações.
- **Heurística**: Injete dependências via construtores em vez de instanciar classes concretas dentro de módulos de alto nível.
- **Exemplo de violação**: Uma classe `PaymentService` instanciando diretamente uma classe `StripeClient`.
