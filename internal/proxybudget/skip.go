package proxybudget

import "github.com/bidshard/parser/internal/metrics"

// ShouldSkipProxySource reports whether proxy-routed crawl should be skipped (budget exceeded).
func ShouldSkipProxySource(sourceID string, usesProxy bool) (bool, string) {
	if !usesProxy {
		return false, ""
	}
	g := Current()
	if g == nil || !g.Enabled() {
		return false, ""
	}
	if g.Allow() {
		return false, ""
	}
	metrics.RecordProxyBudgetSkipped(sourceID)
	return true, "proxy_daily_budget_exceeded"
}
