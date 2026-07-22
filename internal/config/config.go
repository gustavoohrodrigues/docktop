package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type TLS struct {
	Enabled                   bool `yaml:"enabled"`
	CAFile, CertFile, KeyFile string
}
type Context struct {
	Host string `yaml:"host"`
	TLS  TLS    `yaml:"tls"`
}
type Audit struct {
	Enabled              bool `yaml:"enabled"`
	MaxSizeMB, Retention int
}
type Dangerous struct{ RemoveContainers, RemoveImages, RemoveVolumes, RemoveNetworks, Prune, SwarmChanges bool }
type Config struct {
	DefaultContext   string             `yaml:"default_context"`
	Contexts         map[string]Context `yaml:"contexts"`
	RefreshInterval  time.Duration      `yaml:"-"`
	Refresh          string             `yaml:"refresh_interval"`
	ReadOnly         bool               `yaml:"read_only"`
	Theme            string             `yaml:"theme"`
	MouseEnabled     bool               `yaml:"mouse_enabled"`
	TelemetryEnabled bool               `yaml:"telemetry_enabled"`
	Audit            Audit              `yaml:"audit"`
	DangerousActions Dangerous          `yaml:"dangerous_actions"`
}

func Default() Config {
	return Config{DefaultContext: "local", Contexts: map[string]Context{"local": {Host: "unix:///var/run/docker.sock"}}, Refresh: "3s", RefreshInterval: 3 * time.Second, Theme: "dark-ops", MouseEnabled: true, Audit: Audit{Enabled: true, MaxSizeMB: 10, Retention: 5}, DangerousActions: Dangerous{true, true, true, true, true, true}}
}
func Path() string { d, _ := os.UserConfigDir(); return filepath.Join(d, "docktop", "config.yaml") }
func Load(path string) (Config, error) {
	c := Default()
	if path == "" {
		path = Path()
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
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
	return c, nil
}
