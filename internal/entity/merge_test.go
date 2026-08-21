package entity

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestMergeSightingAddsSourceFamily(t *testing.T) {
	doc := NewDoc("ent-1", EntityKey{Kind: KindCompany, Value: "affnet media"}, []EntityKey{
		{Kind: KindCompany, Value: "affnet media"},
	}, SightingInput{
		ResolveInput: ResolveInput{
			CompanyName: "AffNet Media",
			Source:      "telegram:@affnet",
			Contacts:    []extract.Contact{{Type: "telegram", Value: "@buyer_mx"}},
		},
		HashID:  "hash-a",
		Matched: []string{"voluum"},
		Text:    "voluum alternative postback failing",
	})

	result := MergeSighting(&doc, []EntityKey{
		{Kind: KindDomain, Value: "affnet.com"},
	}, SightingInput{
		ResolveInput: ResolveInput{
			Source:   "tgweb:affnet.com",
			Contacts: []extract.Contact{{Type: "email", Value: "ops@affnet.com"}},
		},
		HashID:  "hash-b",
		Matched: []string{"postback"},
		Text:    "postback failing on FTD",
	})

	if !result.NewSourceFamily {
		t.Fatal("expected new source family")
	}
	if doc.SourceCount != 2 {
		t.Fatalf("source_count=%d want 2", doc.SourceCount)
	}
	if doc.SightingCount != 2 {
		t.Fatalf("sighting_count=%d want 2", doc.SightingCount)
	}
	if doc.PainHits != 2 {
		t.Fatalf("pain_hits=%d want 2", doc.PainHits)
	}
	if doc.MergeGeneration != 1 {
		t.Fatalf("merge_generation=%d want 1", doc.MergeGeneration)
	}
}

func TestMemoryStoreLinksBySharedDomain(t *testing.T) {
	store := NewMemoryStore()

	first, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			CompanyName: "AffNet Media",
			Source:      "telegram:@affnet",
			Contacts: []extract.Contact{
				{Type: "email", Value: "ops@affnet.com"},
				{Type: "telegram", Value: "@buyer_mx"},
			},
		},
		HashID:  "hash-a",
		Matched: []string{"voluum"},
		Text:    "voluum alternative",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.NewEntity {
		t.Fatal("expected new entity")
	}

	second, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			Source: "reddit:igaming",
			Contacts: []extract.Contact{
				{Type: "email", Value: "partners@affnet.com"},
			},
		},
		HashID:  "hash-b",
		Matched: []string{"postback"},
		Text:    "postback failing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.EntityID != first.EntityID {
		t.Fatalf("entity ids differ: %q vs %q", second.EntityID, first.EntityID)
	}
	if !second.NewSourceFamily {
		t.Fatal("expected cross-source family")
	}

	doc, ok := store.Get(first.EntityID)
	if !ok {
		t.Fatal("entity missing")
	}
	if len(doc.HashIDs) != 2 {
		t.Fatalf("hash_ids=%v", doc.HashIDs)
	}
	if len(doc.Sightings) != 2 {
		t.Fatalf("sightings=%d want 2", len(doc.Sightings))
	}
}
