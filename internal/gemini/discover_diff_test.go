package gemini

import (
	"context"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/discover"
)

func TestBuildDiscoverICPDiff(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"add_telegram_search":["binom migration pain"],"add_serp_dorks":["site:t.me binom alternative"],"summary":"expand binom queries"}`)

	diff, err := cl.BuildDiscoverICPDiff(context.Background(), discover.ICPConfig{
		TelegramSearch: []string{"voluum alternative"},
		SerpDorks:      []string{"site:t.me voluum"},
	}, []string{"binom"}, []string{"switching from binom"})
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "pending" {
		t.Fatalf("status=%q", diff.Status)
	}
	if len(diff.AddTelegramSearch) != 1 || !strings.Contains(diff.AddTelegramSearch[0], "binom") {
		t.Fatalf("telegram=%v", diff.AddTelegramSearch)
	}
}
