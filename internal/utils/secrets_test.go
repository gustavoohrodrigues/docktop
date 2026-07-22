package utils

import "testing"

func TestMaskEnvironment(t *testing.T) {
	got := MaskEnvironment([]string{"USER=alice", "DB_PASSWORD=hunter2", "API_KEY=abc"})
	if got[0] != "USER=alice" || got[1] != "DB_PASSWORD=[REDACTED]" || got[2] != "API_KEY=[REDACTED]" {
		t.Fatalf("redação incorreta: %#v", got)
	}
}

func TestSanitize(t *testing.T) {
	got := Sanitize("request token=abc endpoint=x")
	if got != "request token=[REDACTED] endpoint=x" {
		t.Fatalf("sanitize: %q", got)
	}
}
