<div align="center">

<img src="docs/assets/docktop-whale.png" alt="Baleia pixel art do DockTop carregando containers" width="680">

# DockTop

### Docker e Docker Swarm, direto do terminal.

Uma TUI escrita em Go para observar, diagnosticar e operar ambientes Docker
em servidores Linux — do Engine local ao cluster Swarm.

[![Release](https://img.shields.io/github/v/release/gustavoohrodrigues/docktop?style=flat-square&color=32e6d0)](https://github.com/gustavoohrodrigues/docktop/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-amd64%20%7C%20arm64-FCC624?style=flat-square&logo=linux&logoColor=black)](#compatibilidade)
[![Docker](https://img.shields.io/badge/Docker-Engine%20%7C%20Swarm-2496ED?style=flat-square&logo=docker&logoColor=white)](#docker-swarm)
[![License](https://img.shields.io/badge/licen%C3%A7a-MIT-56F39A?style=flat-square)](LICENSE)

[**Site oficial ↗**](https://docktop.dev/) •
[**Documentação ↗**](https://docktop.dev/#docs) •
[**Instalar ↗**](https://docktop.dev/#install) •
[**Releases ↗**](https://github.com/gustavoohrodrigues/docktop/releases)

[Visão geral](#visão-geral) •
[Funcionalidades](#funcionalidades) •
[Instalação](#instalação) •
[Uso](#uso) •
[Segurança](#segurança-e-auditoria) •
[Limitações](#limitações-atuais)

</div>

---

## Visão geral

O **DockTop** conecta-se diretamente à API do Docker Engine e organiza
containers, imagens, volumes, redes, eventos e recursos Swarm em uma interface
orientada ao teclado.

Não há servidor web, banco de dados ou agente adicional. A distribuição oficial
consiste em um único binário para Linux `amd64` ou `arm64`, com suporte ao socket
Docker local e a endpoints remotos com TLS.

### Destaques da v0.3.7

- Security Audit somente leitura para containers;
- Apply Hardening Custom com seleção explícita de controles;
- comparação antes/depois e confirmação pelo nome do container;
- backup do container original e rollback automático em caso de falha;
- avaliação de privilégios, capabilities, mounts, namespaces e limites;
- suporte a seccomp, AppArmor, `no-new-privileges`, `tmpfs` e usuário não-root;
- proteção contra recriação direta de containers Compose e tasks Swarm;
- interface e mensagens disponíveis em Português, Inglês e Espanhol.

> [!IMPORTANT]
> A auditoria analisa a configuração de runtime observável. Ela não substitui
> uma análise de vulnerabilidades da imagem e não garante compatibilidade das
> remediações. Em ambientes críticos, comece com `docktop --read-only`.

## Funcionalidades

| Área | Recursos disponíveis |
|---|---|
| **Dashboard** | Informações do Engine, CPU, memória e resumo dos recursos |
| **Containers** | Métricas, criação, start, stop, restart, pause e remoção |
| **Atualização** | Atualização de imagem, recriação do container e rollback |
| **Diagnóstico** | Logs sob demanda, inspect, processos e shell interativo |
| **Imagens** | Listagem local, pull com progresso e pesquisa no Docker Hub |
| **Volumes e redes** | Listagem, criação e remoção |
| **Docker Swarm** | Serviços, tasks, nodes, stacks e scale de serviços replicated |
| **Eventos** | Janela recente de eventos do Docker Engine |
| **Segurança** | Read-only, confirmações, Security Audit e Apply Hardening Custom |
| **Auditoria local** | JSONL, sanitização, rotação e permissões restritas |
| **Interface** | Temas persistentes, i18n, teclado e mouse opcional |

### Detecção do ambiente

O endpoint conectado é classificado automaticamente:

| Endpoint | Comportamento |
|---|---|
| **Docker standalone** | Exibe e opera recursos do Engine local |
| **Swarm worker** | Exibe recursos locais e limita operações de cluster |
| **Swarm manager** | Disponibiliza serviços, tasks, nodes, stacks e plano de controle |

## Instalação

### Instalação rápida

```bash
sh -c "$(curl -fsSL https://docktop.dev/docktop.sh)"
```

O instalador:

1. detecta Linux `amd64` ou `arm64`;
2. consulta a release oficial mais recente;
3. valida o binário com SHA-256;
4. instala o executável;
5. oferece configuração de contexto remoto TLS;
6. pode ativar a verificação diária de atualizações.

O destino padrão depende do usuário:

| Execução | Destino |
|---|---|
| **root** | `/usr/local/bin/docktop` |
| **usuário comum** | `~/.local/bin/docktop` |
| **personalizada** | Valor de `DOCKTOP_INSTALL_DIR` |

Para instalação não interativa:

```bash
DOCKTOP_YES=1 sh -c "$(curl -fsSL https://docktop.dev/docktop.sh)"
```

Para revisar o instalador antes de executar:

```bash
curl -fsSL https://docktop.dev/docktop.sh -o docktop.sh
less docktop.sh
sh docktop.sh
```

### Compilação manual

```bash
git clone https://github.com/gustavoohrodrigues/docktop.git
cd docktop

go test ./...
go build -o docktop ./cmd/docktop
./docktop
```

Para instalar o build local no sistema:

```bash
sudo install -Dm755 docktop /usr/local/bin/docktop
docktop --version
```

## Primeiro uso

Por padrão, o DockTop usa:

```text
unix:///var/run/docker.sock
```

Confirme o acesso antes de iniciar:

```bash
docker info
docktop
```

Se o usuário precisar acessar o socket pelo grupo `docker`:

```bash
sudo usermod -aG docker "$USER"
```

Encerre a sessão e entre novamente após alterar os grupos.

> [!WARNING]
> Acesso ao grupo `docker` equivale, na prática, a privilégios de root no host.

## Uso

```bash
# Contexto padrão
docktop

# Bloquear operações de escrita
docktop --read-only

# Selecionar contexto, tema ou idioma
docktop --context production-manager
docktop --theme nord
docktop --language en-US

# Configuração alternativa
docktop --config /etc/docktop/config.yaml

# Desabilitar mouse
docktop --no-mouse
```

Consulte todas as opções:

```bash
docktop --help
```

## Atalhos principais

| Tecla | Ação |
|---|---|
| `Tab` / `Shift+Tab` | Alternar entre áreas |
| `←` / `→` | Navegar entre módulos |
| `↑` / `↓` ou `j` / `k` | Navegar em listas |
| `PgUp` / `PgDn` ou `Ctrl+U` / `Ctrl+D` | Rolar uma página |
| `g` / `G` | Ir ao primeiro ou último item |
| `Enter` | Abrir detalhes ou confirmar |
| `Esc` | Fechar modal ou retornar |
| `/` | Pesquisa contextual |
| `r` | Atualizar a tela |
| `R` | Alternar auto-refresh |
| `x` | Reiniciar o container selecionado |
| `u` | Atualizar a imagem do container |
| `a` | Executar Security Audit somente leitura |
| `H` | Abrir Apply Hardening |
| `t` | Trocar tema |
| `L` | Selecionar idioma |
| `?` / `F1` | Abrir ajuda contextual |
| `q` | Sair |

Os atalhos disponíveis no contexto atual também aparecem no rodapé da TUI.

## Configuração

O arquivo padrão fica em:

```text
~/.config/docktop/config.yaml
```

Exemplo com socket local:

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

Exemplo de endpoint remoto com TLS:

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

O DockTop também considera `DOCKER_HOST`, `DOCKER_CONTEXT` e
`DOCKER_TLS_VERIFY` quando aplicável. Consulte
[`config.example.yaml`](config.example.yaml) para todas as opções.

## Docker Swarm

As operações disponíveis dependem do endpoint. Workers não recebem ações que
exigem o plano de controle. Em managers, o DockTop apresenta serviços, tasks,
nodes e stacks, além das ações implementadas para esses recursos.

Alterações sensíveis respeitam o modo `--read-only`, a política de ações
perigosas e as confirmações digitadas.

## Segurança e auditoria

### Security Audit

Na tela **Containers**, selecione um container e pressione `a`.

A auditoria usa apenas `ContainerInspect`: não inicia o container, não executa
processos dentro dele e não altera sua configuração. Os achados são
classificados como Critical, High, Medium, Low ou Informational.

As regras atuais verificam, entre outros pontos:

- execução como root e modo privilegiado;
- `no-new-privileges` e Linux capabilities;
- filesystem raiz gravável;
- mounts sensíveis, socket Docker e devices;
- namespaces host PID, IPC, network e user;
- limites de CPU, memória, swap, PIDs e `nofile`;
- portas publicadas e health check;
- seccomp, AppArmor e SELinux;
- possíveis secrets em variáveis de ambiente;
- referências mutáveis de imagem;
- identificação de Docker Compose e Swarm.

Valores de variáveis que aparentam conter credenciais são substituídos por
`[REDACTED]`.

### Apply Hardening

Em um container standalone, pressione `H`. O fluxo:

1. inspeciona novamente o container;
2. mostra controles já aplicados, ausentes ou parciais;
3. não seleciona controles silenciosamente;
4. permite selecionar itens com `Espaço`;
5. apresenta valores atuais, propostas e riscos de compatibilidade;
6. gera um diff antes/depois;
7. exige confirmação pelo nome exato do container;
8. preserva o original como backup;
9. cria e valida o substituto;
10. restaura o original se criação, startup ou validação falhar.

O backup usa o formato:

```text
<nome>.docktop-before-hardening-<timestamp>
```

O backup não é removido automaticamente após sucesso.

Controles disponíveis na v0.3.7:

- `no-new-privileges`;
- desabilitar privileged mode;
- drop de todas as Linux capabilities;
- filesystem raiz somente leitura;
- usuário não-root `65532:65532`;
- limites de CPU, memória, swap, PIDs e `nofile`;
- namespaces privados de PID, IPC e rede;
- remoção do socket Docker e de device mappings;
- seccomp padrão e AppArmor `docker-default`;
- remoção do user namespace do host;
- `tmpfs` restrito para `/tmp` e `/run`;
- conversão de bind mounts sensíveis para somente leitura;
- remoção explícita de portas publicadas.

> [!WARNING]
> Containers gerenciados por Docker Compose e tasks de Docker Swarm não podem
> receber hardening por recriação direta. Essa proteção evita divergência da
> configuração declarativa.

### Auditoria local

- arquivo padrão: `~/.local/share/docktop/audit.jsonl`;
- permissões `0600` no arquivo e `0700` no diretório;
- sanitização de valores sensíveis;
- rotação e retenção configuráveis;
- credenciais TLS não são registradas.

## Atualizações

Quando ativada durante a instalação, a verificação de atualizações é carregada
pelo perfil do shell no máximo uma vez por dia. O DockTop consulta a release
oficial, informa a versão disponível e pede confirmação antes de atualizar.

Para desabilitar a verificação:

```bash
export DOCKTOP_NO_UPDATE_CHECK=1
```

## Arquitetura

```text
cmd/docktop          CLI e ciclo de vida
internal/app         Composição da aplicação
internal/config      Configuração e contextos
internal/docker      Integração com Docker Engine
internal/i18n        Traduções e localização
internal/jobs        Operações em background
internal/audit       Auditoria local
internal/security    Regras de auditoria e hardening
internal/registry    Integração com registries
internal/theme       Temas e cores semânticas
internal/ui          Estado e renderização Bubble Tea
internal/utils       Utilidades e sanitização
data/themes          Temas distribuídos
docs/assets          Recursos da documentação
```

```text
Bubble Tea UI → interface Engine → Docker SDK → Docker Engine
```

## Desenvolvimento

```bash
gofmt -w $(find cmd internal -name '*.go')
go mod tidy
go test ./...
go test -race ./...
go build -o docktop ./cmd/docktop
```

Ou:

```bash
make test
make build
make run
```

## Compatibilidade

- Linux `amd64`;
- Linux `arm64`;
- Docker Engine por socket Unix;
- Docker Engine remoto por TCP/TLS;
- Docker standalone;
- Docker Swarm worker e manager.

## Limitações atuais

- Balanced, Strict, Restore Previous Configuration, Compose overrides, Trivy,
  image hardening, SBOM e scan history ainda não estão disponíveis.
- O backup anterior é preservado, mas o histórico versionado e a ação Restore
  Previous Configuration ainda não foram implementados.
- A auditoria é estática e não executa o workload.
- Logs são carregados sob demanda, sem follow contínuo.
- O shell interativo usa o Docker CLI para controlar o raw TTY.
- Update avançado, rollback e remoção de serviços Swarm não estão expostos.
- Promote e demote de managers ainda não estão disponíveis na TUI.
- Stacks são agrupadas e inspecionadas, mas deploy e remoção não estão
  implementados.
- Events usa uma janela recente, não um stream persistente.

## Solução de problemas

<details>
<summary><strong>Permissão negada no socket Docker</strong></summary>

```bash
id
ls -l /var/run/docker.sock
docker info
```

</details>

<details>
<summary><strong>Docker daemon indisponível</strong></summary>

```bash
sudo systemctl status docker
sudo systemctl start docker
```

</details>

<details>
<summary><strong>Falha no contexto remoto TLS</strong></summary>

Verifique DNS, CA, certificado do cliente, chave privada, permissões dos
arquivos e se o daemon está escutando no endpoint configurado.

</details>

## Contribuindo

Contribuições são bem-vindas.

1. Descreva claramente o problema ou comportamento desejado.
2. Mantenha operações Docker fora da camada de UI.
3. Adicione testes para regras de negócio e parsing.
4. Execute `go test ./...` e `go build ./cmd/docktop`.
5. Não adicione mocks ou ações decorativas ao fluxo de produção.

Para mudanças maiores, abra uma issue ou discussão antes do pull request.

## Publicação de versões

O workflow [`.github/workflows/release.yml`](.github/workflows/release.yml):

1. valida a tag semântica;
2. executa os testes;
3. compila Linux `amd64` e `arm64`;
4. gera o manifesto SHA-256;
5. atesta a procedência dos binários;
6. publica a GitHub Release.

Exemplo:

```bash
git tag -a v0.3.7 -m "DockTop v0.3.7"
git push origin v0.3.7
```

O instalador consome diretamente a release oficial mais recente. O
`docktop-website` sincroniza seu espelho de downloads por um workflow próprio.

## Equipe

<div align="center">

<a href="https://github.com/orqly">
  <img src="https://github.com/orqly.png?size=320" width="260" alt="Logo da Orqly">
</a>

Um projeto da [**Orqly**](https://www.linkedin.com/company/orqly/).

</div>

<table>
  <tr>
    <td align="center">
      <a href="https://github.com/EmanuelSena101">
        <img src="https://github.com/EmanuelSena101.png?size=320" width="110" alt="Foto de perfil de Emanuel Sena">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Emanuel Sena</strong><br>
      <a href="https://github.com/EmanuelSena101">@EmanuelSena101</a><br>
      <a href="https://www.linkedin.com/in/emanuel-sena/">
        <img src="https://img.shields.io/badge/LinkedIn-0A66C2?style=flat-square&logo=linkedin&logoColor=white" alt="LinkedIn de Emanuel Sena">
      </a>
    </td>
    <td align="center">
      <a href="https://github.com/otaviozin">
        <img src="https://github.com/otaviozin.png?size=320" width="110" alt="Foto de perfil de Otávio">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Otávio</strong><br>
      <a href="https://github.com/otaviozin">@otaviozin</a><br>
      <a href="https://www.linkedin.com/in/otavio-ppereira/">
        <img src="https://img.shields.io/badge/LinkedIn-0A66C2?style=flat-square&logo=linkedin&logoColor=white" alt="LinkedIn de Otávio">
      </a>
    </td>
    <td align="center">
      <a href="https://github.com/Lucas-V-Roveri">
        <img src="https://github.com/Lucas-V-Roveri.png?size=320" width="110" alt="Foto de perfil de Lucas Roveri">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Lucas Roveri</strong><br>
      <a href="https://github.com/Lucas-V-Roveri">@Lucas-V-Roveri</a><br>
      <a href="https://www.linkedin.com/in/lucas-vilela-roveri-dev/">
        <img src="https://img.shields.io/badge/LinkedIn-0A66C2?style=flat-square&logo=linkedin&logoColor=white" alt="LinkedIn de Lucas Roveri">
      </a>
    </td>
    <td align="center">
      <a href="https://github.com/gustavoohrodrigues">
        <img src="https://github.com/gustavoohrodrigues.png?size=320" width="110" alt="Foto de perfil de Gustavo Rodrigues">
      </a><br>
      <sub><strong>CRIADOR E MANTENEDOR</strong></sub><br>
      <strong>Gustavo Rodrigues</strong><br>
      <a href="https://github.com/gustavoohrodrigues">@gustavoohrodrigues</a><br>
      <a href="https://www.linkedin.com/in/gustavo-henrique-rodrigues-3070a5260">
        <img src="https://img.shields.io/badge/LinkedIn-0A66C2?style=flat-square&logo=linkedin&logoColor=white" alt="LinkedIn de Gustavo Rodrigues">
      </a>
    </td>
    <td align="center">
      <a href="https://github.com/matjsz">
        <img src="https://github.com/matjsz.png?size=320" width="110" alt="Foto de perfil de Matheus Silva">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Matheus Silva</strong><br>
      <a href="https://github.com/matjsz">@matjsz</a><br>
      <a href="https://www.linkedin.com/in/matjsilva/">
        <img src="https://img.shields.io/badge/LinkedIn-0A66C2?style=flat-square&logo=linkedin&logoColor=white" alt="LinkedIn de Matheus Silva">
      </a>
    </td>
  </tr>
</table>

## Licença

Distribuído sob a licença [MIT](LICENSE).

---

<div align="center">

[**docktop.dev**](https://docktop.dev/) •
[**GitHub**](https://github.com/gustavoohrodrigues/docktop)

By [**Orqly**](https://www.linkedin.com/company/orqly/).

</div>
