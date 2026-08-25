package config

import "testing"

func TestProxyURLsForSourceLegacyGlobal(t *testing.T) {
	cfg := Config{
		ProxyURLs: []string{"http://proxy:8080"},
	}
	if got := cfg.ProxyURLsForSource("reddit"); len(got) != 1 {
		t.Fatalf("reddit proxies=%v want global list", got)
	}
	if got := cfg.ProxyURLsForSource("forum"); len(got) != 1 {
		t.Fatalf("forum proxies=%v want global list", got)
	}
}

func TestProxyURLsForSourceScoped(t *testing.T) {
	cfg := Config{
		ProxyURLs:    []string{"http://proxy:8080"},
		ProxySources: []string{"forum", "tgweb", "lander"},
	}
	if got := cfg.ProxyURLsForSource("reddit"); len(got) != 0 {
		t.Fatalf("reddit proxies=%v want direct", got)
	}
	if got := cfg.ProxyURLsForSource("forum"); len(got) != 1 {
		t.Fatalf("forum proxies=%v want proxy", got)
	}
	if got := cfg.ProxyURLsForSource("TGWEB"); len(got) != 1 {
		t.Fatalf("tgweb proxies=%v want proxy (case insensitive)", got)
	}
	if got := cfg.ProxyURLsForSource("supply"); len(got) != 0 {
		t.Fatalf("supply proxies=%v want direct", got)
	}
}

func TestProxyURLsForSourceEmptyList(t *testing.T) {
	cfg := Config{
		ProxySources: []string{"forum"},
	}
	if got := cfg.ProxyURLsForSource("forum"); got != nil {
		t.Fatalf("forum proxies=%v want nil when list empty", got)
	}
}
