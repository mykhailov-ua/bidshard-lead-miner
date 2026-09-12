package discover

import "testing"

func TestScoreInfraCluster(t *testing.T) {
	t.Parallel()
	score := ScoreInfraCluster(InfraCluster{
		Domains:     []string{"a.com", "b.com", "c.com", "d.com"},
		TrackerHint: "keitaro kclick_id",
		Country:     "US",
		SampleDomain: "a.com",
	})
	if score < 50 {
		t.Fatalf("score=%d want >= 50", score)
	}
}
