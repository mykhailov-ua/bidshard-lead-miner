package warrior

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestWarriorCrawler(t *testing.T) {
	htmlFixture := `<html><body>
	<time datetime="2025-06-01T12:00:00Z"></time>
	<div class="post-container">
		<a class="username">BuyerJohn</a>
		<div class="post-content">Looking for voluum alternative, contact me telegram:@buyerjohn</div>
	</div>
	</body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlFixture))
	}))
	defer ts.Close()

	seedPath := filepath.Join(t.TempDir(), "warrior.csv")
	seed := "url\n" + ts.URL + "/threads/voluum-vs-keitaro.1\n"
	if err := os.WriteFile(seedPath, []byte(seed), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		WarriorSeedPath: seedPath,
		ForumBaseURL:    ts.URL,
	}

	crawler := NewCrawler(cfg, ts.Client())
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
	if emitted[0].ContactTelegram() != "telegram:@buyerjohn" {
		t.Errorf("got contact %q, want telegram:@buyerjohn", emitted[0].ContactTelegram())
	}
	if emitted[0].Source != "warrior:voluum-vs-keitaro.1" {
		t.Errorf("source=%q", emitted[0].Source)
	}
	want := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	if !emitted[0].PostedAt.Equal(want) {
		t.Errorf("posted_at=%v want %v", emitted[0].PostedAt, want)
	}
}

func TestParsePostsDate(t *testing.T) {
	html := `<time datetime="2024-03-15T08:30:00Z"></time>
	<a class="username">buyer</a>
	<div class="post-content">voluum alternative postback failing</div>`
	posts := parsePosts(html)
	if len(posts) != 1 {
		t.Fatalf("posts=%d", len(posts))
	}
	want := time.Date(2024, 3, 15, 8, 30, 0, 0, time.UTC)
	if !posts[0].PostedAt.Equal(want) {
		t.Errorf("posted_at=%v want %v", posts[0].PostedAt, want)
	}
}
