# Contribuindo com o DockTop

Obrigado pelo interesse em contribuir. Este documento reúne as orientações para
preparar o ambiente, desenvolver mudanças e enviar pull requests.

## Formas de contribuir

São bem-vindas contribuições em:

- correções de bugs;
- novas funcionalidades;
- testes;
- documentação;
- traduções;
- acessibilidade e experiência no terminal;
- melhorias de segurança e confiabilidade.

Para mudanças grandes ou que alterem o comportamento público, abra uma issue ou
discussão antes de iniciar a implementação. Isso permite alinhar escopo,
compatibilidade e estratégia de testes.

Falhas de segurança não devem ser publicadas em issues. Siga as instruções de
[`SECURITY.md`](SECURITY.md).

## Preparação do ambiente

Requisitos:

- Linux;
- Go na versão indicada pelo [`go.mod`](go.mod);
- Git;
- Docker Engine para testes de integração e validação manual.

Clone o projeto:

```bash
git clone https://github.com/gustavoohrodrigues/docktop.git
cd docktop
go mod download
```

Confirme o ambiente:

```bash
go version
docker info
go test ./...
```

O acesso ao socket Docker concede privilégios elevados no host. Use um ambiente
de desenvolvimento isolado sempre que possível.

## Organização do código

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
```

Diretrizes:

- mantenha chamadas e operações Docker fora de `internal/ui`;
- implemente regras de auditoria e hardening em `internal/security`;
- preserve a interface Engine entre a UI e o Docker SDK;
- mantenha operações demoradas em `internal/jobs`;
- não exponha credenciais, certificados ou variáveis sensíveis em logs;
- traduza mensagens visíveis para Português, Inglês e Espanhol;
- não adicione dados, dashboards ou ações decorativas que não existam no fluxo
  real da aplicação.

## Desenvolvimento

Comandos principais:

```bash
make build
make run
make test
make lint
make tidy
```

Equivalentes em Go:

```bash
gofmt -w $(find cmd internal -name '*.go')
go mod tidy
go test ./...
go test -race ./...
go build -o docktop ./cmd/docktop
```

`make lint` requer `golangci-lint` instalado.

Não inclua no commit o binário local `docktop`, arquivos temporários,
credenciais, certificados ou configurações específicas do seu ambiente.

## Testes

Toda regra nova deve ter testes proporcionais ao risco da mudança.

- parsing e validação devem incluir entradas válidas e inválidas;
- operações Docker devem testar erros, cancelamento e estados parciais;
- mudanças de segurança devem testar detecção, proposta, compatibilidade e
  rollback;
- alterações de i18n devem manter as três localidades suportadas;
- correções de bugs devem incluir um teste de regressão quando possível.

Antes de abrir o pull request:

```bash
gofmt -w $(find cmd internal -name '*.go')
go test ./...
go test -race ./...
go build ./cmd/docktop
```

Se algum teste depender de Docker ou de uma característica específica do
sistema, documente essa condição no pull request.

## Pull requests

Um pull request deve:

1. ter escopo claro;
2. explicar o problema e a solução;
3. informar como a mudança foi testada;
4. descrever riscos ou incompatibilidades;
5. incluir testes para novas regras;
6. atualizar README, exemplos ou ajuda contextual quando necessário;
7. incluir traduções PT-BR, EN-US e ES para novos textos visíveis;
8. não misturar refatorações sem relação com a mudança principal.

Para alterações visuais, inclua capturas apenas de telas que correspondam ao
comportamento real da aplicação.

## Commits

Prefira commits pequenos, objetivos e revisáveis. Cada commit deve representar
uma alteração coerente e deixar o projeto compilável sempre que possível.

Não reescreva o histórico de branches compartilhadas e não inclua alterações de
formatação sem relação com o objetivo do pull request.

## Segurança

- nunca envie tokens, chaves privadas, certificados ou senhas;
- não registre valores de variáveis potencialmente sensíveis;
- preserve as permissões restritas dos arquivos de auditoria;
- trate dados vindos do daemon como entrada não confiável;
- mantenha confirmações para ações destrutivas;
- respeite o modo `--read-only`;
- não publique detalhes exploráveis em issues.

Consulte [`SECURITY.md`](SECURITY.md) para relatar vulnerabilidades.

## Publicação de versões

> [!IMPORTANT]
> Esta seção é destinada somente aos mantenedores. Colaboradores não devem
> criar ou enviar tags de release.

O DockTop utiliza versionamento semântico no formato `vMAJOR.MINOR.PATCH`.

Antes de publicar:

1. confirme que o `main` está atualizado e limpo;
2. execute testes, race detector e build;
3. revise README, changelog e limitações;
4. escolha a versão conforme o impacto da mudança;
5. confirme que a tag ainda não existe.

Validação local:

```bash
git status --short
go test ./...
go test -race ./...
go build ./cmd/docktop
git tag --list
```

Criação da tag:

```bash
git tag -a vX.Y.Z -m "DockTop vX.Y.Z"
git push origin vX.Y.Z
```

O workflow [`.github/workflows/release.yml`](.github/workflows/release.yml):

1. valida a tag semântica;
2. executa os testes;
3. compila Linux `amd64` e `arm64`;
4. gera o manifesto com checksums SHA-256;
5. atesta a procedência dos binários;
6. publica a GitHub Release.

Após a publicação:

1. confirme que o workflow terminou com sucesso;
2. valide os binários e o `manifest.txt` da release;
3. teste o instalador oficial em diretório temporário;
4. confirme que `docktop --version` retorna a versão publicada;
5. verifique a sincronização do `docktop-website`;
6. confirme o manifesto e o instalador em `docktop.dev`.

O instalador consome diretamente a release oficial mais recente. O
`docktop-website` mantém seu espelho por um workflow próprio, sem depender de
tokens entre repositórios.
