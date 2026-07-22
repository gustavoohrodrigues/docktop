package theme

import "testing"

func TestFallback(t *testing.T) {
	if Get("unknown").Name != "dark-ops" {
		t.Fatal("fallback inválido")
	}
}
