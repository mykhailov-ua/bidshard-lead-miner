package forum

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestAdapterEmitsPainPost(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("../../../testdata/forum/stm_thread.html")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	seed := writeForumSeed(t, "https://stm-forum.example/threads/voluum-alternative.123/")
	cfg := config.Config{
		ForumSeedPath: seed,
		HTTPTimeout:   5 * time.Second,
		ForumBaseURL:  server.URL,
	}

	adapter := NewAdapter(cfg, NewFetcher(cfg.HTTPTimeout, server.URL))
	var items []model.RawItem
	err = adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].Contact == "" {
		t.Fatal("expected contact")
	}
}

func writeForumSeed(t *testing.T, url string) string {
	t.Helper()
	path := t.TempDir() + "/forum.csv"
	if err := os.WriteFile(path, []byte("url\n"+url+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
