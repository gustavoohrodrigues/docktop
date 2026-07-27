# Changelog

## 0.3.0 - 2026-07-27

- Adiciona splash animada e localizada na inicialização.
- Adiciona viewport com rolagem para listas extensas de containers, imagens, registry, volumes, redes, serviços, nodes, stacks, events e auditoria.
- Adiciona navegação por `PgUp`/`PgDn`, `Ctrl+U`/`Ctrl+D`, `g`/`G` e roda do mouse, mantendo a seleção visível.
- Aceita `contexts` no YAML tanto como mapa quanto como lista nomeada, preservando compatibilidade com configurações antigas.

## 0.2.1 - 2026-07-21

- Corrige mapeamento `snake_case` da pesquisa real do Docker Hub.
- Criação guiada com portas, bind mounts, ambiente, restart policy e comando.
- Pull automático da imagem ausente antes de criar o container.
- Manual Docker em PT-BR acessível por F1/? e instruções contextuais mais claras.

## 0.2.0 - 2026-07-21

- Dashboard dark-ops redesenhado com painéis, meters e sparklines de CPU/RAM reais.
- Stats concorrentes e limitados para containers em execução.
- Pesquisa Docker Hub, pull com progresso consumido da Engine e tratamento de rate limit.
- Criação e inicialização de containers, volumes e redes.
- Logs, inspect, processos e terminal exec interativo com detecção de shell.
- Remoção reforçada de containers, imagens, volumes e redes.
- Sete temas internos e ajuda contextual.

## 0.1.0 - 2026-07-21

- Dashboard Docker real e inventário de containers, imagens, volumes e redes.
- Start, stop, restart, pause, unpause e remoção reforçada de containers.
- Contextos Unix/TCP/TLS, modo somente leitura, temas e auditoria JSONL.
