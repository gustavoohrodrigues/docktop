package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFallback(t *testing.T) {
	if Get("unknown").Name != "dark-ops" {
		t.Fatal("fallback inválido")
	}
}

func TestLoadFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "custom.yaml")
	b := []byte("name: custom\nbackground: '#000000'\npanel: '#111111'\nborder: '#222222'\nborder_focus: '#ffffff'\ntext: '#eeeeee'\nprimary: '#00ffff'\ndanger: '#ff0000'\n")
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "custom" {
		t.Fatal(got.Name)
	}
}
