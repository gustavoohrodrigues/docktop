package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type TLS struct {
	Enabled  bool   `yaml:"enabled"`
	CAFile   string `yaml:"ca_file"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}
type Context struct {
	Host        string `yaml:"host"`
	Description string `yaml:"description,omitempty"`
	TLS         TLS    `yaml:"tls"`
}
type Contexts map[string]Context

// UnmarshalYAML accepts both the current map format and the legacy list format:
//
//	contexts:
//	  local:
//	    host: unix:///var/run/docker.sock
//
//	contexts:
//	  - name: local
//	    host: unix:///var/run/docker.sock
func (c *Contexts) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var contexts map[string]Context
		if err := value.Decode(&contexts); err != nil {
			return err
		}
		*c = contexts
		return nil
	case yaml.SequenceNode:
		contexts := make(map[string]Context, len(value.Content))
		for index, item := range value.Content {
			var named struct {
				Name    string `yaml:"name"`
				Context `yaml:",inline"`
			}
			if err := item.Decode(&named); err != nil {
				return fmt.Errorf("contexts[%d]: %w", index, err)
			}
			name := strings.TrimSpace(named.Name)
			if name == "" {
				return fmt.Errorf("contexts[%d]: campo name é obrigatório", index)
			}
			if _, exists := contexts[name]; exists {
				return fmt.Errorf("contexts[%d]: contexto %q duplicado", index, name)
			}
			contexts[name] = named.Context
		}
		*c = contexts
		return nil
	default:
		return errors.New("contexts deve ser um mapa ou uma lista")
	}
}

type Audit struct {
	Enabled   bool   `yaml:"enabled"`
	Path      string `yaml:"path,omitempty"`
	MaxSizeMB int    `yaml:"max_size_mb"`
	Retention int    `yaml:"retention"`
}
type Dangerous struct {
	RemoveContainers bool `yaml:"remove_containers"`
	RemoveImages     bool `yaml:"remove_images"`
	RemoveVolumes    bool `yaml:"remove_volumes"`
	RemoveNetworks   bool `yaml:"remove_networks"`
	Prune            bool `yaml:"prune"`
	SwarmChanges     bool `yaml:"swarm_changes"`
}
type Registry struct {
	HubURL   string        `yaml:"hub_url"`
	PageSize int           `yaml:"page_size"`
	Timeout  time.Duration `yaml:"-"`
	TimeoutS string        `yaml:"timeout"`
}
type UI struct {
	CompactWidth     int `yaml:"compact_width"`
	RecommendedWidth int `yaml:"recommended_width"`
}
type Shell struct {
	Candidates []string `yaml:"candidates"`
	EscapeKey  string   `yaml:"escape_key"`
}
type Timeouts struct{ Connect, Operation, Stream string }
type Config struct {
	DefaultContext   string        `yaml:"default_context"`
	Contexts         Contexts      `yaml:"contexts"`
	RefreshInterval  time.Duration `yaml:"-"`
	Refresh          string        `yaml:"refresh_interval"`
	ReadOnly         bool          `yaml:"read_only"`
	Theme            string        `yaml:"theme"`
	Language         string        `yaml:"language"`
	MouseEnabled     bool          `yaml:"mouse_enabled"`
	TelemetryEnabled bool          `yaml:"telemetry_enabled"`
	Audit            Audit         `yaml:"audit"`
	DangerousActions Dangerous     `yaml:"dangerous_actions"`
	Registry         Registry      `yaml:"registry"`
	UI               UI            `yaml:"ui"`
	Shell            Shell         `yaml:"shell"`
	Timeouts         Timeouts      `yaml:"timeouts"`
	Debug            bool          `yaml:"debug"`
}

func Default() Config {
	return Config{DefaultContext: "local", Contexts: Contexts{"local": {Host: "unix:///var/run/docker.sock"}}, Refresh: "3s", RefreshInterval: 3 * time.Second, Theme: "dark-ops", Language: "pt-BR", MouseEnabled: true, Audit: Audit{Enabled: true, MaxSizeMB: 10, Retention: 5}, DangerousActions: Dangerous{true, true, true, true, true, true}, Registry: Registry{HubURL: "https://hub.docker.com", PageSize: 25, TimeoutS: "10s", Timeout: 10 * time.Second}, UI: UI{CompactWidth: 76, RecommendedWidth: 120}, Shell: Shell{Candidates: []string{"/bin/bash", "/bin/sh", "bash", "sh", "ash"}, EscapeKey: "ctrl+]"}, Timeouts: Timeouts{Connect: "8s", Operation: "35s", Stream: "30m"}}
}
func Path() string { d, _ := os.UserConfigDir(); return filepath.Join(d, "docktop", "config.yaml") }
func Load(path string) (Config, error) {
	c := Default()
	if path == "" {
		path = Path()
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return applyEnvironment(c), nil
	}
	if err != nil {
		return c, err
	}
	if err = yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.Refresh == "" {
		c.Refresh = "3s"
	}
	c.RefreshInterval, err = time.ParseDuration(c.Refresh)
	if err != nil || c.RefreshInterval < time.Second {
		return c, errors.New("refresh_interval deve ser uma duração >= 1s")
	}
	if _, ok := c.Contexts[c.DefaultContext]; !ok {
		return c, errors.New("default_context não existe em contexts")
	}
	if c.Registry.PageSize < 1 || c.Registry.PageSize > 100 {
		return c, errors.New("registry.page_size deve estar entre 1 e 100")
	}
	if c.Registry.TimeoutS == "" {
		c.Registry.TimeoutS = "10s"
	}
	c.Registry.Timeout, err = time.ParseDuration(c.Registry.TimeoutS)
	if err != nil || c.Registry.Timeout <= 0 {
		return c, errors.New("registry.timeout inválido")
	}
	for name, dc := range c.Contexts {
		if err := validateContext(name, dc); err != nil {
			return c, err
		}
	}
	return applyEnvironment(c), nil
}

// applyEnvironment follows Docker's standard precedence when the default local
// context was not explicitly replaced. Named DockTop contexts remain authoritative.
func applyEnvironment(c Config) Config {
	if name := strings.TrimSpace(os.Getenv("DOCKER_CONTEXT")); name != "" {
		if _, ok := c.Contexts[name]; ok {
			c.DefaultContext = name
		}
	}
	if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); host != "" && c.DefaultContext == "local" {
		dc := c.Contexts["local"]
		dc.Host = host
		if os.Getenv("DOCKER_TLS_VERIFY") != "" {
			dc.TLS.Enabled = true
		}
		c.Contexts["local"] = dc
	}
	return c
}

func validateContext(name string, c Context) error {
	if c.Host == "" {
		return fmt.Errorf("contexto %q: host vazio", name)
	}
	if !strings.HasPrefix(c.Host, "unix://") && !strings.HasPrefix(c.Host, "tcp://") && !strings.HasPrefix(c.Host, "http://") && !strings.HasPrefix(c.Host, "https://") {
		return fmt.Errorf("contexto %q: endpoint Docker inválido", name)
	}
	if c.TLS.Enabled && (c.TLS.CAFile == "" || c.TLS.CertFile == "" || c.TLS.KeyFile == "") {
		return fmt.Errorf("contexto %q: TLS exige ca_file, cert_file e key_file", name)
	}
	return nil
}

func Save(path string, c Config) error {
	if path == "" {
		path = Path()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
