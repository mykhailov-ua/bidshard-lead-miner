package filter

import "testing"

func TestRejectProgrammaticContext(t *testing.T) {
	drop, reason := RejectProgrammaticContext("forum:test", "Programmatic guaranteed deal for CPM buyers", "")
	if !drop || reason != "programmatic vertical" {
		t.Fatalf("drop=%v reason=%q", drop, reason)
	}
	drop, _ = RejectProgrammaticContext("forum:test", "Head of programmatic looking at DSP stack", "")
	if !drop {
		t.Fatal("expected head of programmatic without pain to drop")
	}
	drop, _ = RejectProgrammaticContext("forum:test", "OpenRTB postback failing after voluum migration", "")
	if drop {
		t.Fatal("expected performance pain bypass")
	}
	drop, _ = RejectProgrammaticContext("forum:test", "We run FB igaming, postback not working on keitaro", "")
	if drop {
		t.Fatal("expected media buy postback bypass")
	}
	drop, _ = RejectProgrammaticContext("forum:test", "Voluum alternative for media buying team", "")
	if drop {
		t.Fatal("expected no programmatic marker")
	}
}

func TestRejectNonBuyerContextProgrammatic(t *testing.T) {
	drop, reason := RejectNonBuyerContext("serp:example.com", "Billboard brand awareness CPM campaign", "")
	if !drop || reason != "programmatic vertical" {
		t.Fatalf("drop=%v reason=%q", drop, reason)
	}
}
