package store

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestListFilterMatchQueryInboxNew(t *testing.T) {
	q := ListFilter{Status: "new", InboxOnly: true}.matchQuery()
	if q["status"] != "new" {
		t.Fatalf("status=%v want new", q["status"])
	}
	nin, ok := q["analysis_status"].(bson.M)["$nin"].([]string)
	if !ok || len(nin) != 1 || nin[0] != "pending" {
		t.Fatalf("analysis_status filter=%v", q["analysis_status"])
	}
}

func TestListFilterMatchQueryInboxDefault(t *testing.T) {
	q := ListFilter{InboxOnly: true}.matchQuery()
	nin, ok := q["status"].(bson.M)["$nin"].([]string)
	if !ok || len(nin) != 2 {
		t.Fatalf("status nin=%v", q["status"])
	}
}

func TestListFilterMatchQueryWithoutInbox(t *testing.T) {
	q := ListFilter{Status: "new"}.matchQuery()
	if q["status"] != "new" {
		t.Fatalf("status=%v want new", q["status"])
	}
	if _, ok := q["analysis_status"]; ok {
		t.Fatal("expected no analysis_status filter without inbox")
	}
}
