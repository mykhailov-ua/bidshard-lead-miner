package telegrambot

import (
	"context"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/sink"
)

func TestFormatLeadNotifyHTML(t *testing.T) {
	t.Parallel()
	card := FormatLeadNotifyHTML(sink.LeadDoc{
		HashID:     "abc123",
		Score:      72,
		Priority:   "High",
		Source:     "telegram:@voluum",
		Snippet:    "postback failing from voluum to meta",
		Matched:    []string{"postback(+15)", "voluum(+12)"},
		Contacts:   []sink.StoredContact{{Type: "telegram", Value: "@buyer_mx"}},
		GeoCountry: "US",
		ICP:        "in_house_buyer",
		HeatTier:   "warm",
	})
	if !strings.Contains(card, "<b>New lead</b>") {
		t.Fatalf("missing header: %s", card)
	}
	if !strings.Contains(card, "Score: <b>72</b> High") {
		t.Fatalf("missing score: %s", card)
	}
	if !strings.Contains(card, "telegram:@voluum") {
		t.Fatalf("missing source: %s", card)
	}
	if !strings.Contains(card, "postback failing") {
		t.Fatalf("missing snippet: %s", card)
	}
}

func TestFormatLeadNotifyHTMLMinScoreGate(t *testing.T) {
	t.Parallel()
	client := NewClient("token")
	n := NewLeadNotifier(client, []int64{-1001}, 50, 70)
	if n == nil {
		t.Fatal("notifier nil")
	}
	// NotifyLead with low score should not panic (async no-op before network).
	n.NotifyLead(context.Background(), sink.LeadDoc{HashID: "x", Score: 10})
}
