package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "c.yaml")
	if err := os.WriteFile(p, []byte("default_context: test\ncontexts:\n  test:\n    host: unix:///tmp/docker.sock\nrefresh_interval: 2s\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, e := Load(p)
	if e != nil {
		t.Fatal(e)
	}
	if c.RefreshInterval != 2*time.Second || c.Contexts["test"].Host != "unix:///tmp/docker.sock" {
		t.Fatalf("config inesperada: %#v", c)
	}
}
func TestRejectFastRefresh(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	_ = os.WriteFile(p, []byte("default_context: x\ncontexts: {x: {host: unix:///x}}\nrefresh_interval: 10ms\n"), 0600)
	if _, e := Load(p); e == nil {
		t.Fatal("esperava erro")
	}
}

func TestLoadLanguage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "language.yaml")
	data := "default_context: local\ncontexts: {local: {host: unix:///var/run/docker.sock}}\nrefresh_interval: 3s\nlanguage: en-US\n"
	if err := os.WriteFile(p, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != "en-US" {
		t.Fatalf("idioma não carregado: %q", c.Language)
	}
}
