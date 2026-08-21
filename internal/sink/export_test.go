package sink

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestResolveExportFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path, format, want string
	}{
		{"leads.jsonl", "auto", ExportFormatNDJSON},
		{"leads.ndjson", "auto", ExportFormatNDJSON},
		{"leads.json", "auto", ExportFormatPretty},
		{"leads.jsonl", "pretty", ExportFormatPretty},
		{"leads.json", "ndjson", ExportFormatNDJSON},
	}
	for _, tc := range cases {
		if got := ResolveExportFormat(tc.path, tc.format); got != tc.want {
			t.Fatalf("ResolveExportFormat(%q, %q) = %q, want %q", tc.path, tc.format, got, tc.want)
		}
	}
}

func TestEncodeLeadExportPretty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	lead := model.Lead{
		HashID:   "abc",
		Priority: "High",
		Score:    62,
		Source:   "stub:telegram_en",
		Contacts: []string{"telegram:@buyer"},
		Matched:  []string{"voluum alternative"},
		TS:       time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
	if err := EncodeLeadExport(&buf, lead, ExportFormatPretty); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"\"hash_id\": \"abc\"", "\"score\": 62", "\"matched\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestEncodeLeadExportNDJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	lead := model.Lead{
		HashID: "abc",
		Source: "stub:telegram_en",
		Score:  62,
	}
	if err := EncodeLeadExport(&buf, lead, ExportFormatNDJSON); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, `"hash_id":"abc"`) && !strings.Contains(out, `"hash_id": "abc"`) {
		t.Fatalf("unexpected ndjson: %q", out)
	}
}
