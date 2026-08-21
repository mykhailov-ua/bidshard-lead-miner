package gemini

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAnalyzeJunkBatch(t *testing.T) {
	t.Parallel()

	id := primitive.NewObjectID()
	mockID := id.Hex()
	cl := newTestClient(t, `{"items":[{"id":"`+mockID+`","category":"false_negative","why":"mentions voluum billing pain","suggestions":["lower score threshold"]}]}`)

	results, err := cl.AnalyzeJunkBatch(context.Background(), []sink.JunkDoc{{
		ID:      id,
		Source:  "stub:test",
		Reason:  "low_score",
		Snippet: "voluum alternative billing too high",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d want 1", len(results))
	}
	if results[0].ID != id.Hex() {
		t.Fatalf("id=%q", results[0].ID)
	}
	if results[0].Category != "false_negative" {
		t.Fatalf("category=%q", results[0].Category)
	}
}

func TestBuildJunkReport(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"summary":"Most drops are prescan misses.","top_reasons":[{"reason":"keyword_prescan","count":12,"why":"missing synonyms"}],"false_negative_candidates":3,"recommendations":["add voluum synonyms"]}`)

	result, err := cl.BuildJunkReport(context.Background(), ReportInput{
		PeriodFrom:  "2026-01-01T00:00:00Z",
		PeriodTo:    "2026-01-02T00:00:00Z",
		TotalJunk:   20,
		ReasonStats: []sink.ReasonCount{{Reason: "keyword_prescan", Count: 12}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary == "" {
		t.Fatal("expected summary")
	}
	if result.FalseNegativeCandidates != 3 {
		t.Fatalf("false_negative=%d", result.FalseNegativeCandidates)
	}
}
