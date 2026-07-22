package ui

const helpManual = `DOCKERMIN — MANUAL OPERACIONAL DOCKER EM PT-BR

ATALHOS DO DOCKERMIN
  F1 ou ?       abre/fecha este manual
  Tab/Shift+Tab próxima/anterior aba
  ←/→           navega entre módulos
  j/k ou ↑/↓    move a seleção
  g/G           primeiro/último item
  r             atualiza; em Containers reinicia o selecionado
  R             liga/desliga atualização automática
  t             alterna tema
  q / Ctrl+C    encerra com segurança

CONTAINERS NO DOCKERMIN
  n  criar       abre formulário; baixa a imagem, cria e inicia
  S  start       inicia um container parado
  T  stop        solicita parada graciosa (timeout de 10s)
  r  restart     reinicia o container
  p  pause       congela/descongela todos os processos
  l  logs        exibe as últimas 300 linhas com timestamps
  i  inspect     mostra configuração e estado em JSON
  o  processos   equivale ao docker top
  e  exec        abre bash/sh/ash real; use exit para voltar
  d  remover     remove após confirmação digitada

FORMULÁRIO DE CONTAINER
  Ordem: nome | imagem | portas | volumes | ambiente | restart | comando
  Exemplo:
  web | nginx:alpine | 8080:80 | /srv/site:/usr/share/nginx/html:ro | APP_ENV=prod | unless-stopped |
  Portas: host:container, separadas por vírgula.
  Volumes: origem:destino[:ro], separados por vírgula.
  Ambiente: CHAVE=valor, separado por vírgula. Evite segredos na tela.
  Restart: no, always, unless-stopped ou on-failure.
  Comando: opcional; vazio usa CMD/ENTRYPOINT da imagem.

COMANDOS DOCKER — REFERÊNCIA RÁPIDA
  docker run       cria e inicia um container a partir de uma imagem
  docker create    cria sem iniciar
  docker start     inicia containers existentes
  docker stop      encerra graciosamente
  docker restart   reinicia
  docker kill      envia um sinal, SIGKILL por padrão
  docker rm        remove containers
  docker rename    altera o nome
  docker pause     suspende processos
  docker unpause   retoma processos
  docker ps        lista containers; -a inclui parados
  docker logs      mostra stdout/stderr; -f acompanha em tempo real
  docker exec      executa um processo dentro de container em execução
  docker attach    conecta ao processo principal do container
  docker top       lista processos internos
  docker stats     acompanha CPU, RAM, rede e I/O
  docker inspect   retorna configuração/estado detalhados em JSON
  docker port      mostra mapeamentos de portas
  docker diff      mostra alterações no filesystem do container
  docker cp        copia arquivos entre host e container
  docker export    exporta filesystem do container para TAR
  docker wait      aguarda o término e retorna o exit code
  docker update    altera limites e política de restart

  docker search    pesquisa imagens no Docker Hub
  docker pull      baixa imagem/tag
  docker push      envia imagem para registry
  docker images    lista imagens locais
  docker image ls  forma agrupada de listar imagens
  docker rmi       remove imagens
  docker tag       cria outra referência/tag
  docker history   mostra camadas e comandos da imagem
  docker save      salva uma ou mais imagens em TAR
  docker load      carrega imagens de TAR
  docker import    cria imagem a partir de um filesystem TAR
  docker build     constrói imagem usando Dockerfile/BuildKit
  docker builder   gerencia cache e instâncias do builder
  docker buildx    build multiplataforma e builders avançados
  docker manifest  inspeciona/cria manifests multi-arquitetura

  docker volume ls       lista volumes
  docker volume create   cria volume persistente
  docker volume inspect  mostra driver, labels e mountpoint
  docker volume rm       remove volume; pode causar perda de dados
  docker volume prune    remove volumes não utilizados

  docker network ls          lista redes
  docker network create      cria bridge, overlay ou outra rede
  docker network inspect     mostra IPAM e endpoints conectados
  docker network connect     conecta container a uma rede
  docker network disconnect  desconecta container
  docker network rm          remove rede
  docker network prune       remove redes não utilizadas

  docker login      autentica em registry; prefira credential helper
  docker logout     remove credencial do registry
  docker info       mostra daemon, storage, cgroups, plugins e Swarm
  docker version    mostra versões do cliente, servidor e API
  docker events     acompanha eventos em tempo real
  docker system df  mostra uso de disco do Docker
  docker system prune remove recursos não utilizados; ação destrutiva
  docker context    cria e alterna endpoints Docker

  docker swarm init    inicializa um Swarm neste manager
  docker swarm join    adiciona manager ou worker
  docker swarm leave   remove o nó do Swarm
  docker swarm update  altera configuração do cluster
  docker swarm unlock  desbloqueia manager com autolock

  docker service ls       lista serviços Swarm
  docker service create   cria serviço
  docker service scale    altera réplicas
  docker service update   atualiza imagem/configuração/placement
  docker service rollback volta à especificação anterior
  docker service ps       mostra tasks do serviço
  docker service logs     mostra logs agregados
  docker service rm       remove serviço

  docker node ls       lista nós do Swarm
  docker node inspect  detalha nó
  docker node update   altera role, labels ou availability
  docker node promote  promove worker a manager
  docker node demote   rebaixa manager para worker
  docker node rm       remove nó do cluster

  docker stack deploy    aplica Compose como stack Swarm
  docker stack services  lista serviços da stack
  docker stack ps        mostra tasks da stack
  docker stack rm        remove stack

  docker config create/ls/inspect/rm  gerencia configs Swarm
  docker secret create/ls/inspect/rm  gerencia secrets sem exibir conteúdo
  docker plugin install/ls/enable/disable/rm  gerencia plugins
  docker checkpoint create/ls/rm      checkpoints quando suportados
  docker trust       gerencia assinatura de imagens
  docker compose     aplicações Compose locais/multi-container

SEGURANÇA
  O grupo docker concede poder equivalente a root no host.
  Confirme contexto e endpoint no rodapé antes de agir.
  Use --read-only para observação. Nunca digite secrets em campos comuns.
  Volumes e prune podem apagar dados sem recuperação.`
