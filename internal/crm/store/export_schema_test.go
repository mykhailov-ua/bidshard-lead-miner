package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

func TestExportNDJSONUsesLeadDocSchema(t *testing.T) {
	doc := sink.LeadDoc{
		HashID:   "abcdef1234567890abcdef1234567890",
		TS:       time.Now().UTC(),
		Priority: "High",
		Score:    90,
		Source:   "forum:test",
		Status:   "new",
		Snippet:  "Need voluum alternative",
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded sink.LeadDoc
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.HashID != doc.HashID || decoded.Score != doc.Score || decoded.Source != doc.Source {
		t.Fatalf("decoded=%+v", decoded)
	}
}
