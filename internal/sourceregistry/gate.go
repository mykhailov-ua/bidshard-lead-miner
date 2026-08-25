package sourceregistry

import (
	"strings"
)

// ShouldSkipCrawl reports whether HTTP crawl should be skipped for domain before fetch.
// Heuristic drops always apply. Registry triage applies when triageEnabled and domain is in registry.
func ShouldSkipCrawl(registryPath string, triageEnabled bool, meta DomainMeta) (skip bool, reason string) {
	if action, why, ok := HeuristicTriage(meta); ok && action == "drop" {
		return true, "heuristic:" + why
	}
	if !triageEnabled {
		return false, ""
	}
	entry, ok := LookupEntry(registryPath, meta.Domain)
	if !ok {
		// Not in registry yet (e.g. CSV-only supply seed): heuristic gate only.
		return false, ""
	}
	switch strings.ToLower(strings.TrimSpace(entry.TriageStatus)) {
	case "keep":
		return false, ""
	case "drop":
		return true, "triage:drop"
	default:
		return true, "triage:pending"
	}
}
