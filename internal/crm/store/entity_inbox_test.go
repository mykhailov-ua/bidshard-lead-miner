package store

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestLinkedLeadsFilter(t *testing.T) {
	t.Parallel()

	q := linkedLeadsFilter("ent-1", []string{"h1", "h2"})
	or, ok := q["$or"].(bson.A)
	if !ok || len(or) != 2 {
		t.Fatalf("filter=%v", q)
	}
	if linkedLeadsFilter("", nil) != nil {
		t.Fatal("expected nil for empty input")
	}
	if f := linkedLeadsFilter("ent-1", nil); f["entity_id"] != "ent-1" {
		t.Fatalf("entity filter=%v", f)
	}
}
