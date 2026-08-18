package scoring

import "testing"

func TestApplySpendGateBoost(t *testing.T) {
	t.Parallel()
	score := ApplySpendGate(20, "we run $50k/mo on traffic with voluum pain", 15)
	if score < 35 {
		t.Fatalf("score=%d want boost above medium", score)
	}
}

func TestApplySpendGateCapWithoutSignals(t *testing.T) {
	t.Parallel()
	score := ApplySpendGate(25, "generic affiliate marketing discussion", 15)
	if score > 14 {
		t.Fatalf("score=%d want capped below medium", score)
	}
}

func TestApplySpendGateCompetitorBypassesCap(t *testing.T) {
	t.Parallel()
	score := ApplySpendGate(25, "keitaro keeps crashing on postbacks", 15)
	if score != 25 {
		t.Fatalf("score=%d want uncapped competitor mention", score)
	}
}
