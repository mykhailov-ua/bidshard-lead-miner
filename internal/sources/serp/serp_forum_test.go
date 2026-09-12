package serp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/forum"
)

func TestSERPCollectQueuesForumThreads(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "discovered_forum_threads.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
<div class="result">
	<a class="result__a" href="https://blackhatworld.com/seo/voluum-alternative.123/">Cheap VOLUUM alternative</a>
	<a class="result__snippet">I can't afford voluum, looking for cheap tracker</a>
</div>
`))
	}))
	defer ts.Close()

	cfg := config.Config{}
	crawler := NewCrawler(cfg, ts.Client())
	crawler.SetBaseURL(ts.URL)

	var emitted []model.RawItem
	emit := func(ctx context.Context, item model.RawItem) error {
		emitted = append(emitted, item)
		return nil
	}

	if err := crawler.Collect(context.Background(), emit); err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(emitted) != 0 {
		t.Fatalf("expected no hot-path emit for forum SERP, got %d", len(emitted))
	}

	// Registry write uses default path; override by calling harvest helper directly.
	added, err := forum.AppendThreadDiscoveries(registryPath, "test", "dork", ExtractForumThreadDiscoveries([]SERPResult{{
		URL:     "https://blackhatworld.com/seo/voluum-alternative.123/",
		Title:   "Cheap VOLUUM alternative",
		Snippet: "I can't afford voluum",
		Domain:  "blackhatworld.com",
	}}))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
	if _, err := os.Stat(registryPath); err != nil {
		t.Fatalf("registry missing: %v", err)
	}
}
