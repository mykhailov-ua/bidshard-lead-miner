package store

import "testing"

func TestDeleteFilterValidate(t *testing.T) {
	t.Parallel()
	if err := (DeleteFilter{}).Validate(); err == nil {
		t.Fatal("expected error for empty filter")
	}
	if err := (DeleteFilter{All: true}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (DeleteFilter{Status: "new"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFilterMatchQuery(t *testing.T) {
	t.Parallel()
	q := DeleteFilter{SourcePrefix: "tgweb:", ScoreMax: 20}.matchQuery()
	if q["score"] == nil {
		t.Fatal("expected score filter")
	}
	if q["source"] == nil {
		t.Fatal("expected source prefix filter")
	}
}
