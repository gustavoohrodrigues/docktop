<div align="center">

<img src="docs/assets/docktop-whale.png" alt="Baleia pixel art do DockTop carregando containers" width="680">

# DockTop

### Docker e Docker Swarm, direto do terminal.

Uma TUI rápida e segura, escrita em Go, para observar e operar ambientes Docker  
sem sair do terminal — do container local ao cluster Swarm.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-amd64%20%7C%20arm64-FCC624?style=flat-square&logo=linux&logoColor=black)](#compatibilidade)
[![Docker](https://img.shields.io/badge/Docker-Engine%20%7C%20Swarm-2496ED?style=flat-square&logo=docker&logoColor=white)](#docker-swarm)
[![Status](https://img.shields.io/badge/status-em%20produ%C3%A7%C3%A3o-22C55E?style=flat-square)](#status-do-projeto)
[![License](https://img.shields.io/badge/licen%C3%A7a-MIT-56F39A?style=flat-square)](LICENSE)

[**Site oficial ↗**](https://docktop.dev/) •
[**Documentação ↗**](https://docktop.dev/#docs) •
[**Instalar ↗**](https://docktop.dev/#install) •
[**GitHub ↗**](https://github.com/gustavoohrodrigues/docktop)

[Visão geral](#visão-geral) •
[Funcionalidades](#funcionalidades) •
[Instalação](#instalação) •
[Uso](#uso) •
[Configuração](#configuração) •
[Equipe](#equipe)

</div>

---

## Visão geral

O **DockTop** simplifica a rotina de quem administra Docker em servidores Linux e prefere continuar no terminal. Ele se conecta diretamente à API do Docker Engine, organiza os recursos em telas navegáveis e oferece proteções para operações que podem causar indisponibilidade ou perda de dados.

Não há servidor web, banco de dados ou agente adicional instalado no host. O DockTop é distribuído como um único binário e pode se conectar tanto ao socket Docker local quanto a um daemon remoto.

### Por que usar o DockTop?

- **Tudo em um só lugar:** containers, imagens, volumes, redes, eventos e recursos Swarm.
- **Operação mais segura:** modo somente leitura, confirmações e auditoria local.
- **Feito para servidores:** interface leve, responsiva e adaptada a terminais menores.
- **Sem infraestrutura extra:** basta o binário e acesso ao Docker Engine.
- **Pronto para diferentes ambientes:** Docker standalone, Swarm worker e Swarm manager.
- **Experiência localizada:** Português do Brasil, Inglês e Espanhol.

## Status do projeto

O DockTop está **em produção** e continua em evolução ativa.

> [!IMPORTANT]
> Antes de usar operações de escrita em ambientes críticos, conheça as
> [limitações atuais](#limitações-atuais) e considere iniciar com `--read-only`.

## Funcionalidades

| Área | Recursos |
|---|---|
| **Dashboard** | Informações do Engine, uso de CPU e memória e resumo dos recursos |
| **Containers** | Métricas, criação, start, stop, restart, pause e remoção |
| **Atualizações** | Atualização de imagem com recriação do container e rollback |
| **Diagnóstico** | Logs, inspect, processos e shell interativo |
| **Imagens** | Listagem local, pull com progresso e pesquisa no Docker Hub |
| **Storage e rede** | Criação e remoção de volumes e redes |
| **Docker Swarm** | Serviços, tasks, nodes, stacks e scale de serviços replicated |
| **Eventos** | Visualização dos eventos recentes do Docker Engine |
| **Segurança** | Modo somente leitura, confirmações e proteção de ações perigosas |
| **Auditoria** | Registro local em JSONL, sanitização, rotação e permissões restritas |
| **Experiência** | Temas persistentes, i18n, teclado e suporte opcional a mouse |

### Detecção automática do ambiente

O DockTop identifica automaticamente o tipo de endpoint conectado:

- Docker standalone;
- Swarm worker;
- Swarm manager.

A interface disponibiliza apenas as operações compatíveis com o ambiente e com as permissões do endpoint atual.

## Instalação

### Instalação rápida

O instalador oficial detecta automaticamente a arquitetura do sistema e instala o binário correto para Linux `amd64` ou `arm64`:

```bash
curl -fsSL https://docktop.dev/docktop.sh | sh
```

Por padrão, o DockTop é instalado em `~/.local/bin/docktop`. Caso o diretório ainda não esteja no `PATH`, o instalador configura o perfil do shell e orienta a abertura de um novo terminal.

Para instalações não interativas:

```bash
curl -fsSL https://docktop.dev/docktop.sh | DOCKTOP_YES=1 sh
```

> [!TIP]
> Se preferir revisar o instalador antes de executá-lo:
>
> ```bash
> curl -fsSL https://docktop.dev/docktop.sh -o docktop.sh
> less docktop.sh
> sh docktop.sh
> ```

Consulte também a página de [instalação no site oficial](https://docktop.dev/#install).

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
<summary><strong>Ubuntu / Debian</strong></summary>

```bash
sudo apt update
sudo apt install golang-go make git
```

</details>

<details>
<summary><strong>Fedora</strong></summary>

```bash
sudo dnf install golang make git
```

</details>

<details>
<summary><strong>RHEL / Rocky Linux / AlmaLinux</strong></summary>

```bash
sudo dnf install golang make git
```

</details>

## Primeiro uso

Por padrão, o DockTop usa o socket local:

```text
unix:///var/run/docker.sock
```

O usuário precisa ter permissão para acessar esse socket. Normalmente, isso é feito por meio do grupo `docker`:

```bash
sudo usermod -aG docker "$USER"
```

Depois da alteração, encerre a sessão e entre novamente. Confirme o acesso antes de iniciar:

```bash
docker info
docktop
```

> [!WARNING]
> O grupo `docker` concede, na prática, privilégios equivalentes a root no host.
> Não adicione usuários ao grupo sem compreender esse impacto.

## Uso

```bash
# Iniciar com o contexto padrão
docktop

# Impedir qualquer mutação
docktop --read-only

# Escolher contexto, tema ou idioma
docktop --context production-manager
docktop --theme nord
docktop --language en-US

# Usar um arquivo de configuração alternativo
docktop --config /etc/docktop/config.yaml

# Desabilitar o mouse
docktop --no-mouse
```

Para consultar todas as opções:

```bash
docktop --help
```

## Atalhos principais

| Tecla | Ação |
|---|---|
| `Tab` / `Shift+Tab` | Ir para a próxima área ou voltar à anterior |
| `←` / `→` | Navegar entre módulos |
| `↑` / `↓` ou `j` / `k` | Navegar em listas |
| `PgUp` / `PgDn` ou `Ctrl+U` / `Ctrl+D` | Rolar uma página |
| `g` / `G` | Ir ao primeiro ou último item |
| `Enter` | Abrir detalhes ou confirmar |
| `Esc` | Fechar modal ou retornar |
| `/` | Iniciar pesquisa contextual |
| `r` | Atualizar a tela atual |
| `R` | Ligar ou desligar o auto-refresh |
| `x` | Reiniciar o container selecionado |
| `u` | Atualizar a imagem do container |
| `t` | Trocar o tema |
| `L` | Escolher o idioma em Settings |
| `?` / `F1` | Abrir a ajuda |
| `q` | Sair |
| `a` | Executar auditoria de segurança somente leitura no container |
| `H` | Selecionar e aplicar controles de hardening com recriação e rollback |

Os atalhos disponíveis em cada contexto também aparecem no rodapé da aplicação.

## Configuração

O arquivo de configuração padrão fica em:

```text
~/.config/docktop/config.yaml
```

Use o arquivo [config.example.yaml](config.example.yaml) como ponto de partida.

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

O comportamento do DockTop se adapta ao endpoint conectado:

| Endpoint | Comportamento |
|---|---|
| **Standalone** | Exibe e opera os recursos locais do Docker Engine |
| **Swarm worker** | Exibe recursos locais e informa as limitações de acesso ao cluster |
| **Swarm manager** | Disponibiliza serviços, tasks, nodes, stacks e operações do plano de controle |

Alterações em nodes e outras ações sensíveis respeitam o modo `--read-only`, a política de ações perigosas e as confirmações digitadas.

## Segurança e auditoria

Segurança operacional faz parte do fluxo da aplicação:

- na tela Containers, `a` executa explicitamente uma **Security Audit** somente leitura;
- `--read-only` bloqueia operações de escrita;
- remoções e mudanças críticas exigem confirmação;
- variáveis com nomes como `password`, `token`, `secret`, `key` e `credential` são mascaradas;
- credenciais TLS não são exibidas nem gravadas na auditoria;
- o histórico padrão fica em `~/.local/share/docktop/audit.jsonl`;
- arquivos de auditoria usam permissão `0600` e diretórios usam `0700`;
- rotação e retenção dos registros são configuráveis.

### Security Audit de containers

Selecione um container na tela **Containers** e pressione `a`. O DockTop usa
somente `ContainerInspect` da Docker Engine API: não inicia o container, não
executa processos dentro dele e não altera sua configuração.

O relatório classifica achados como Critical, High, Medium, Low ou
Informational e mostra, para cada item, o valor atual, risco, remediação,
possibilidade de aplicação automática, necessidade de recriação, impacto de
compatibilidade e a propriedade Docker usada como evidência. Valores de
variáveis de ambiente com nomes que aparentam conter credenciais são sempre
substituídos por `[REDACTED]`.

As regras atuais avaliam execução como root, modo privilegiado,
`no-new-privileges`, capabilities, filesystem raiz gravável, mounts sensíveis e
amplos, socket Docker, devices, namespaces host PID/IPC/network/user, limites de
CPU/memória/PIDs, portas publicadas, health check, seccomp, AppArmor/SELinux,
possíveis secrets em ambiente, referências mutáveis de imagem e labels Docker
Compose.

Depois dos achados, a auditoria explica todos os controles disponíveis no
Apply Hardening, incluindo limites de CPU, memória, swap, PIDs e `nofile`,
seccomp, AppArmor, user namespaces, `tmpfs`, mounts sensíveis, portas
publicadas, capabilities, usuário e filesystem raiz. Cada controle mostra:

- `[✓] já aplicado`;
- `[ ] não aplicado`;
- `[~] parcial/revisar`;
- valor detectado e valor proposto;
- benefício e risco de compatibilidade.

Essa seção é dinâmica: cada execução faz um novo `ContainerInspect`. Portanto,
após aplicar hardening, uma nova auditoria mostra os controles efetivamente
presentes na configuração recriada, em vez de confiar no plano anterior.

O relatório inclui uma pontuação explicável de hardening em runtime. Cada
dedução corresponde a um achado visível; a pontuação não mede vulnerabilidades
da imagem e uma pontuação alta não garante segurança.

> [!WARNING]
> A auditoria é uma análise estática da configuração observável. Ela não garante
> compatibilidade de uma remediação nem prova que o container ou a imagem são
> seguros. A maioria dos controles de runtime exige recriação do container.

### Apply Hardening

Selecione um container standalone e pressione `H`. O DockTop:

1. inspeciona novamente a configuração;
2. mostra todos os controles disponíveis sem selecionar nenhum silenciosamente;
   cada controle informa **já aplicado**, **não aplicado** ou
   **parcial/revisar**;
3. permite selecionar ou desmarcar cada controle com Espaço;
4. mostra o valor atual, valor proposto, benefício, risco de compatibilidade,
   suporte de sistema e incompatibilidade provável;
5. gera um diff antes/depois;
6. exige confirmação digitando exatamente o nome do container;
7. para e renomeia o original para
   `<nome>.docktop-before-hardening-<timestamp>`;
8. cria o substituto preservando imagem, comando, ambiente, mounts, volumes,
   portas, restart policy, health check, logging e aliases de rede;
9. inicia e valida que o substituto permanece em execução e, quando disponível,
   aguarda o health check;
10. restaura automaticamente o original se criação, startup ou validação falhar.

O backup original nunca é removido automaticamente após sucesso.

Controles selecionáveis atualmente:

- `no-new-privileges`;
- desabilitar privileged mode;
- drop de todas as Linux capabilities;
- root filesystem somente leitura;
- usuário não-root `65532:65532`;
- limite de 512 PIDs;
- limite de memória de 512 MiB;
- namespaces privados de PID, IPC e rede;
- remoção do socket Docker;
- remoção de device mappings;
- limite de CPU;
- limite combinado de memória/swap;
- limite `nofile`;
- proteção seccomp padrão;
- perfil AppArmor `docker-default`;
- remoção do user namespace do host;
- `tmpfs` restrito para `/tmp` e `/run`;
- conversão de bind mounts sensíveis para somente leitura;
- remoção explícita de todas as portas publicadas.

Controles já aplicados aparecem com `[✓]`, permanecem visíveis para inspeção e
não podem ser selecionados novamente. Configurações existentes que não
correspondem exatamente à proposta — por exemplo, um perfil seccomp customizado
ou um tmpfs menos restrito — aparecem como **parcial/revisar**.

Os valores propostos de usuário, PIDs, CPU, memória, swap, `nofile` e `tmpfs`
são sempre mostrados antes da confirmação. Eles não são considerados
universalmente seguros ou compatíveis.

Containers gerenciados por Docker Compose e tasks de Docker Swarm são
bloqueados: recriá-los diretamente faria o estado divergir da configuração
declarativa. O suporte futuro deve gerar um Compose override ou alterar o
service spec correspondente.

### Próximas fases de segurança

- Vulnerability Scan com Trivy, JSON estruturado, execução local ou pela imagem
  oficial fixada, SBOM e histórico serão adicionados atrás de uma interface de
  scanner. Scans continuarão sendo executados somente quando solicitados.
- Trivy local não requer uma API key para scans comuns. Credenciais são
  opcionais para registries privados ou servidores remotos; como o DockTop ainda
  não possui armazenamento seguro de credenciais, elas deverão permanecer
  somente em memória até existir uma abstração apropriada.
- Balanced e Strict ainda serão adicionados sobre o plano Custom já disponível.
  Image hardening produzirá propostas de patch para Dockerfile e exigirá rebuild.
- Restore Previous Configuration manterá versões e fará recriação com validação
  e rollback. Containers Compose receberão propostas de override separadas, sem
  sobrescrever o arquivo Compose original.

Automated hardening não pode garantir compatibilidade. Uma varredura bem
sucedida não prova que a imagem é segura. Resultados de vulnerabilidades
dependem da atualização e cobertura da base do scanner. Restaurar uma
configuração antiga pode reintroduzir fraquezas.

## Arquitetura

```text
cmd/docktop          CLI e ciclo de vida
internal/app         Composição da aplicação
internal/config      Configuração e contextos
internal/docker      Integração com Docker Engine
internal/i18n        Traduções e localização de erros
internal/jobs        Operações em background
internal/audit       Auditoria local
internal/registry    Integração com registries
internal/theme       Temas e cores semânticas
internal/ui          Estado e renderização Bubble Tea
internal/utils       Utilidades e sanitização
data/themes          Temas distribuídos com o projeto
docs/assets          Recursos usados pela documentação
```

Fluxo principal:

```text
Bubble Tea UI → interface Engine → Docker SDK → Docker Engine
```

A interface do Engine mantém a camada visual desacoplada do SDK e permite testes sem inserir mocks no fluxo de produção.

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

- Linux amd64;
- Linux arm64;
- Docker Engine por socket Unix;
- Docker Engine remoto por TCP/TLS;
- Docker standalone;
- Docker Swarm worker e manager.

## Limitações atuais

- Security Audit e Apply Hardening Custom para containers standalone estão
  implementados. Balanced, Strict, Restore Previous Configuration, Compose
  overrides, Trivy, image hardening, SBOM e scan history ainda não estão
  disponíveis.
- O backup anterior é preservado como container, mas o histórico versionado e a
  ação Restore Previous Configuration ainda não foram implementados.
- A auditoria não executa o workload; portanto, requisitos de escrita,
  capabilities e portas são marcados para validação quando não podem ser
  inferidos com segurança.
- Logs são carregados sob demanda, ainda sem follow contínuo.
- O shell interativo ainda utiliza o Docker CLI para controlar o raw TTY; a detecção do shell usa a API.
- Update avançado, rollback e remoção de serviços Swarm ainda não estão expostos.
- Promote e demote de managers ainda não estão disponíveis na TUI.
- Stacks são agrupadas e inspecionadas, mas deploy e remoção ainda não estão implementados.
- Events usa uma janela recente em vez de um stream persistente.
- Alguns textos livres retornados pelo daemon, logs e labels permanecem no idioma original da fonte.

## Solução de problemas

<details>
<summary><strong>Permissão negada no socket Docker</strong></summary>

Confira o usuário, os grupos e as permissões:

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

Verifique os nomes DNS, a validade da CA, o certificado do cliente, a chave privada e as permissões dos arquivos. Confirme também se o daemon está escutando no endpoint configurado.

</details>

## Contribuindo

O DockTop será aberto para novas contribuições. Acompanhe o projeto e as novidades pelo [GitHub](https://github.com/gustavoohrodrigues/docktop).

Quando as contribuições externas estiverem abertas:

1. Descreva claramente o problema ou o comportamento desejado.
2. Mantenha operações Docker fora da camada de UI.
3. Adicione testes para regras de negócio e parsing.
4. Execute `go test ./...` e `go build ./cmd/docktop`.
5. Não adicione mocks ou ações decorativas ao fluxo de produção.

### Publicação de versões

O workflow `.github/workflows/release.yml` automatiza a publicação. Para
reduzir riscos de cadeia de suprimentos, as actions usadas estão fixadas por
commit e o pipeline opera com permissões explícitas. Os binários recebem
atestação de procedência e são publicados com checksums SHA-256.

Depois, crie e envie uma tag semântica:

```bash
git tag -a v0.3.1 -m "DockTop v0.3.1"
git push origin v0.3.1
```

O workflow testa o código, compila Linux `amd64` e `arm64`, publica a GitHub
Release e seu manifesto. O instalador do site consome a release oficial
diretamente, sem tokens entre repositórios. Releases existentes não são
sobrescritas.

CodeQL, Dependabot e `govulncheck` monitoram código e dependências. Relatos
sensíveis devem seguir a política em [`SECURITY.md`](SECURITY.md), sem abrir
detalhes exploráveis em uma issue pública.

## Equipe

<div align="center">

<a href="https://github.com/orqly">
  <img src="https://github.com/orqly.png?size=320" width="300" alt="Logo da Orqly">
</a>

<br>

Um projeto da [**Orqly**](https://www.linkedin.com/company/orqly/), desenvolvido por:

[GitHub ↗](https://github.com/orqly) • [LinkedIn ↗](https://www.linkedin.com/company/orqly/)

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
      <a href="https://www.linkedin.com/in/emanuel-sena/">LinkedIn ↗</a>
    </td>
    <td align="center">
      <a href="https://github.com/otaviozin">
        <img src="https://github.com/otaviozin.png?size=320" width="110" alt="Foto de perfil de Otávio">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Otávio</strong><br>
      <a href="https://github.com/otaviozin">@otaviozin</a><br>
      <a href="https://www.linkedin.com/in/otavio-ppereira/">LinkedIn ↗</a>
    </td>
    <td align="center">
      <a href="https://github.com/Lucas-V-Roveri">
        <img src="https://github.com/Lucas-V-Roveri.png?size=320" width="110" alt="Foto de perfil de Lucas Roveri">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Lucas Roveri</strong><br>
      <a href="https://github.com/Lucas-V-Roveri">@Lucas-V-Roveri</a><br>
      <a href="https://www.linkedin.com/in/lucas-vilela-roveri-dev/">LinkedIn ↗</a>
    </td>
    <td align="center">
      <a href="https://github.com/gustavoohrodrigues">
        <img src="https://github.com/gustavoohrodrigues.png?size=320" width="110" alt="Foto de perfil de Gustavo Rodrigues">
      </a><br>
      <sub><strong>CRIADOR E MANTENEDOR</strong></sub><br>
      <strong>Gustavo Rodrigues</strong><br>
      <a href="https://github.com/gustavoohrodrigues">@gustavoohrodrigues</a><br>
      <a href="https://www.linkedin.com/in/gustavo-henrique-rodrigues-3070a5260">LinkedIn ↗</a>
    </td>
    <td align="center">
      <a href="https://github.com/matjsz">
        <img src="https://github.com/matjsz.png?size=320" width="110" alt="Foto de perfil de Matheus Silva">
      </a><br>
      <sub><strong>CONTRIBUIDOR</strong></sub><br>
      <strong>Matheus Silva</strong><br>
      <a href="https://github.com/matjsz">@matjsz</a><br>
      <a href="https://www.linkedin.com/in/matjsilva/">LinkedIn ↗</a>
    </td>
  </tr>
</table>

## Licença

Distribuído sob a licença [MIT](LICENSE).

---

<div align="center">

[**docktop.dev**](https://docktop.dev/) • [**GitHub**](https://github.com/gustavoohrodrigues/docktop)

By [**Orqly**](https://www.linkedin.com/company/orqly/).

</div>
