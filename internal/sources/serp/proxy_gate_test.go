package serp

import (
	"context"
	"errors"
	"testing"

	"github.com/bidshard/parser/internal/config"
)

func TestHarvestProxyReadyDirectByDefault(t *testing.T) {
	cfg := config.Config{
		ProxyURLs:    []string{"http://user:pass@proxy:8080"},
		ProxySources: config.DefaultProxySources(),
	}
	if !HarvestProxyReady(cfg) {
		t.Fatal("serp should use direct egress with default proxy scope")
	}
}

func TestHarvestProxyReadyRequiresListWhenScoped(t *testing.T) {
	cfg := config.Config{
		ProxySources: []string{"serp"},
	}
	if HarvestProxyReady(cfg) {
		t.Fatal("expected skip when serp scoped to proxy but list empty")
	}
	cfg.ProxyURLs = []string{"http://user:pass@proxy:8080"}
	if !HarvestProxyReady(cfg) {
		t.Fatal("expected ready when serp scoped and proxy list set")
	}
}

func TestRunBGHarvestSkipsWhenNoProxy(t *testing.T) {
	cfg := config.Config{
		ProxySources: []string{"serp"},
	}
	called := false
	err := RunBGHarvest(context.Background(), cfg, "serp_telegram_catalog", func(context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("expected harvest fn to be skipped")
	}
}

func TestRunBGHarvestRunsWhenReady(t *testing.T) {
	sentinel := errors.New("ran")
	err := RunBGHarvest(context.Background(), config.Config{}, "serp_forum_threads", func(context.Context) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}
}
