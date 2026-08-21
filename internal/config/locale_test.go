package config

import "testing"

func TestKeywordOverlayPaths(t *testing.T) {
	t.Parallel()

	if got := KeywordOverlayPaths("", ""); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if got := KeywordOverlayPaths("es,pt", ""); len(got) != 2 {
		t.Fatalf("locales=%v", got)
	}
	if got := KeywordOverlayPaths("es,pt", ""); got[0] != "data/keywords-es.json" || got[1] != "data/keywords-pt.json" {
		t.Fatalf("paths=%v", got)
	}
	if got := KeywordOverlayPaths("es", "custom/overlay.json"); len(got) != 1 || got[0] != "custom/overlay.json" {
		t.Fatalf("locale path override=%v", got)
	}
}
