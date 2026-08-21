//go:build integration

package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sink"
)

func TestLeadStoreListAndGet(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Skip("MONGO_URI not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "parser_test"
	}
	collName := "leads_crm_test"

	writeStore, err := sink.ConnectMongo(ctx, uri, dbName, collName, 4)
	if err != nil {
		t.Fatal(err)
	}

	readClient, err := sink.ConnectMongoClient(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		discCtx, discCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer discCancel()
		_ = readClient.Disconnect(discCtx)
	}()

	lead := model.Lead{
		TS:       time.Now().UTC(),
		HashID:   sink.LeadHashIDFromExtract([]extract.Contact{{Type: "email", Value: "crm-test@example.com"}}),
		Priority: "High",
		Score:    90,
		Source:   "forum:test",
		Title:    "CRM store test",
		Contacts: []string{"crm-test@example.com"},
		Matched:  []string{"voluum"},
		Snippet:  "Need voluum alternative",
		Status:   "new",
	}
	if lead.HashID == "" {
		t.Fatal("empty hash_id")
	}
	if err := writeStore.Upsert(ctx, lead); err != nil {
		t.Fatal(err)
	}

	leadStore := store.New(readClient, store.Options{
		DBName:          dbName,
		LeadsCollection: collName,
		QueryTimeout:    5 * time.Second,
		WriteTimeout:    3 * time.Second,
	})

	got, err := leadStore.GetByHashID(ctx, lead.HashID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HashID != lead.HashID {
		t.Fatalf("hash_id=%q want %q", got.HashID, lead.HashID)
	}

	list, err := leadStore.List(ctx, store.ListFilter{Status: "new", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, doc := range list.Leads {
		if doc.HashID == lead.HashID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("lead not found in list")
	}

	_, err = leadStore.GetByHashID(ctx, "missing-hash-id")
	if err != store.ErrNotFound {
		t.Fatalf("err=%v want ErrNotFound", err)
	}

	if err := leadStore.UpdateStatus(ctx, lead.HashID, "contacted"); err != nil {
		t.Fatal(err)
	}
	updated, err := leadStore.GetByHashID(ctx, lead.HashID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "contacted" {
		t.Fatalf("status=%q want contacted", updated.Status)
	}
	if updated.StatusAt.IsZero() {
		t.Fatal("expected status_at after update")
	}

	resolved, err := leadStore.ResolveHashID(ctx, lead.HashID[:8])
	if err != nil {
		t.Fatal(err)
	}
	if resolved != lead.HashID {
		t.Fatalf("resolved=%q want %q", resolved, lead.HashID)
	}

	if err := leadStore.UpdateStatus(ctx, "missing-hash-id", "won"); err != store.ErrNotFound {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
	if err := leadStore.UpdateStatus(ctx, lead.HashID, "invalid"); err == nil {
		t.Fatal("expected invalid status error")
	}
}
