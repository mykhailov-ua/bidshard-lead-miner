package forum

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	if len(items) != 2 {
		t.Fatalf("items=%d want 2 (pain post + author-only post; processor filters noise)", len(items))
	}
	var painPost bool
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Raw), "voluum alternative") {
			painPost = true
			if item.Contact == "" {
				t.Fatal("expected contact on pain post")
			}
		}
	}
	if !painPost {
		t.Fatalf("items=%v", items)
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

func TestAdapterEmitsRegistryKeywordPainOnly(t *testing.T) {
	t.Parallel()

	// "redtrack alternative" is in testdata/keywords.json but was not in legacy HasPainSignal list.
	html := `<html><body>
<article class="post">
  <div class="username">buyer_red</div>
  <div class="postbody">
    <p>Need redtrack alternative before renewal hits. Email ops@igaming-team.com</p>
  </div>
</article>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	seed := writeForumSeed(t, "https://affiliatefix.example/threads/redtrack-switch.999/")
	cfg := config.Config{
		ForumSeedPath: seed,
		HTTPTimeout:   5 * time.Second,
		ForumBaseURL:  server.URL,
	}

	adapter := NewAdapter(cfg, NewFetcher(cfg.HTTPTimeout, server.URL))
	var items []model.RawItem
	if err := adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1 (crawl must not pre-drop registry-only pain)", len(items))
	}
	if !strings.Contains(items[0].Raw, "redtrack alternative") {
		t.Fatalf("raw=%q", items[0].Raw)
	}
}

func TestAdapterEmitsWarriorForumThread(t *testing.T) {
	t.Parallel()

	htmlFixture := `<html><body>
	<time datetime="2025-06-01T12:00:00Z"></time>
	<div class="post-container">
		<a class="username">BuyerJohn</a>
		<div class="post-content">Looking for voluum alternative, contact me telegram:@buyerjohn</div>
	</div>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(htmlFixture))
	}))
	defer server.Close()

	warriorSeed := writeForumSeed(t, server.URL+"/threads/voluum-alternative-post.1")
	cfg := config.Config{
		WarriorSeedPath: warriorSeed,
		ForumSeedPath:   t.TempDir() + "/missing_forum.csv",
		HTTPTimeout:     5 * time.Second,
		ForumBaseURL:    server.URL,
	}

	adapter := NewAdapter(cfg, NewFetcher(cfg.HTTPTimeout, server.URL))
	var items []model.RawItem
	if err := adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].ContactTelegram() != "telegram:@buyerjohn" {
		t.Fatalf("contact=%q", items[0].Contact)
	}
	if !strings.Contains(items[0].Source, "voluum-alternative-post.1") {
		t.Fatalf("source=%q", items[0].Source)
	}
}
