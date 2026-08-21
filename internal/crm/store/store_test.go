package store

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestListFilterClampLimit(t *testing.T) {
	if got := (ListFilter{Limit: 0}).clampLimit(); got != 10 {
		t.Fatalf("default limit=%d want 10", got)
	}
	if got := (ListFilter{Limit: 999}).clampLimit(); got != 50 {
		t.Fatalf("max limit=%d want 50", got)
	}
}

func TestListFilterMatchQueryStatus(t *testing.T) {
	q := ListFilter{Status: "new"}.matchQuery()
	if q["status"] != "new" {
		t.Fatalf("status filter missing: %v", q)
	}
}

func TestListFilterMatchQueryCursor(t *testing.T) {
	q := ListFilter{
		Cursor: &ListCursor{Score: 80, HashID: "abc"},
	}.matchQuery()
	orClause, ok := q["$or"].(bson.A)
	if !ok || len(orClause) != 2 {
		t.Fatalf("$or cursor clause missing: %v", q)
	}
}
