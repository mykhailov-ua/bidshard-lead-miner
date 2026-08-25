package serp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	<a class="result__a" href="https://example.com/blog/not-a-forum">Ignore me</a>
	<a class="result__snippet">noise</a>
</div>
`

func TestExtractForumThreadURLsFromSERPHTML(t *testing.T) {
	results := parseSERPResults(forumSERPFixture)
	urls := ExtractForumThreadURLs(results)
	if len(urls) != 3 {
		t.Fatalf("urls=%v want 3 thread URLs", urls)
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
	if added != 3 {
		t.Fatalf("added=%d want 3", added)
	}

	reg, err := forum.LoadThreadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Threads) != 3 {
		t.Fatalf("threads=%d want 3", len(reg.Threads))
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
		`"title": "Voluum alternative thread"`,
		`"snippet": "Looking for tracker migration"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("registry missing %q:\n%s", want, body)
		}
	}
}
