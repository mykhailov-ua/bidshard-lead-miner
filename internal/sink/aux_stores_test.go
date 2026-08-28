package sink

import "testing"

func TestSourceStatsBoostOutcomeWeight(t *testing.T) {
	t.Parallel()

	doc := SourceStatsDoc{
		Source:           "forum:affiliatefix.com",
		Accepted:         40,
		Junk:             10,
		OutcomePilot:     2,
		OutcomeReplied:   1,
	}
	boost := SourceStatsBoost(doc)
	if boost < 12 {
		t.Fatalf("boost=%d want outcome-weighted boost", boost)
	}
}

func TestOutcomeField(t *testing.T) {
	t.Parallel()
	if outcomeField("pilot_started") != "outcome_pilot" {
		t.Fatal("unexpected field mapping")
	}
	if outcomeField("bogus") != "" {
		t.Fatal("expected empty for unknown outcome")
	}
}
