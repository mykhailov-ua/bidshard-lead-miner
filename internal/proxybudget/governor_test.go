package proxybudget

import (
	"path/filepath"
	"testing"
)

func TestGovernorCapBlocksAfterLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proxy_budget.json")
	g := Configure(1, path)
	if !g.Allow() {
		t.Fatal("expected allow before usage")
	}
	g.Record(1024 * 1024)
	if g.Allow() {
		t.Fatal("expected block at 1MB cap")
	}
	if g.UsedBytes() != 1024*1024 {
		t.Fatalf("used=%d", g.UsedBytes())
	}
}

func TestGovernorDisabledAllowsAlways(t *testing.T) {
	g := Configure(0, "")
	if !g.Allow() {
		t.Fatal("disabled governor should allow")
	}
	g.Record(1 << 30)
	if !g.Allow() {
		t.Fatal("disabled governor should still allow")
	}
}

func TestShouldSkipProxySourceOnlyWhenProxyRouted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proxy_budget.json")
	g := Configure(1, path)
	g.Record(1024 * 1024)

	skip, reason := ShouldSkipProxySource("forum", false)
	if skip || reason != "" {
		t.Fatalf("non-proxy source skip=%v reason=%q", skip, reason)
	}
	skip, reason = ShouldSkipProxySource("forum", true)
	if !skip || reason != "proxy_daily_budget_exceeded" {
		t.Fatalf("skip=%v reason=%q", skip, reason)
	}
}
