package entity

import (
	"strings"
	"testing"
)

func TestBuildEntityProofSummary(t *testing.T) {
	t.Parallel()

	doc := EntityDoc{
		SightingCount:  3,
		SourceFamilies: []string{"forum", "telegram"},
		UnifiedPain:    "Voluum postback failures",
	}
	got := BuildEntityProofSummary(doc)
	if got == "" {
		t.Fatal("expected proof summary")
	}
	if !strings.Contains(got, "3") || !strings.Contains(got, "Voluum postback") {
		t.Fatalf("proof=%q", got)
	}
}
