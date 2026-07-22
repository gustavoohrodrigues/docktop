package docker

import "testing"

func TestParseCreateSpec(t *testing.T) {
	r, err := ParseCreateSpec("web | nginx:alpine | 8080:80,8443:443 | /srv/html:/usr/share/nginx/html:ro | APP_ENV=prod | unless-stopped | nginx -g daemon_off")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "web" || r.Image != "nginx:alpine" || len(r.Ports) != 2 || len(r.Volumes) != 1 || r.Restart != "unless-stopped" {
		t.Fatalf("resultado: %#v", r)
	}
}
func TestParseCreateSpecRequiresIdentity(t *testing.T) {
	if _, err := ParseCreateSpec(" | nginx"); err == nil {
		t.Fatal("esperava erro")
	}
}
