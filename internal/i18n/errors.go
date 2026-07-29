package i18n

import "strings"

var errorTranslations = map[string]map[string]string{
	"en-US": {
		"digite ao menos 2 caracteres": "enter at least 2 characters", "Docker Hub indisponível": "Docker Hub unavailable", "limite de requisições do Docker Hub atingido": "Docker Hub rate limit reached", "apenas serviços replicated podem ser escalados": "only replicated services can be scaled", "availability inválida": "invalid availability", "container pertence a um serviço Swarm; atualize a imagem na tela Services para preservar o estado desejado": "container belongs to a Swarm service; update the image in Services to preserve desired state", "container não possui referência de imagem atualizável": "container has no updatable image reference", "inspecionar imagem baixada": "inspect pulled image", "parar container antigo": "stop old container", "preparar substituição": "prepare replacement", "criar substituto; rollback aplicado": "create replacement; rollback applied", "iniciar substituto; rollback aplicado": "start replacement; rollback applied", "copiar configuração do container": "copy container configuration", "use exatamente 7 campos separados por |; o comando não pode conter |": "use exactly 7 fields separated by |; command cannot contain |", "nome e imagem são obrigatórios": "name and image are required", "imagem é obrigatória": "image is required", "nome do container é obrigatório": "container name is required", "imagem não existe localmente e o pull falhou": "image is not available locally and pull failed", "portas inválidas": "invalid ports", "criado, mas não iniciado": "created but not started", "container não está em execução": "container is not running", "Docker CLI é necessário para o terminal interativo": "Docker CLI is required for the interactive terminal", "container não possui bash, sh ou ash": "container does not have bash, sh or ash", "usuário não possui acesso ao endpoint": "user cannot access endpoint", "Docker daemon não está ativo ou o socket não existe": "Docker daemon is not running or the socket does not exist", "falha na validação TLS do endpoint remoto": "remote endpoint TLS validation failed", "Docker indisponível": "Docker unavailable"},
	"es": {
		"digite ao menos 2 caracteres": "escriba al menos 2 caracteres", "Docker Hub indisponível": "Docker Hub no disponible", "limite de requisições do Docker Hub atingido": "límite de solicitudes de Docker Hub alcanzado", "apenas serviços replicated podem ser escalados": "solo se pueden escalar servicios replicados", "availability inválida": "disponibilidad inválida", "container pertence a um serviço Swarm; atualize a imagem na tela Services para preservar o estado desejado": "el contenedor pertenece a un servicio Swarm; actualice la imagen en Servicios para preservar el estado deseado", "container não possui referência de imagem atualizável": "el contenedor no tiene una referencia de imagen actualizable", "inspecionar imagem baixada": "inspeccionar imagen descargada", "parar container antigo": "detener contenedor anterior", "preparar substituição": "preparar sustitución", "criar substituto; rollback aplicado": "crear sustituto; rollback aplicado", "iniciar substituto; rollback aplicado": "iniciar sustituto; rollback aplicado", "copiar configuração do container": "copiar configuración del contenedor", "use exatamente 7 campos separados por |; o comando não pode conter |": "use exactamente 7 campos separados por |; el comando no puede contener |", "nome e imagem são obrigatórios": "nombre e imagen son obligatorios", "imagem é obrigatória": "la imagen es obligatoria", "nome do container é obrigatório": "el nombre del contenedor es obligatorio", "imagem não existe localmente e o pull falhou": "la imagen no existe localmente y la descarga falló", "portas inválidas": "puertos inválidos", "criado, mas não iniciado": "creado, pero no iniciado", "container não está em execução": "el contenedor no está en ejecución", "Docker CLI é necessário para o terminal interativo": "Docker CLI es necesario para el terminal interactivo", "container não possui bash, sh ou ash": "el contenedor no tiene bash, sh ni ash", "usuário não possui acesso ao endpoint": "el usuario no tiene acceso al endpoint", "Docker daemon não está ativo ou o socket não existe": "Docker daemon no está activo o el socket no existe", "falha na validação TLS do endpoint remoto": "falló la validación TLS del endpoint remoto", "Docker indisponível": "Docker no disponible"}}

func LocalizeError(language, text string) string {
	language = Normalize(language)
	for from, to := range errorTranslations[language] {
		text = strings.ReplaceAll(text, from, to)
	}
	for from, translations := range securityErrorTranslations {
		to := translations[0]
		if language == "es" {
			to = translations[1]
		} else if language == "pt-BR" {
			to = from
		}
		text = strings.ReplaceAll(text, from, to)
	}
	return text
}

var securityErrorTranslations = map[string][2]string{
	"inspeção do container retornou configuração incompleta":               {"container inspection returned incomplete configuration", "la inspección del contenedor devolvió una configuración incompleta"},
	"container gerenciado por Docker Compose":                              {"container is managed by Docker Compose", "contenedor gestionado por Docker Compose"},
	"hardening direto foi bloqueado para não divergir do projeto Compose":  {"direct hardening was blocked to avoid diverging from the Compose project", "el hardening directo fue bloqueado para no divergir del proyecto Compose"},
	"gere um override antes de aplicar hardening":                          {"generate an override before applying hardening", "genere un override antes de aplicar hardening"},
	"container pertence a um serviço Swarm":                                {"container belongs to a Swarm service", "el contenedor pertenece a un servicio Swarm"},
	"altere o service spec em vez de recriar a task":                       {"change the service spec instead of recreating the task", "cambie el service spec en lugar de recrear la task"},
	"hardening deve alterar o service spec":                                {"hardening must change the service spec", "el hardening debe cambiar el service spec"},
	"container mudou desde a inspeção; execute o hardening novamente":      {"container changed since inspection; run hardening again", "el contenedor cambió desde la inspección; ejecute el hardening nuevamente"},
	"container mudou durante a operação; configuração original preservada": {"container changed during the operation; original configuration preserved", "el contenedor cambió durante la operación; configuración original preservada"},
	"selecione ao menos um controle de hardening":                          {"select at least one hardening control", "seleccione al menos un control de hardening"},
	"controle de hardening desconhecido":                                   {"unknown hardening control", "control de hardening desconocido"},
	"os controles selecionados já estão aplicados":                         {"the selected controls are already applied", "los controles seleccionados ya están aplicados"},
	"parar container original":                                             {"stop original container", "detener contenedor original"},
	"preservar container original":                                         {"preserve original container", "preservar contenedor original"},
	"container substituto não permaneceu em execução":                      {"replacement container did not remain running", "el contenedor sustituto no permaneció en ejecución"},
	"health check do substituto retornou unhealthy":                        {"replacement health check returned unhealthy", "el health check del sustituto devolvió unhealthy"},
	"tempo esgotado aguardando health check do substituto":                 {"timed out waiting for replacement health check", "se agotó el tiempo esperando el health check del sustituto"},
	"criar substituto":                         {"create replacement", "crear sustituto"},
	"iniciar substituto":                       {"start replacement", "iniciar sustituto"},
	"validar substituto":                       {"validate replacement", "validar sustituto"},
	"falhou; configuração original restaurada": {"failed; original configuration restored", "falló; configuración original restaurada"},
	"rollback também falhou":                   {"rollback also failed", "el rollback también falló"},
	"restaurar nome original":                  {"restore original name", "restaurar nombre original"},
	"reiniciar original":                       {"restart original", "reiniciar original"},
	"preservar/remover substituto falho":       {"preserve/remove failed replacement", "preservar/eliminar sustituto fallido"},
}
