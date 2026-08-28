package serp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/forum"
)

const forumSERPFixture = `
<div class="result">
	<a class="result__a" href="https://www.affiliatefix.com/threads/voluum-alternative-thread.1001/">Voluum alternative thread</a>
	<a class="result__snippet">Looking for tracker migration</a>
</div>
<div class="result">
	<a class="result__a" href="https://stmforum.com/threads/keitaro-too-expensive.2002/">Keitaro pricing pain</a>
	<a class="result__snippet">Self-hosted tracker discussion</a>
</div>
<div class="result">
	<a class="result__a" href="https://blackhatworld.com/seo/postback-failing-tracker.3003/">Postback failing</a>
	<a class="result__snippet">S2S postback debug</a>
</div>
<div class="result">
	<a class="result__a" href="https://forums.gpwa.org/threads/keitaro-migration.4004/">Keitaro migration</a>
	<a class="result__snippet">Tracker switch discussion</a>
</div>
<div class="result">
	<a class="result__a" href="https://example.com/blog/not-a-forum">Ignore me</a>
	<a class="result__snippet">noise</a>
</div>
`

func TestExtractForumThreadURLsFromSERPHTML(t *testing.T) {
	results := parseSERPResults(forumSERPFixture)
	urls := ExtractForumThreadURLs(results)
	if len(urls) != 4 {
		t.Fatalf("urls=%v want 4 thread URLs", urls)
	}
}

func TestAppendForumThreadDiscoveriesRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "discovered_forum_threads.json")

	results := parseSERPResults(forumSERPFixture)
	items := ExtractForumThreadDiscoveries(results)
	added, err := forum.AppendThreadDiscoveries(path, "serp", `site:affiliatefix.com "voluum"`, items)
	if err != nil {
		t.Fatal(err)
	}
	if added != 4 {
		t.Fatalf("added=%d want 4", added)
	}

	reg, err := forum.LoadThreadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Threads) != 4 {
		t.Fatalf("threads=%d want 4", len(reg.Threads))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{
		"affiliatefix.com/threads/voluum-alternative-thread.1001",
		"stmforum.com/threads/keitaro-too-expensive.2002",
		"blackhatworld.com/seo/postback-failing-tracker.3003",
		"forums.gpwa.org/threads/keitaro-migration.4004",
		`"title": "Voluum alternative thread"`,
		`"snippet": "Looking for tracker migration"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("registry missing %q:\n%s", want, body)
		}
	}
}

func TestSerpHarvestDorksFromICPIncludesAllHosts(t *testing.T) {
	icp := []string{
		"site:affiliatefix.com voluum",
		"site:gpwa.org tracker",
		"site:reddit.com/r/affiliatemarketing postback",
		"site:affiliatefix.com voluum",
	}
	got := serpHarvestDorksFromICP(icp)
	if len(got) != 3 {
		t.Fatalf("dorks=%v want 3 deduped entries", got)
	}
	for _, want := range []string{
		"site:gpwa.org tracker",
		"site:reddit.com/r/affiliatemarketing postback",
	} {
		found := false
		for _, d := range got {
			if d == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing dork %q in %v", want, got)
		}
	}
}

func TestHarvestForumThreadsFromICPDorks(t *testing.T) {
	forum.ConfigureHosts(nil)

	dir := t.TempDir()
	icpDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(icpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	icpRaw := `{
  "telegram_search": [],
  "serp_dorks": [
    "site:gpwa.org tracker",
    "site:example.com not-forum"
  ]
}`
	if err := os.WriteFile(filepath.Join(icpDir, "discover.icp.json"), []byte(icpRaw), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(forumSERPFixture))
	}))
	defer ts.Close()

	cfg := config.Config{}
	crawler := NewCrawler(cfg, ts.Client())
	crawler.SetBaseURL(ts.URL)

	regPath := filepath.Join(dir, "discovered_forum_threads.json")
	if err := crawler.HarvestForumThreads(context.Background(), regPath); err != nil {
		t.Fatal(err)
	}

	reg, err := forum.LoadThreadRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Threads) != 4 {
		t.Fatalf("threads=%d want 4 from allowlisted forum hosts", len(reg.Threads))
	}
	for _, thread := range reg.Threads {
		if strings.Contains(thread.URL, "forums.gpwa.org") {
			return
		}
	}
	t.Fatalf("registry missing GPWA thread: %v", reg.Threads)
}
