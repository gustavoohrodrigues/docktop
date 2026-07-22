# DockTop

**DockerMin — Docker & Swarm Operations Console**

DockTop é uma TUI Linux para operar Docker Engine. A versão 0.2 consulta o daemon de verdade, acompanha métricas, pesquisa e baixa imagens, cria recursos e oferece logs, inspect, processos e acesso interativo aos containers. Não há servidor, banco ou Docker-in-Docker.

## Arquitetura

- `cmd/docktop`: CLI, sinais e ciclo de vida.
- `internal/config`: YAML, contextos e políticas.
- `internal/docker`: adaptador isolado do SDK Docker.
- `internal/ui`: estado Bubble Tea e renderização Lip Gloss.
- `internal/audit`: trilha JSON Lines com modo `0600`.
- `internal/theme`: cores semânticas e troca em runtime.

O fluxo é `UI -> interface Engine -> SDK Docker`; isso permite testes sem acoplar componentes visuais ao daemon.

## Compilar

Requer Go 1.23 ou superior e acesso ao Docker Engine.

Fedora:

```sh
sudo dnf install golang make
go mod tidy
make test
make build
```

Ubuntu/Debian:

```sh
sudo apt update
sudo apt install golang-go make
go mod tidy
make test
make build
```

RHEL/Rocky/Alma:

```sh
sudo dnf install golang make
go mod tidy
make test
make build
```

Execute `./docktop`. O usuário precisa acessar `/var/run/docker.sock`, normalmente por associação ao grupo `docker`; esse grupo equivale operacionalmente a privilégio elevado no host. Não execute como root apenas para contornar uma configuração incorreta.

## Uso

```sh
./docktop --read-only
./docktop --context production-manager
./docktop --config /caminho/config.yaml --theme nord --no-mouse
```

Copie `config.example.yaml` para `~/.config/docktop/config.yaml`. Para TLS, use arquivos CA/cert/key protegidos; seus conteúdos nunca são registrados. A auditoria fica em `~/.local/share/docktop/audit.jsonl`.

Atalhos: `Tab`/setas horizontais navegam abas; `j/k` selecionam; `r` atualiza (ou reinicia na aba Containers); `R` alterna auto-refresh; `S` start; `T` stop; `p` pause/unpause; `l` logs; `i` inspect; `o` processos; `e` exec; `n` cria; `d` remove com confirmação digitada; `/` pesquisa no Registry; `?` abre ajuda; `t` troca tema; `q` sai.

### Criar um container

Na aba Containers, pressione `n`. O formulário usa sete campos separados por `|`:

```text
nome | imagem | portas | volumes | ambiente | restart | comando
```

Exemplo:

```text
web | nginx:alpine | 8080:80 | /srv/site:/usr/share/nginx/html:ro | APP_ENV=prod | unless-stopped |
```

Portas, volumes e variáveis múltiplas são separados por vírgula. Campos opcionais podem ficar vazios. Se a imagem não existir localmente, o DockTop executa o pull antes da criação. Use `F1` para abrir o manual completo dentro da TUI.

## Segurança

`--read-only` bloqueia mutações. Remoção de container respeita a política, mostra impacto e exige o nome exato. Erros passam por sanitização antes da auditoria. A UI não mostra secrets nem credenciais TLS.

## Recursos implementados

- Unix socket, `DOCKER_HOST` por contexto explícito e TCP/mTLS.
- Dashboard com Engine, host, Swarm, métricas reais, meters e sparklines históricos.
- Listagem real de containers, imagens, volumes e redes.
- Start/stop/restart/pause/unpause/remove real via SDK.
- Criação e inicialização de containers; criação de volumes e redes.
- Logs, inspect e processos; terminal `docker exec` interativo com shell detectado.
- Pesquisa no Docker Hub e pull de imagens pela Engine API.
- Auto-refresh cancelável, timeouts, mensagens de conexão acionáveis.
- Sete temas em runtime; auditoria JSONL.

## Limites conhecidos da versão 0.1

Logs são obtidos sob demanda (últimas 300 linhas), mas follow contínuo ainda não está disponível. O exec interativo requer o Docker CLI instalado porque ele fornece o raw-TTY; detecção e seleção do shell usam a Engine API. Formulários de criação da 0.2 são deliberadamente compactos e ainda não expõem portas, mounts e limites. Events, Audit UI, administração Swarm/Services/Nodes/Stacks, temas externos, persistência do tema e rotação de auditoria ficam para as próximas versões. `docker stack` exigirá fallback pelo CLI com argumentos estruturados, pois stacks não são recurso nativo universal da Engine API.

## Troubleshooting

- “usuário não possui acesso”: verifique `id`, grupo `docker` e permissões do socket; faça novo login após alterar grupos.
- “daemon não está ativo”: `systemctl status docker` e confirme o endpoint.
- TLS: confira nomes DNS, validade da CA e permissões dos arquivos.
- Swarm inativo: as informações básicas aparecem, mas esta versão não oferece mutações Swarm.
