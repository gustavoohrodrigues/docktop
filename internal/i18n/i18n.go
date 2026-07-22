package i18n

import "strings"

type Language struct{ Code, Label string }

var supported = []Language{{"pt-BR", "Português (Brasil)"}, {"en-US", "English (United States)"}, {"es", "Español"}}

func Languages() []Language {
	out := make([]Language, len(supported))
	copy(out, supported)
	return out
}
func Normalize(code string) string {
	v := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "_", "-"))
	switch v {
	case "en", "en-us":
		return "en-US"
	case "es", "es-es", "es-419":
		return "es"
	default:
		return "pt-BR"
	}
}

var catalog = map[string]map[string]string{
	"pt-BR": {"dashboard": "Dashboard", "containers": "Containers", "images": "Imagens", "registry": "Registry", "volumes": "Volumes", "networks": "Redes", "swarm": "Swarm", "services": "Serviços", "nodes": "Nós", "stacks": "Stacks", "events": "Eventos", "audit": "Auditoria", "settings": "Configurações", "help": "ajuda", "tabs": "abas", "refresh": "atualizar", "theme": "tema", "quit": "sair", "language": "idioma", "select_language": "selecionar idioma", "effective_config": "CONFIGURAÇÃO EFETIVA", "default_context": "contexto padrão", "endpoint": "endpoint", "auto_refresh": "auto refresh", "read_only": "somente leitura", "mouse": "mouse", "telemetry": "telemetria", "file": "Arquivo", "confirm": "Enter seleciona · Esc cancela"},
	"en-US": {"dashboard": "Dashboard", "containers": "Containers", "images": "Images", "registry": "Registry", "volumes": "Volumes", "networks": "Networks", "swarm": "Swarm", "services": "Services", "nodes": "Nodes", "stacks": "Stacks", "events": "Events", "audit": "Audit", "settings": "Settings", "help": "help", "tabs": "tabs", "refresh": "refresh", "theme": "theme", "quit": "quit", "language": "language", "select_language": "select language", "effective_config": "EFFECTIVE CONFIGURATION", "default_context": "default context", "endpoint": "endpoint", "auto_refresh": "auto refresh", "read_only": "read-only", "mouse": "mouse", "telemetry": "telemetry", "file": "File", "confirm": "Enter selects · Esc cancels"},
	"es":    {"dashboard": "Panel", "containers": "Contenedores", "images": "Imágenes", "registry": "Registro", "volumes": "Volúmenes", "networks": "Redes", "swarm": "Swarm", "services": "Servicios", "nodes": "Nodos", "stacks": "Stacks", "events": "Eventos", "audit": "Auditoría", "settings": "Ajustes", "help": "ayuda", "tabs": "pestañas", "refresh": "actualizar", "theme": "tema", "quit": "salir", "language": "idioma", "select_language": "seleccionar idioma", "effective_config": "CONFIGURACIÓN EFECTIVA", "default_context": "contexto predeterminado", "endpoint": "endpoint", "auto_refresh": "actualización automática", "read_only": "solo lectura", "mouse": "mouse", "telemetry": "telemetría", "file": "Archivo", "confirm": "Enter selecciona · Esc cancela"}}

func T(language, key string) string {
	language = Normalize(language)
	if v := catalog[language][key]; v != "" {
		return v
	}
	if v := modules[language][key]; v != "" {
		return v
	}
	if v := catalog["pt-BR"][key]; v != "" {
		return v
	}
	if v := modules["pt-BR"][key]; v != "" {
		return v
	}
	return key
}
