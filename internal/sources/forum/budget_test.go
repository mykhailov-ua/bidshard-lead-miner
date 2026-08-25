package forum

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/proxybudget"
)

func TestAdapterCollectSkipsWhenProxyBudgetExceeded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proxy_budget.json")
	g := proxybudget.Configure(1, path)
	g.Record(1024 * 1024)

	cfg := config.Config{
		ForumSeedPath: filepath.Join("data", "seeds", "forum_threads.csv"),
		HTTPWorkers:   1,
		ProxyURLs:     []string{"http://proxy:8080"},
		ProxySources:  []string{"forum"},
	}
	a := NewAdapter(cfg, NewFetcher(0, ""))
	err := a.Collect(context.Background(), func(context.Context, model.RawItem) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}
