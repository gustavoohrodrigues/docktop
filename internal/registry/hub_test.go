package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchRejectsShortQuery(t *testing.T) {
	if _, err := NewHub().Search(context.Background(), "x"); err == nil {
		t.Fatal("esperava validação local")
	}
}

func TestSearchMapsDockerHubFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"repo_name":"library/nginx","short_description":"web server","star_count":42,"pull_count":9000,"is_official":true}]}`))
	}))
	defer srv.Close()
	h := NewHub()
	h.baseURL = srv.URL
	got, err := h.Search(context.Background(), "nginx")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "library/nginx" || got[0].Stars != 42 || !got[0].Official {
		t.Fatalf("mapeamento: %#v", got)
	}
}
