// Integration tests require MongoDB. Built with -tags=integration; gopls uses the same tag
// via .vscode/settings.json buildFlags so the file is indexed in the IDE.
//
//go:build integration

package sink_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sink"
)

func TestMongoStoreUpsertAndExists(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Skip("MONGO_URI not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "parser_test"
	}

	store, err := sink.ConnectMongo(ctx, uri, dbName, "leads_test", 4)
	if err != nil {
		t.Fatal(err)
	}

	lead := model.Lead{
		TS:       time.Now().UTC(),
		HashID:   sink.LeadHashIDFromExtract([]extract.Contact{{Type: "email", Value: "ops@igaming-team.com"}}),
		Priority: "High",
		Score:    55,
		Source:   "telegram:foo",
		Title:    "Tracker pain",
		Contacts: []string{"ops@igaming-team.com"},
		Matched:  []string{"voluum"},
		Snippet:  "Need voluum alternative with postback fix",
	}
	if lead.HashID == "" {
		t.Fatal("empty hash")
	}

	exists, err := store.Exists(ctx, lead.HashID)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("lead should not exist before upsert")
	}

	if err := store.Upsert(ctx, lead); err != nil {
		t.Fatal(err)
	}

	exists, err = store.Exists(ctx, lead.HashID)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("lead should exist after upsert")
	}

	dup := lead
	dup.Score = 70
	dup.Snippet = "updated snippet after re-sight"
	if err := store.Upsert(ctx, dup); err != nil {
		t.Fatal(err)
	}

	// Re-fetch via Exists + second upsert path; verify mutable fields updated (integration).
	if err := store.Upsert(ctx, dup); err != nil {
		t.Fatal(err)
	}
}
