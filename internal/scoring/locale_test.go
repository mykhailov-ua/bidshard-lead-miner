package scoring

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSpanishOverlayScoring(t *testing.T) {
	ctx := context.Background()
	reg := NewRegistry("../../data/keywords.json")
	esPath := filepath.Join("../../data", "keywords-es.json")

	if err := reg.LoadWithOverlay(ctx, "../../data/keywords.json", esPath); err != nil {
		t.Fatalf("LoadWithOverlay failed: %v", err)
	}

	leadText := &LeadText{Context: "Buscando voluum alternativa para mi equipo"}
	prio := ScoreText(reg, leadText)

	if leadText.Score < 15 {
		t.Errorf("expected Spanish fixture score >= 15, got %d", leadText.Score)
	}
	if prio == PriorityLow {
		t.Errorf("expected Spanish fixture priority >= Medium, got %s", prio)
	}
}

func TestLocaleOverlayScoring(t *testing.T) {
	ctx := context.Background()
	locales := map[string]string{
		"es": "Buscando voluum alternativa para mi equipo",
		"pt": "Procurando alternativa ao voluum muito caro",
		"pl": "Szukam alternatywa dla voluum bo postback nie dziala",
		"de": "Tracker zu teuer suche alternative zu voluum",
		"fr": "Tracker trop cher alternative a voluum",
	}

	for locale, text := range locales {
		t.Run(locale, func(t *testing.T) {
			reg := NewRegistry("../../data/keywords.json")
			overlay := filepath.Join("../../data", "keywords-"+locale+".json")
			if err := reg.LoadWithOverlay(ctx, "../../data/keywords.json", overlay); err != nil {
				t.Fatalf("load overlay: %v", err)
			}
			leadText := &LeadText{Context: text}
			prio := ScoreText(reg, leadText)
			if leadText.Score < 15 {
				t.Errorf("locale %s score=%d want >=15", locale, leadText.Score)
			}
			if prio == PriorityLow {
				t.Errorf("locale %s priority=%s want >= Medium", locale, prio)
			}
		})
	}
}
