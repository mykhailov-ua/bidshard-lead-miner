package config

import "testing"

func TestAutoDiscoverRegistryOKForum(t *testing.T) {
	cfg := Config{ParserAutoDiscover: true}
	ok, msg := AutoDiscoverRegistryOK(cfg, "forum", 5)
	if !ok || msg == "" {
		t.Fatalf("ok=%v msg=%q", ok, msg)
	}
}

func TestAutoDiscoverRegistryOKDisabled(t *testing.T) {
	cfg := Config{ParserAutoDiscover: false}
	if ok, _ := AutoDiscoverRegistryOK(cfg, "forum", 5); ok {
		t.Fatal("expected false when auto-discover disabled")
	}
}

func TestAutoDiscoverRegistryOKEmptyRegistry(t *testing.T) {
	cfg := Config{ParserAutoDiscover: true}
	if ok, _ := AutoDiscoverRegistryOK(cfg, "forum", 0); ok {
		t.Fatal("expected false for empty registry")
	}
}
