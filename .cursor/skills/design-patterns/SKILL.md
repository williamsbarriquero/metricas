---
name: design-patterns
description: Catálogo de padrões de projeto GoF e arquiteturas empresariais.
disable-model-invocation: true
---

# Padrões de Projeto (Design Patterns)

## Filosofia Central

Padrões de projeto são soluções típicas para problemas comuns em design de software. Funcionam como plantas prontas que você pode adaptar para resolver um problema recorrente de design no seu código.

**Aviso importante**: não aplique padrões desnecessariamente. Overengineering é uma armadilha comum. Use um padrão somente se o problema que ele resolve existir no seu domínio e se o padrão simplificar o código ou torná-lo significativamente mais sustentável.

## Padrões Criacionais (Creational)

Tratam dos mecanismos de criação de objetos, buscando instanciá-los de forma adequada ao contexto.

- **Singleton**: Garante que uma classe tenha apenas uma instância e fornece um ponto global de acesso a ela. (Use com moderação).
- **Factory Method**: Define uma interface para criar um objeto, mas deixa as subclasses decidirem qual classe instanciar.
- **Abstract Factory**: Fornece uma interface para criar famílias de objetos relacionados ou dependentes sem especificar suas classes concretas.
- **Builder**: Separa a construção de um objeto complexo de sua representação, permitindo que o mesmo processo de construção crie representações diferentes.
- **Prototype**: Especifica os tipos de objetos a serem criados a partir de uma instância prototípica e cria novos objetos copiando esse protótipo.

## Padrões Estruturais (Structural)

Tratam da composição de objetos ou da forma de estabelecer relações entre entidades.

- **Adapter**: Converte a interface de uma classe em outra interface esperada pelos clientes.
- **Bridge**: Desacopla uma abstração de sua implementação para que ambas possam variar de forma independente.
- **Composite**: Compõe objetos em estruturas em árvore para representar hierarquias parte-todo.
- **Decorator**: Anexa responsabilidades adicionais a um objeto de forma dinâmica.
- **Facade**: Fornece uma interface unificada para um conjunto de interfaces de um subsistema.
- **Proxy**: Fornece um substituto ou placeholder para outro objeto, controlando o acesso a ele.

## Padrões Comportamentais (Behavioral)

Tratam da comunicação entre objetos.

- **Observer**: Define uma dependência um-para-muitos entre objetos, de modo que, quando um objeto muda de estado, todos os seus dependentes são notificados e atualizados automaticamente.
- **Strategy**: Define uma família de algoritmos, encapsula cada um deles e os torna intercambiáveis.
- **Command**: Encapsula uma requisição como um objeto, permitindo parametrizar clientes com diferentes requisições, enfileirar ou registrar operações e suportar ações desfazíveis (undo).
- **State**: Permite que um objeto altere seu comportamento quando seu estado interno muda.
- **Chain of Responsibility**: Evita acoplar o emissor de uma requisição ao seu receptor, dando a mais de um objeto a chance de tratá-la.
