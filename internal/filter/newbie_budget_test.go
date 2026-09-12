package filter

import "testing"

func TestRejectNewbieNoBudget(t *testing.T) {
	t.Parallel()
	drop, reason := RejectNewbieNoBudget(
		"Hello guys, I want to start my journey promoting CPA offers and I can't afford VOLUUM cost now",
		"Looking for Best Cheap VOLUUM alternative",
	)
	if !drop || reason != "newbie no budget" {
		t.Fatalf("drop=%v reason=%q", drop, reason)
	}
	if drop, _ := RejectNewbieNoBudget("voluum postback failing after 2M clicks/day", ""); drop {
		t.Fatal("expected infra pain to pass")
	}
}
