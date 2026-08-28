package store

import "testing"

func TestNormalizeOutcome(t *testing.T) {
	t.Parallel()

	got, err := NormalizeOutcome("Pilot_Started")
	if err != nil || got != OutcomePilotStarted {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := NormalizeOutcome("won"); err == nil {
		t.Fatal("expected error for status-like value")
	}
}

func TestListFilterOutcome(t *testing.T) {
	t.Parallel()

	q := ListFilter{Outcome: "pilot_started"}.matchQuery()
	if q["outcome"] != OutcomePilotStarted {
		t.Fatalf("outcome=%v", q["outcome"])
	}
}
