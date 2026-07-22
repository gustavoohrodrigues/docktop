package ui

import "testing"

func TestAppendLimit(t *testing.T) {
	v := []float64{}
	for i := 0; i < 100; i++ {
		v = appendLimit(v, float64(i), 10)
	}
	if len(v) != 10 || v[0] != 90 || v[9] != 99 {
		t.Fatalf("janela incorreta: %v", v)
	}
}
func TestSanitize(t *testing.T) {
	got := sanitize("request password=secret")
	if got == "request password=secret" {
		t.Fatal("credencial não sanitizada")
	}
}
func TestShortID(t *testing.T) {
	if got := shortID("sha256:1234567890abcdef"); got != "1234567890ab" {
		t.Fatalf("id: %s", got)
	}
}
