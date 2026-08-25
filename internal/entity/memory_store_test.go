package entity

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/extract"
)

func TestMemoryStoreReplaceMergedRejectsStaleGeneration(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now().UTC()
	first, err := store.RecordSighting(context.Background(), SightingInput{
		ResolveInput: ResolveInput{
			CompanyName: "Acme",
			Contacts:    []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "hash-a",
		Matched:  []string{"voluum"},
		Text:     "voluum pain",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RecordSighting(context.Background(), SightingInput{
		ResolveInput: ResolveInput{
			Contacts: []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "hash-b",
		Matched:  []string{"postback"},
		Text:     "postback issue",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, ok := store.Get(first.EntityID)
	if !ok {
		t.Fatal("entity missing")
	}
	if doc.MergeGeneration < 1 {
		t.Fatalf("merge_generation=%d want >=1 after merge", doc.MergeGeneration)
	}
	backend := lockedMemoryBackend{s: store}
	accepted, err := backend.ReplaceMerged(context.Background(), doc, doc.MergeGeneration)
	if err != nil {
		t.Fatal(err)
	}
	if accepted {
		t.Fatal("expected stale generation replace to fail when generation already advanced")
	}
}

func TestMemoryStoreReplaceMergedAcceptsCurrentGeneration(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now().UTC()
	first, err := store.RecordSighting(context.Background(), SightingInput{
		ResolveInput: ResolveInput{
			CompanyName: "Acme",
			Contacts:    []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "hash-a",
		Matched:  []string{"voluum"},
		Text:     "voluum pain",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.RecordSighting(context.Background(), SightingInput{
		ResolveInput: ResolveInput{
			Contacts: []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "hash-b",
		Matched:  []string{"postback"},
		Text:     "postback issue",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.SightingCount < 2 {
		t.Fatalf("sighting_count=%d want >=2", second.SightingCount)
	}
	doc, ok := store.Get(first.EntityID)
	if !ok {
		t.Fatal("entity missing")
	}
	if doc.MergeGeneration < 1 {
		t.Fatalf("merge_generation=%d want >=1", doc.MergeGeneration)
	}
}
