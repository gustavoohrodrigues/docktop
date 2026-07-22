package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Result struct {
	Name, Description   string
	Stars, Pulls        int64
	Official, Automated bool
}
type Hub struct {
	client  *http.Client
	baseURL string
}

func NewHub() *Hub {
	return &Hub{client: &http.Client{Timeout: 8 * time.Second}, baseURL: "https://hub.docker.com"}
}
func (h *Hub) Search(ctx context.Context, query string) ([]Result, error) {
	if len(query) < 2 {
		return nil, errors.New("digite ao menos 2 caracteres")
	}
	u := h.baseURL + "/v2/search/repositories/?page_size=25&query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Docker Hub indisponível: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("limite de requisições do Docker Hub atingido")
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub respondeu HTTP %d", res.StatusCode)
	}
	var payload struct {
		Results []struct {
			RepoName         string `json:"repo_name"`
			ShortDescription string `json:"short_description"`
			StarCount        int64  `json:"star_count"`
			PullCount        int64  `json:"pull_count"`
			IsOfficial       bool   `json:"is_official"`
			IsAutomated      bool   `json:"is_automated"`
		} `json:"results"`
	}
	if err = json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(payload.Results))
	for _, x := range payload.Results {
		if x.RepoName == "" {
			continue
		}
		out = append(out, Result{x.RepoName, x.ShortDescription, x.StarCount, x.PullCount, x.IsOfficial, x.IsAutomated})
	}
	return out, nil
}
