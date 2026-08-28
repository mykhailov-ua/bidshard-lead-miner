package serp

import (
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/sources/forum"
)

func TestSerpHarvestOpenWebDorksSkipsSiteOperator(t *testing.T) {
	icp := []string{
		"site:affiliatefix.com voluum",
		`"postback failing" tracker`,
		`"voluum alternative" affiliate`,
	}
	got := serpHarvestOpenWebDorks(icp)
	if len(got) != 2 {
		t.Fatalf("dorks=%v want 2 open-web entries", got)
	}
	for _, d := range got {
		if strings.Contains(strings.ToLower(d), "site:") {
			t.Fatalf("site dork leaked into open web list: %q", d)
		}
	}
}

func TestExtractWebPainDiscoveries(t *testing.T) {
	forum.ConfigureHosts(nil)

	results := []SERPResult{
		{URL: "https://blog-affiliate.example/postback-failing-tracker", Title: "Pain post", Snippet: "voluum alternative"},
		{URL: "https://www.affiliatefix.com/threads/voluum.1001/", Title: "Forum", Snippet: "skip"},
		{URL: "https://www.reddit.com/r/affiliatemarketing/comments/abc", Title: "Reddit", Snippet: "skip"},
		{URL: "https://example.com/blog/not-a-forum", Title: "Blog", Snippet: "tracker pain"},
	}
	got := ExtractWebPainDiscoveries(results)
	if len(got) != 2 {
		t.Fatalf("discoveries=%v want 2", got)
	}
}
