package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSanitizesAndProtectsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	l, err := NewWithOptions(true, path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = l.Write(Entry{Action: "login", Error: "token=abc"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "abc") || !strings.Contains(string(b), "[REDACTED]") {
		t.Fatalf("credencial não sanitizada: %s", b)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("modo inseguro: %o", info.Mode().Perm())
	}
}
