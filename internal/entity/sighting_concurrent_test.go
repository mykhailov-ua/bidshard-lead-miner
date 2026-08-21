package entity

import (
	"sync"
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestMemoryStoreConcurrentSightings(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := store.RecordSighting(t.Context(), SightingInput{
				ResolveInput: ResolveInput{
					Source:   "tgweb:example.com",
					Contacts: []extract.Contact{{Type: "email", Value: "ops@example.com"}},
				},
				HashID:  "hash-" + string(rune('a'+n)),
				Matched: []string{"voluum"},
				Text:    "voluum alternative postback failing",
			})
			if err != nil {
				t.Errorf("RecordSighting: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if len(store.docs) != 1 {
		t.Fatalf("entities=%d want 1", len(store.docs))
	}
	var doc EntityDoc
	for _, d := range store.docs {
		doc = *d
	}
	if doc.SightingCount != 8 {
		t.Fatalf("sighting_count=%d want 8", doc.SightingCount)
	}
	if len(doc.HashIDs) != 8 {
		t.Fatalf("hash_ids=%d want 8", len(doc.HashIDs))
	}
}
