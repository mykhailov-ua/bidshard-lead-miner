package discover

import (
	"testing"

	"github.com/bidshard/parser/internal/sink"
)

func TestEvaluateDorkPrune(t *testing.T) {
	t.Parallel()

	cfg := DefaultDorkPruneConfig()
	rows := []OutcomeDorkRow{
		{Query: "good dork", Accepted: 20, Junk: 5, AcceptRate: 0.8},
		{Query: "bad dork", Accepted: 1, Junk: 49, AcceptRate: 0.02},
		{Query: "low but pilot", Accepted: 2, Junk: 40, AcceptRate: 0.05, OutcomePilot: 1},
		{Query: "small sample", Accepted: 0, Junk: 10, AcceptRate: 0},
	}
	got := EvaluateDorkPrune(rows, cfg)
	if len(got) != 1 || got[0] != "bad dork" {
		t.Fatalf("prune=%v want [bad dork]", got)
	}
}

func TestBuildKeywordTuneRows(t *testing.T) {
	t.Parallel()

	docs := []sink.KeywordStatDoc{
		{KeywordID: "kw_ok", AcceptedCount: 30, JunkCount: 2},
		{KeywordID: "kw_bad", AcceptedCount: 5, JunkCount: 25},
	}
	rows := BuildKeywordTuneRows(docs, map[string]int{"kw_ok": 20, "kw_bad": 20})
	if len(rows) != 1 || rows[0].KeywordID != "kw_bad" || !rows[0].RecommendDisable {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestOutcomeDorkRowTotalOutcomes(t *testing.T) {
	t.Parallel()
	row := OutcomeDorkRow{OutcomeContacted: 1, OutcomePilot: 2}
	if row.TotalOutcomes() != 3 {
		t.Fatalf("total=%d", row.TotalOutcomes())
	}
}
