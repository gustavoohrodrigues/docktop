package docker

import (
	"errors"
	"strings"
)

// ParseCreateSpec converte o formulário compacto da TUI em argumentos estruturados.
// Formato: nome | imagem | portas | volumes | ambiente | restart | comando
func ParseCreateSpec(input string) (CreateRequest, error) {
	parts := strings.Split(input, "|")
	for len(parts) < 7 {
		parts = append(parts, "")
	}
	if len(parts) > 7 {
		return CreateRequest{}, errors.New("use exatamente 7 campos separados por |; o comando não pode conter |")
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if parts[0] == "" || parts[1] == "" {
		return CreateRequest{}, errors.New("nome e imagem são obrigatórios")
	}
	req := CreateRequest{Name: parts[0], Image: parts[1], Restart: parts[5], Command: parts[6]}
	req.Ports = splitCSV(parts[2])
	req.Volumes = splitCSV(parts[3])
	req.Env = splitCSV(parts[4])
	return req, nil
}
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
