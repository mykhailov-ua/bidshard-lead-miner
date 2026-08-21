package entity

import (
	"testing"
	"time"

	"github.com/bidshard/parser/internal/extract"
)

func TestEntityIDForSplitSharedDomain(t *testing.T) {
	keys := []EntityKey{{Kind: KindDomain, Value: "acme.com"}}
	base := EntityID(keys)
	fork := EntityIDForSplit(keys, "hash-b", base)
	if fork == base {
		t.Fatal("expected forked entity_id for shared-domain split")
	}
	if fork2 := EntityIDForSplit(keys, "hash-b", "other-entity"); fork2 != base {
		t.Fatalf("expected base id when no collision, got %q", fork2)
	}
}

func TestSplitHashFromEntity(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	heat := DefaultHeatConfig()

	domainKeys := []EntityKey{{Kind: KindDomain, Value: "acme.com"}}
	sourceID := EntityID(domainKeys)

	doc := NewDocWithHeat(sourceID, domainKeys[0], domainKeys, SightingInput{
		ResolveInput: ResolveInput{Source: "forum:affiliatefix.com/thread-a"},
		HashID:       "hash-a",
		Matched:      []string{"voluum"},
		Text:         "voluum bill too high",
		Score:        60,
		PostedAt:     now.Add(-24 * time.Hour),
		SeenAt:       now,
	}, heat, now)

	_ = MergeSightingWithHeat(&doc, []EntityKey{
		{Kind: KindDomain, Value: "acme.com"},
	}, SightingInput{
		ResolveInput: ResolveInput{Source: "tgweb:@ch:acme.com"},
		HashID:       "hash-b",
		Matched:      []string{"tracker"},
		Text:         "need better tracker",
		Score:        55,
		PostedAt:     now.Add(-48 * time.Hour),
		SeenAt:       now,
	}, heat, now)

	doc.UnifiedPain = "billing pain"
	doc.ActorConfidence = 0.4
	doc.NeedsReview = true

	sightB, ok := findSighting(doc, "hash-b")
	if !ok {
		t.Fatal("hash-b sighting missing")
	}
	splitIn := SightingInputFromLead(LeadSightingSource{
		Source:  "tgweb:@ch:acme.com",
		Snippet: "need better tracker",
		Matched: []string{"tracker"},
		Score:   55,
	}, sightB)

	result, newDoc, err := SplitHashFromEntity(&doc, "hash-b", splitIn, heat)
	if err != nil {
		t.Fatal(err)
	}
	if result.NewEntityID == "" || result.SourceEntityID != sourceID {
		t.Fatalf("unexpected split result: %+v", result)
	}
	if result.NewEntityID == sourceID {
		t.Fatal("shared-domain split must fork entity_id")
	}
	if result.HashID != "hash-b" {
		t.Fatalf("hash_id=%q", result.HashID)
	}
	if result.SourceDeleted {
		t.Fatal("expected source entity to remain")
	}
	if len(doc.HashIDs) != 1 || doc.HashIDs[0] != "hash-a" {
		t.Fatalf("remaining hashes=%v", doc.HashIDs)
	}
	if doc.UnifiedPain != "" || doc.NeedsReview {
		t.Fatal("expected classification cleared on source")
	}
	if newDoc.EntityID != result.NewEntityID {
		t.Fatalf("new entity_id=%q result=%q", newDoc.EntityID, result.NewEntityID)
	}
	if len(newDoc.HashIDs) != 1 || newDoc.HashIDs[0] != "hash-b" {
		t.Fatalf("new hashes=%v", newDoc.HashIDs)
	}
}

func TestSplitHashFromEntityLastHashDeletesSource(t *testing.T) {
	heat := DefaultHeatConfig()
	now := time.Now().UTC()
	doc := NewDocWithHeat("only", EntityKey{Kind: KindDomain, Value: "solo.com"}, []EntityKey{
		{Kind: KindDomain, Value: "solo.com"},
	}, SightingInput{
		ResolveInput: ResolveInput{Source: "forum:affiliatefix.com/x"},
		HashID:       "only-hash",
		Text:         "solo post",
		SeenAt:       now,
	}, heat, now)

	sight := doc.Sightings[0]
	splitIn := SightingInputFromLead(LeadSightingSource{
		Source: "forum:affiliatefix.com/x",
		Contacts: []extract.Contact{
			{Type: "email", Value: "buyer@solo.com"},
		},
		Snippet: "solo post",
	}, sight)

	result, _, err := SplitHashFromEntity(&doc, "only-hash", splitIn, heat)
	if err != nil {
		t.Fatal(err)
	}
	if !result.SourceDeleted {
		t.Fatal("expected source deleted flag when last hash removed")
	}
	if len(doc.HashIDs) != 0 {
		t.Fatalf("expected empty source hashes, got %v", doc.HashIDs)
	}
}

func TestSplitHashFromEntityMissingHash(t *testing.T) {
	doc := EntityDoc{EntityID: "x", HashIDs: []string{"a"}}
	_, _, err := SplitHashFromEntity(&doc, "missing", SightingInput{
		ResolveInput: ResolveInput{Source: "forum:x"},
		HashID:       "missing",
	}, DefaultHeatConfig())
	if err != ErrSplitHashMissing {
		t.Fatalf("err=%v", err)
	}
}

func findSighting(doc EntityDoc, hashID string) (EntitySighting, bool) {
	for _, s := range doc.Sightings {
		if s.HashID == hashID {
			return s, true
		}
	}
	return EntitySighting{}, false
}
