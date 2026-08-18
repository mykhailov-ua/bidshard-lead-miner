package scoring

import "testing"

func TestCompetitorPainBoost(t *testing.T) {
	t.Parallel()
	score := CompetitorPainBoost(10, "voluum bill too high [stack:keitaro]", []string{"keitaro"})
	if score != 30 {
		t.Fatalf("score=%d want 30", score)
	}
}

func TestDetectCompetitorStack(t *testing.T) {
	t.Parallel()
	stack := DetectCompetitorStack(`<script src="https://voluumtrk.com/click"></script>`)
	if len(stack) == 0 {
		t.Fatal("expected stack hit")
	}
}
