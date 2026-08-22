package entity

import (
	"fmt"
	"strings"
)

// BuildEntityProofSummary formats multi-thread proof for linked leads (no PII).
func BuildEntityProofSummary(doc EntityDoc) string {
	if doc.SightingCount < 2 {
		return ""
	}
	pain := strings.TrimSpace(doc.UnifiedPain)
	sources := len(doc.SourceFamilies)
	if sources == 0 {
		sources = len(doc.Sources)
	}
	if pain == "" {
		return fmt.Sprintf("Same actor across %d sightings on %d sources.", doc.SightingCount, maxInt(sources, 1))
	}
	return fmt.Sprintf("Same actor across %d sightings on %d sources. Pain: %s", doc.SightingCount, maxInt(sources, 1), pain)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
