Implementação do backend da rinha de maneira assíncrona

## Sobre a Solução

Este projeto foi feito para a Rinha de Backend 2025, onde o objetivo é implementar um sistema de pagamentos assíncrono utilizando Go e Docker. A solução foi projetada para ser escalável, eficiente e fácil de manter.

### Objetivo
O objetivo é criar um sistema de pagamentos que possa processar transações de forma assíncrona, permitindo que o sistema continue respondendo a outras requisições enquanto os pagamentos são processados em segundo plano. Isso melhora a performance e a experiência do usuário, evitando bloqueios e lentidões no sistema.

### Arquitetura
A arquitetura do sistema é baseada em microserviços, onde cada serviço é responsável por uma parte específica do processamento de pagamentos. A comunicação entre os serviços é feita através de filas, permitindo que os serviços sejam desacoplados e escaláveis.

### Tecnologias utilizadas
- **Go**: A linguagem principal do projeto, escolhida por sua performance e simplicidade.
  - foi utilizado o pkg gorpc para a comunicação entre os serviços, permitindo uma integração eficiente e leve.

### Como funciona
- Existem três aplicações principais:
  - **Api**: Recebe as requisições de pagamento e as encaminha para a fila de processamento.
  - **Consumer**: Consome as mensagens da fila e processa os pagamentos.
  - **Cache**: Armazena os dados dos pagamentos pendentes e já processados.

### Como rodar o projeto
- É necessario ter o docker instalado.
- Para rodar o projeto, execute o seguinte comando na raiz do projeto:
```bash
docker compose up --build
```
- Isso irá construir e iniciar os containers necessários para o funcionamento do backend.

Para testar o projeto, você pode usar o guia [mini-guia de setup](https://github.com/zanfranceschi/rinha-de-backend-2025/blob/main/rinha-test/MINIGUIA.md)
