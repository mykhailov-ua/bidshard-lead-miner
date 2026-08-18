package sink

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestStatusDefaults(t *testing.T) {
	lead := model.Lead{
		HashID: "hash123",
		Title:  "Test Lead",
	}

	doc := ToLeadDoc(lead)
	if doc.Status != "new" {
		t.Errorf("got status %q, want %q", doc.Status, "new")
	}
	if doc.StatusAt.IsZero() {
		t.Errorf("expected non-zero status_at time")
	}
}

func TestExplicitStatusPreserved(t *testing.T) {
	now := time.Now().UTC()
	lead := model.Lead{
		HashID:   "hash123",
		Status:   "contacted",
		StatusAt: now,
	}

	doc := ToLeadDoc(lead)
	if doc.Status != "contacted" {
		t.Errorf("got status %q, want %q", doc.Status, "contacted")
	}
	if !doc.StatusAt.Equal(now) {
		t.Errorf("status_at not preserved")
	}
}

type setOnInsertStore struct {
	docs map[string]LeadDoc
}

func (s *setOnInsertStore) Exists(ctx context.Context, hashID string) (bool, error) {
	_, ok := s.docs[hashID]
	return ok, nil
}

func (s *setOnInsertStore) Upsert(ctx context.Context, lead model.Lead) error {
	doc := ToLeadDoc(lead)
	if _, ok := s.docs[doc.HashID]; !ok {
		s.docs[doc.HashID] = doc
	}
	return nil
}

func (s *setOnInsertStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	doc, ok := s.docs[hashID]
	if !ok {
		return nil
	}
	doc.Status = status
	doc.StatusAt = time.Now().UTC()
	s.docs[hashID] = doc
	return nil
}

func TestUpsertPreservesStatusOnDuplicate(t *testing.T) {
	store := &setOnInsertStore{docs: map[string]LeadDoc{}}
	ctx := context.Background()

	lead := model.Lead{HashID: "abc", Title: "first", Status: "new"}
	if err := store.Upsert(ctx, lead); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(ctx, "abc", "contacted"); err != nil {
		t.Fatal(err)
	}

	lead2 := model.Lead{HashID: "abc", Title: "second", Status: "new"}
	if err := store.Upsert(ctx, lead2); err != nil {
		t.Fatal(err)
	}
	if store.docs["abc"].Status != "contacted" {
		t.Fatalf("status overwritten: %q", store.docs["abc"].Status)
	}
	if store.docs["abc"].Title != "first" {
		t.Fatalf("title overwritten on setOnInsert replay")
	}
}

func TestUpdateStatusTransition(t *testing.T) {
	store := &setOnInsertStore{docs: map[string]LeadDoc{}}
	ctx := context.Background()

	lead := model.Lead{HashID: "abc", Title: "lead", Status: "new"}
	if err := store.Upsert(ctx, lead); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(ctx, "abc", "contacted"); err != nil {
		t.Fatal(err)
	}
	if store.docs["abc"].Status != "contacted" {
		t.Fatalf("status=%q want contacted", store.docs["abc"].Status)
	}
}
