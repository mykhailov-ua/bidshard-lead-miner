package forum

import (
	"context"
	"errors"
	"testing"

	"github.com/bidshard/parser/internal/config"
)

func TestCrawlProxyReadyDirectWhenForumNotScoped(t *testing.T) {
	cfg := config.Config{
		ProxySources: []string{"tgweb"},
	}
	if !CrawlProxyReady(cfg) {
		t.Fatal("forum should allow direct egress when not in proxy scope")
	}
}

func TestCrawlProxyReadyRequiresListWhenScoped(t *testing.T) {
	cfg := config.Config{
		ProxySources: []string{"forum"},
	}
	if CrawlProxyReady(cfg) {
		t.Fatal("expected skip when forum scoped to proxy but list empty")
	}
	cfg.ProxyURLs = []string{"http://user:pass@proxy:8080"}
	if !CrawlProxyReady(cfg) {
		t.Fatal("expected ready when forum scoped and proxy list set")
	}
}

func TestRunBGCrawlSkipsWhenNoProxy(t *testing.T) {
	cfg := config.Config{
		ProxySources: []string{"forum"},
	}
	called := false
	err := RunBGCrawl(context.Background(), cfg, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("expected crawl fn to be skipped")
	}
}

func TestRunBGCrawlRunsWhenReady(t *testing.T) {
	sentinel := errors.New("ran")
	err := RunBGCrawl(context.Background(), config.Config{}, func(context.Context) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}
}
