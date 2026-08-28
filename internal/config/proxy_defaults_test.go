package config

import "testing"

func TestApplyProxyDefaultsScopesWhenUnset(t *testing.T) {
	cfg := Config{
		ProxyURLs: []string{"http://user:pass@proxy:8080"},
	}
	applyProxyDefaults(&cfg)
	if len(cfg.ProxySources) != len(defaultProxySources) {
		t.Fatalf("sources=%v", cfg.ProxySources)
	}
	if !cfg.ProxyEnabledForSource("serp") {
		// ok
	} else {
		t.Fatal("serp should not use proxy with default scope")
	}
	if !cfg.ProxyEnabledForSource("forum") {
		t.Fatal("forum should use proxy with default scope")
	}
}

func TestApplyProxyDefaultsPreservesExplicitSources(t *testing.T) {
	cfg := Config{
		ProxyURLs:    []string{"http://user:pass@proxy:8080"},
		ProxySources: []string{"serp"},
	}
	applyProxyDefaults(&cfg)
	if len(cfg.ProxySources) != 1 || cfg.ProxySources[0] != "serp" {
		t.Fatalf("sources=%v", cfg.ProxySources)
	}
}
