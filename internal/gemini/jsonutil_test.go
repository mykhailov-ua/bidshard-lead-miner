package gemini

import (
	"testing"
)

func TestExtractModelJSONStripsFences(t *testing.T) {
	raw := []byte("```json\n{\"icp\":\"starter\",\"hot\":true,\"spend_tier\":\"unknown\"}\n```")
	var out icpResponse
	if err := decodeModelJSON(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.ICP != "starter" || !out.Hot {
		t.Fatalf("got %+v", out)
	}
}

func TestDecodeModelJSONRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"icp":"none","hot":false,"spend_tier":"unknown","extra":1}`)
	var out icpResponse
	if err := decodeModelJSON(raw, &out); err == nil {
		t.Fatal("expected unknown field error")
	}
}
