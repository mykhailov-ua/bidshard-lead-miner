package entity

import (
	"testing"
	"time"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
)

func TestCrossSourceHotRequiresTwoFamiliesAndPain(t *testing.T) {
	now := time.Now().UTC()
	doc := EntityDoc{
		SourceCount: 2,
		PainHits:    1,
		FirstSeen:   now.Add(-2 * 24 * time.Hour),
		LastSeen:    now,
	}
	if !CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected cross-source hot")
	}

	doc.SourceCount = 1
	if CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected false with one source family")
	}

	doc.SourceCount = 2
	doc.PainHits = 0
	if CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected false without pain hits")
	}

	doc.PainHits = 1
	doc.FirstSeen = now.Add(-10 * 24 * time.Hour)
	doc.LastSeen = now
	if CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected false outside window")
	}
}

func TestApplyCrossSourceHotBoostPromotesMedium(t *testing.T) {
	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(t.Context()); err != nil {
		t.Fatal(err)
	}

	lead := model.Lead{
		Priority: string(scoring.PriorityMedium),
		Score:    40,
	}
	ApplyCrossSourceHotBoost(&lead, reg, 20)
	if !lead.Hot {
		t.Fatal("expected hot=true")
	}
	if lead.Priority != string(scoring.PriorityHigh) {
		t.Fatalf("priority=%s want High", lead.Priority)
	}
	if lead.Score != 60 {
		t.Fatalf("score=%d want 60", lead.Score)
	}
	foundTag := false
	for _, tag := range lead.Tags {
		if tag == TagCrossSourceHot {
			foundTag = true
			break
		}
	}
	if !foundTag {
		t.Fatalf("tags=%v", lead.Tags)
	}
}

func TestMemoryStoreCrossSourceHotAfterSecondFamily(t *testing.T) {
	store := NewMemoryStore()
	contacts := []extract.Contact{{Type: "email", Value: "ops@affnet.com"}}

	_, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			CompanyName: "AffNet Media",
			Source:      "telegram:@affnet",
			Contacts:    contacts,
		},
		HashID:  "hash-a",
		Matched: []string{"voluum"},
		Text:    "voluum alternative",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			Source:   "reddit:igaming",
			Contacts: contacts,
		},
		HashID:  "hash-b",
		Matched: []string{"postback"},
		Text:    "postback failing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.CrossSourceHot {
		t.Fatalf("expected cross-source hot result=%+v", result)
	}
}
