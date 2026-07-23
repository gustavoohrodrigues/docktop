<div align="center">

<img src="docs/assets/docktop-whale.png" alt="Baleia pixel art do DockTop carregando containers" width="680">

# DockTop

### Docker e Docker Swarm, direto do terminal.

Uma TUI em Go para acompanhar e operar ambientes Docker sem precisar alternar entre dezenas de comandos.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-amd64%20%7C%20arm64-FCC624?style=flat-square&logo=linux&logoColor=black)](#compatibilidade)
[![License](https://img.shields.io/badge/license-MIT-56F39A?style=flat-square)](LICENSE)

</div>

---

## Sobre o projeto

O DockTop nasceu para facilitar o trabalho de quem administra Docker em servidores Linux e prefere continuar no terminal. Ele conversa com o Docker Engine pela API, organiza os recursos em telas navegáveis e adiciona proteções para operações que podem causar indisponibilidade ou perda de dados.

Não há servidor web, banco de dados ou agente instalado no host. O DockTop é um binário único que se conecta ao socket local ou a um daemon remoto.

> [!IMPORTANT]
> O projeto está em desenvolvimento ativo. Consulte [Limitações atuais](#limitações-atuais) antes de utilizá-lo em produção.

## O que já funciona

- Dashboard com informações do Engine, uso de CPU, memória e resumo dos recursos.
- Containers com métricas, criação, start, stop, restart, pause e remoção.
- Atualização da imagem de um container com recriação e rollback.
- Logs, inspect, processos e shell interativo.
- Imagens locais e pull com progresso pela API do Docker.
- Pesquisa de imagens no Docker Hub.
- Criação e remoção de volumes e redes.
- Detecção automática de Docker standalone, Swarm worker e Swarm manager.
- Serviços e tasks Swarm, incluindo inspect e scale de serviços replicated.
- Nodes Swarm com inspect e alteração de availability.
- Stacks agrupadas por `com.docker.stack.namespace`.
- Eventos recentes do Docker Engine.
- Auditoria local em JSONL, com sanitização, rotação e permissões restritas.
- Modo somente leitura para ambientes sensíveis.
- Temas persistentes e interface adaptada a terminais menores.
- Interface em Português do Brasil, Inglês e Espanhol.
- Navegação por teclado e suporte opcional a mouse.

## Instalação

### Compilação manual

É necessário ter Go 1.25 ou uma versão compatível com o `go.mod`.

```bash
git clone https://github.com/gustavoohrodrigues/docktop.git
cd docktop
go mod tidy
go test ./...
go build -o docktop ./cmd/docktop
./docktop
```

Para disponibilizar o binário para todos os usuários:

```bash
sudo install -Dm755 docktop /usr/local/bin/docktop
docktop --version
```

### Dependências de compilação

<details>
<summary>Ubuntu / Debian</summary>

```bash
sudo apt update
sudo apt install golang-go make git
```

</details>

<details>
<summary>Fedora</summary>

```bash
sudo dnf install golang make git
```

</details>

<details>
<summary>RHEL / Rocky Linux / AlmaLinux</summary>

```bash
sudo dnf install golang make git
```

</details>

> Binários para Linux amd64 e arm64 serão disponibilizados em GitHub Releases quando a primeira versão pública for publicada.

## Primeiro uso

O contexto padrão usa o socket local:

```text
unix:///var/run/docker.sock
```

O usuário precisa ter permissão para acessar esse socket. Normalmente isso é feito por associação ao grupo `docker`:

```bash
sudo usermod -aG docker "$USER"
```

Depois da alteração, encerre a sessão e entre novamente. Confirme o acesso antes de iniciar o DockTop:

```bash
docker info
docktop
```

> [!WARNING]
> O grupo `docker` concede, na prática, privilégios equivalentes a root no host. Não adicione usuários ao grupo sem compreender esse impacto.

## Uso

```bash
# Contexto padrão
docktop

# Impedir qualquer mutação
docktop --read-only

# Escolher contexto, tema e idioma
docktop --context production-manager
docktop --theme nord
docktop --language en-US

# Usar configuração alternativa
docktop --config /etc/docktop/config.yaml

# Desabilitar mouse
docktop --no-mouse
```

Todas as opções:

```bash
docktop --help
```

## Atalhos principais

| Tecla | Ação |
|---|---|
| `Tab` / `Shift+Tab` | Próxima ou anterior área |
| `←` / `→` | Navegar entre módulos |
| `↑` / `↓` ou `j` / `k` | Navegar em listas |
| `Enter` | Abrir detalhes ou confirmar |
| `Esc` | Fechar modal ou retornar |
| `/` | Pesquisa contextual |
| `r` | Atualizar a tela atual |
| `R` | Ligar ou desligar auto-refresh |
| `x` | Reiniciar container selecionado |
| `u` | Atualizar imagem do container |
| `t` | Trocar tema |
| `L` | Escolher idioma em Settings |
| `?` / `F1` | Abrir ajuda |
| `q` | Sair |

Os atalhos disponíveis para cada recurso também aparecem no rodapé da aplicação.

## Configuração

O arquivo padrão fica em:

```text
~/.config/docktop/config.yaml
```

Use [config.example.yaml](config.example.yaml) como ponto de partida.

### Socket local

```yaml
default_context: local

contexts:
  local:
    host: unix:///var/run/docker.sock

theme: dark-ops
language: pt-BR
refresh_interval: 3s
read_only: false
mouse_enabled: true
telemetry_enabled: false
```

### Docker remoto com TLS

```yaml
default_context: production-manager

contexts:
  production-manager:
    host: tcp://docker-manager.example.net:2376
    description: Manager Swarm de produção
    tls:
      enabled: true
      ca_file: /home/admin/.docker/ca.pem
      cert_file: /home/admin/.docker/cert.pem
      key_file: /home/admin/.docker/key.pem
```

O DockTop também respeita `DOCKER_HOST`, `DOCKER_CONTEXT` e `DOCKER_TLS_VERIFY` quando aplicável.

## Docker Swarm

O comportamento depende do endpoint conectado:

| Endpoint | Comportamento |
|---|---|
| Standalone | Recursos locais do Docker Engine |
| Swarm worker | Recursos locais e explicação das limitações do cluster |
| Swarm manager | Serviços, tasks, nodes, stacks e operações do plano de controle |

Alterações em nodes e outras ações sensíveis respeitam `--read-only`, a política de ações perigosas e confirmações digitadas.

## Segurança e auditoria

- `--read-only` impede operações de escrita.
- Remoções e mudanças críticas exigem confirmação.
- Variáveis com nomes como `password`, `token`, `secret`, `key` e `credential` são mascaradas.
- Credenciais TLS não são exibidas ou gravadas na auditoria.
- O histórico padrão fica em `~/.local/share/docktop/audit.jsonl`.
- Arquivos de auditoria usam permissão `0600` e diretórios usam `0700`.
- Rotação e retenção são configuráveis.

## Arquitetura

```text
cmd/docktop          CLI e ciclo de vida
internal/app         composição da aplicação
internal/config      configuração e contextos
internal/docker      integração com Docker Engine
internal/i18n        traduções e localização de erros
internal/jobs        operações em background
internal/audit       auditoria local
internal/registry    integração com registries
internal/theme       temas e cores semânticas
internal/ui          estado e renderização Bubble Tea
internal/utils       utilidades e sanitização
data/themes          temas distribuídos com o projeto
docs/assets          recursos usados pela documentação
```

O fluxo principal é:

```text
Bubble Tea UI → interface Engine → Docker SDK → Docker Engine
```

A interface do Engine mantém a camada visual desacoplada do SDK e permite testes sem colocar mocks no fluxo de produção.

## Desenvolvimento

```bash
# Formatação
gofmt -w $(find cmd internal -name '*.go')

# Dependências
go mod tidy

# Testes
go test ./...

# Testes com detector de race
go test -race ./...

# Build
go build -o docktop ./cmd/docktop
```

Também estão disponíveis:

```bash
make test
make build
make run
```

## Compatibilidade

- Linux amd64.
- Linux arm64.
- Docker Engine por socket Unix.
- Docker Engine remoto por TCP/TLS.
- Docker standalone.
- Docker Swarm worker e manager.

## Limitações atuais

- Logs são carregados sob demanda, ainda sem follow contínuo.
- O shell interativo ainda utiliza o Docker CLI para controlar o raw TTY; a detecção do shell usa a API.
- Update avançado, rollback e remoção de serviços Swarm ainda não estão expostos.
- Promote e demote de managers ainda não estão disponíveis na TUI.
- Stacks são agrupadas e inspecionadas, mas deploy e remoção ainda não estão implementados.
- Events utiliza uma janela recente em vez de um stream persistente.
- Alguns textos livres retornados pelo daemon, logs e labels permanecem no idioma original da fonte.

## Solução de problemas

<details>
<summary>Permissão negada no socket Docker</summary>

Confira o usuário, os grupos e as permissões:

```bash
id
ls -l /var/run/docker.sock
docker info
```

</details>

<details>
<summary>Docker daemon indisponível</summary>

```bash
sudo systemctl status docker
sudo systemctl start docker
```

</details>

<details>
<summary>Falha no contexto remoto TLS</summary>

Verifique nomes DNS, validade da CA, certificado do cliente, chave privada e permissões dos arquivos. Confirme também se o daemon escuta no endpoint configurado.

</details>

## Contribuindo

Issues, discussões e pull requests são bem-vindos. Antes de enviar uma mudança:

1. Descreva claramente o problema ou comportamento desejado.
2. Mantenha operações Docker fora da camada de UI.
3. Adicione testes para regras de negócio e parsing.
4. Execute `go test ./...` e `go build ./cmd/docktop`.
5. Não adicione mocks ou ações decorativas ao fluxo de produção.

## Licença

Distribuído sob a licença [MIT](LICENSE).

---

<div align="center">

By [Orqly](https://github.com/orqly).

</div>
