package scoring

import "testing"

func TestHasBuyerIntentSignal(t *testing.T) {
	t.Parallel()
	if !HasBuyerIntentSignal("I'm looking for a voluum alternative for our media buying team?") {
		t.Fatal("expected buyer intent")
	}
	if HasBuyerIntentSignal("companies migrating from voluum in 2025") {
		t.Fatal("expected editorial without buyer intent")
	}
}
