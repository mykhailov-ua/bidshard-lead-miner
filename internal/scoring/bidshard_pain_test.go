package scoring

import "testing"

func TestBidShardPainBoost(t *testing.T) {
	t.Parallel()
	base := 20
	got := BidShardPainBoost(base, "keitaro on 200k clicks/day hangs server, admin 2 min load")
	if got < base+30 {
		t.Fatalf("expected volume+bucket boost, got %d", got)
	}
}

func TestHasVolumeSignal(t *testing.T) {
	t.Parallel()
	if !HasVolumeSignal("running 200k clicks/day on keitaro") {
		t.Fatal("expected volume signal")
	}
}
