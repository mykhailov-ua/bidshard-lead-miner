package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestGitHubSearchResponse(t *testing.T) {
	fixture := `{
		"items": [
			{
				"html_url": "https://github.com/bidshard/parser/issues/42",
				"title": "Need voluum alternative for tracking",
				"body": "We are migrating to self-hosted tracker, contact telegram:@gh_user",
				"user": {"login": "gh_user"}
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fixture))
	}))
	defer ts.Close()

	cfg := config.Config{
		GitHubSearchQueries: []string{"voluum alternative"},
	}

	crawler := NewCrawler(cfg)
	crawler.SetBaseURL(ts.URL)
	crawler.SetHTTPClient(ts.Client())

	var emitted []model.RawItem
	err := crawler.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		emitted = append(emitted, item)
		return nil
	})

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(emitted) == 0 {
		t.Fatalf("expected emitted items, got 0")
	}
	if emitted[0].ContactTelegram() != "telegram:@gh_user" {
		t.Errorf("got contact %q, want telegram:@gh_user", emitted[0].ContactTelegram())
	}
}
