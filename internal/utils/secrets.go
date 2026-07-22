package utils

import (
	"regexp"
	"strings"
)

var sensitiveName = regexp.MustCompile(`(?i)(password|passwd|token|secret|credential|api[_-]?key|private[_-]?key)`)

// MaskSecret masks values whose key is commonly used for credentials. It is
// shared by inspect views, errors and audit so redaction cannot drift by screen.
func MaskSecret(key, value string) string {
	if sensitiveName.MatchString(key) {
		return "[REDACTED]"
	}
	return value
}

func MaskEnvironment(env []string) []string {
	out := make([]string, len(env))
	for i, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			out[i] = item
			continue
		}
		out[i] = key + "=" + MaskSecret(key, value)
	}
	return out
}

func Sanitize(text string) string {
	parts := strings.Fields(text)
	for i, part := range parts {
		key, _, ok := strings.Cut(part, "=")
		if ok && sensitiveName.MatchString(strings.TrimLeft(key, "-")) {
			parts[i] = key + "=[REDACTED]"
		}
	}
	return strings.Join(parts, " ")
}
