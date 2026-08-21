package entity

import (
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/extract"
)

func TestSightingEventTimePrefersPostedAt(t *testing.T) {
	posted := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	seen := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	got := sightingEventTime(SightingInput{
		PostedAt: posted,
		SeenAt:   seen,
	})
	if !got.Equal(posted) {
		t.Fatalf("event time=%v want posted %v", got, posted)
	}
}

func TestNewDocUsesPostedAtForFirstSeen(t *testing.T) {
	posted := time.Now().UTC().Add(-30 * 24 * time.Hour)
	seen := time.Now().UTC()

	doc := NewDoc("ent-1", EntityKey{Kind: KindDomain, Value: "affnet.com"}, []EntityKey{
		{Kind: KindDomain, Value: "affnet.com"},
	}, SightingInput{
		ResolveInput: ResolveInput{Source: "forum:affiliatefix"},
		HashID:       "hash-a",
		Matched:      []string{"postback failing"},
		Text:         "postback failing on FTD",
		Score:        40,
		PostedAt:     posted,
		SeenAt:       seen,
	})

	if !doc.FirstSeen.Equal(posted) {
		t.Fatalf("first_seen=%v want %v", doc.FirstSeen, posted)
	}
	if !doc.LastSeen.Equal(posted) {
		t.Fatalf("last_seen=%v want %v", doc.LastSeen, posted)
	}
	if len(doc.Sightings) != 1 {
		t.Fatalf("sightings=%d want 1", len(doc.Sightings))
	}
	if !doc.Sightings[0].PostedAt.Equal(posted) {
		t.Fatalf("sighting posted_at=%v want %v", doc.Sightings[0].PostedAt, posted)
	}
	if doc.Sightings[0].SeenAt.Before(seen.Add(-time.Second)) || doc.Sightings[0].SeenAt.After(seen.Add(time.Second)) {
		t.Fatalf("sighting seen_at=%v want ~%v", doc.Sightings[0].SeenAt, seen)
	}
}

func TestRefreshEntitySightingDedupHash(t *testing.T) {
	posted := time.Now().UTC().Add(-10 * 24 * time.Hour)
	doc := NewDoc("ent-1", EntityKey{Kind: KindDomain, Value: "affnet.com"}, []EntityKey{
		{Kind: KindDomain, Value: "affnet.com"},
	}, SightingInput{
		ResolveInput: ResolveInput{Source: "forum:affiliatefix"},
		HashID:       "hash-a",
		Matched:      []string{"voluum"},
		Text:         "voluum alternative",
		PostedAt:     posted,
		SeenAt:       posted.Add(2 * time.Hour),
	})

	beforeCount := doc.SightingCount
	refresh := RefreshEntitySighting(&doc, SightingInput{
		ResolveInput: ResolveInput{Source: "forum:affiliatefix"},
		HashID:       "hash-a",
		Matched:      []string{"voluum"},
		Text:         "voluum alternative updated",
		PostedAt:     posted,
		SeenAt:       time.Now().UTC(),
	})
	if refresh.SightingCount != beforeCount {
		t.Fatalf("sighting_count=%d want %d", refresh.SightingCount, beforeCount)
	}
	if len(doc.Sightings) != 1 {
		t.Fatalf("sightings=%d want 1", len(doc.Sightings))
	}
	if doc.Sightings[0].Snippet != "voluum alternative updated" {
		t.Fatalf("snippet=%q", doc.Sightings[0].Snippet)
	}
}

func TestTrimEntitySightingsKeepsNewest(t *testing.T) {
	doc := EntityDoc{EntityID: "ent-1"}
	now := time.Now().UTC()
	for i := 0; i < MaxEntitySightings+5; i++ {
		appendOrRefreshEntitySighting(&doc, SightingInput{
			ResolveInput: ResolveInput{Source: "forum:affiliatefix"},
			HashID:       "hash-" + strconv.Itoa(i),
			Text:         "pain",
			PostedAt:     now.Add(-time.Duration(i) * 24 * time.Hour),
			SeenAt:       now,
		})
	}
	if len(doc.Sightings) != MaxEntitySightings {
		t.Fatalf("sightings=%d want %d", len(doc.Sightings), MaxEntitySightings)
	}
	newest := sightingEventTimeOf(doc.Sightings[0])
	oldestKept := sightingEventTimeOf(doc.Sightings[len(doc.Sightings)-1])
	if newest.Before(oldestKept) {
		t.Fatal("expected newest-first ordering after trim")
	}
}

func TestMergeSightingUpdatesFirstSeenWhenOlderPostLinked(t *testing.T) {
	recent := time.Now().UTC().Add(-3 * 24 * time.Hour)
	old := time.Now().UTC().Add(-40 * 24 * time.Hour)

	doc := NewDoc("ent-1", EntityKey{Kind: KindTelegram, Value: "@buyer_mx"}, []EntityKey{
		{Kind: KindTelegram, Value: "@buyer_mx"},
	}, SightingInput{
		ResolveInput: ResolveInput{
			Source:   "telegram:@buyer",
			Contacts: []extract.Contact{{Type: "telegram", Value: "@buyer_mx"}},
		},
		HashID:   "hash-recent",
		Matched:  []string{"voluum"},
		Text:     "voluum alternative",
		PostedAt: recent,
		SeenAt:   recent,
	})

	MergeSighting(&doc, []EntityKey{
		{Kind: KindDomain, Value: "affnet.com"},
	}, SightingInput{
		ResolveInput: ResolveInput{
			Source:   "forum:affiliatefix",
			Contacts: []extract.Contact{{Type: "email", Value: "ops@affnet.com"}},
		},
		HashID:   "hash-old",
		Matched:  []string{"postback failing"},
		Text:     "postback failing",
		PostedAt: old,
		SeenAt:   time.Now().UTC(),
	})

	if !doc.FirstSeen.Equal(old) {
		t.Fatalf("first_seen=%v want %v", doc.FirstSeen, old)
	}
	if len(doc.Sightings) != 2 {
		t.Fatalf("sightings=%d want 2", len(doc.Sightings))
	}
}

func TestCrossSourceHotUsesFirstLastSeenSpan(t *testing.T) {
	now := time.Now().UTC()
	doc := EntityDoc{
		SourceCount: 2,
		PainHits:    1,
		FirstSeen:   now.Add(-5 * 24 * time.Hour),
		LastSeen:    now.Add(-2 * 24 * time.Hour),
	}
	if !CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected hot when span within window")
	}
	doc.FirstSeen = now.Add(-10 * 24 * time.Hour)
	if CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected false when span exceeds window")
	}
}
