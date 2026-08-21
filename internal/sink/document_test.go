package sink

import (
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestToLeadDocUpdateBSONOmitsStatus(t *testing.T) {
	t.Parallel()

	doc := ToLeadDoc(model.Lead{
		HashID:   "abc",
		Priority: "High",
		Score:    60,
		Status:   "contacted",
		StatusAt: time.Now().UTC(),
	})
	fields, err := ToLeadDocUpdateBSON(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["status"]; ok {
		t.Fatalf("update fields must not include status: %v", fields)
	}
	if _, ok := fields["status_at"]; ok {
		t.Fatalf("update fields must not include status_at: %v", fields)
	}
	if _, ok := fields["hash_id"]; ok {
		t.Fatalf("update fields must not include hash_id: %v", fields)
	}
	if fields["priority"] != "High" {
		t.Fatalf("priority=%v", fields["priority"])
	}
}
