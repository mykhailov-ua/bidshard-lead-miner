package serp

import "testing"

func TestIsTGCatalogCandidateURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url  string
		want bool
	}{
		{"https://tgstat.com/ru/channel/@affhub", true},
		{"https://telemetr.io/en/channels", true},
		{"https://afflift.com/f/threads/telegram-groups.123/", true},
		{"https://t.me/voluum", false},
		{"https://example.com/about", false},
		{"https://affiliatefix.com/threads/best-telegram-channels.999/", true},
	}
	for _, tc := range cases {
		if got := isTGCatalogCandidateURL(tc.url); got != tc.want {
			t.Errorf("isTGCatalogCandidateURL(%q)=%v want %v", tc.url, got, tc.want)
		}
	}
}

func TestAppendTGCatalogPageDiscoveriesDedup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/catalog_pages.json"
	n1, err := appendTGCatalogPageDiscoveries(path, "dork1", []string{"https://tgstat.com/foo"})
	if err != nil || n1 != 1 {
		t.Fatalf("first append n=%d err=%v", n1, err)
	}
	n2, err := appendTGCatalogPageDiscoveries(path, "dork1", []string{"https://tgstat.com/foo"})
	if err != nil || n2 != 0 {
		t.Fatalf("dedup append n=%d err=%v", n2, err)
	}
}
