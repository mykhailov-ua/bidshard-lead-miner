package tail

import (
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

func TestFormatLine(t *testing.T) {
	t.Parallel()
	line := FormatLine(sink.LeadDoc{
		TS:         time.Date(2026, 9, 11, 12, 30, 0, 0, time.UTC),
		Score:      80,
		Priority:   "High",
		Source:     "forum:affiliatefix",
		Matched:    []string{"voluum alternative"},
		Snippet:    "looking for tracker migration help",
		Contacts:   []sink.StoredContact{{Type: "telegram", Value: "@buyer_mx"}},
		GeoCountry: "MX",
	})
	for _, part := range []string{"score=80", "High", "forum:affiliatefix", "@buyer_mx", "voluum"} {
		if !strings.Contains(line, part) {
			t.Fatalf("missing %q in line: %s", part, line)
		}
	}
}
