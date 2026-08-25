package sourceregistry

import (
	"strings"
	"time"
)

// RelevanceScore is a 0-100 heuristic from registry metadata (no Gemini).
// Weights: base 40, keep +35, multi-type +10, accepted-lead seed +15, telegram +5.
func RelevanceScore(e Entry) int {
	score := 40
	switch strings.ToLower(strings.TrimSpace(e.TriageStatus)) {
	case "keep":
		score += 35
	case "defer", "":
		score += 10
	case "drop":
		return 5
	}
	if len(e.Types) >= 2 {
		score += 10
	}
	if strings.Contains(strings.ToLower(e.DiscoveredBy), "accepted") {
		score += 15
	}
	if strings.Contains(strings.ToLower(e.DiscoveredBy), "telegram") {
		score += 5
	}
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

type ScoreSummary struct {
	Domain         string
	Types          []string
	TriageStatus   string
	RelevanceScore int
	DiscoveredBy   string
	LastSeen       string
	LastTriageAt   string
	TriageWhy      string
}

func Summarize(f File) []ScoreSummary {
	out := make([]ScoreSummary, 0, len(f.Sources))
	for _, e := range f.Sources {
		domain := NormalizeDomain(e.Domain)
		if domain == "" {
			continue
		}
		out = append(out, ScoreSummary{
			Domain:         domain,
			Types:          append([]string(nil), e.Types...),
			TriageStatus:   e.TriageStatus,
			RelevanceScore: RelevanceScore(e),
			DiscoveredBy:   e.DiscoveredBy,
			LastSeen:       e.LastSeen,
			LastTriageAt:   e.LastTriageAt,
			TriageWhy:      e.TriageWhy,
		})
	}
	return out
}

// TouchTriageScore records triage outcome on registry rows (used by domain triage jobs).
func TouchTriageScore(entry *Entry, status, why string) {
	if entry == nil {
		return
	}
	setEntryTriageStatus(entry, status)
	entry.TriageWhy = strings.TrimSpace(why)
	entry.LastTriageAt = time.Now().UTC().Format(time.RFC3339)
}
