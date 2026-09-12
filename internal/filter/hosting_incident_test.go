package filter

import "testing"

func TestHasHostingIncidentPain(t *testing.T) {
	t.Parallel()
	if !HasHostingIncidentPain("AlexHost 502 on keitaro redirect upstream timed out") {
		t.Fatal("expected hosting incident pain")
	}
	if HasHostingIncidentPain("when will server be up again?") {
		t.Fatal("generic hosting question should not pass without tracker")
	}
}

func TestIsHostingChannelHint(t *testing.T) {
	t.Parallel()
	if !IsHostingChannelHint("alexhost_chat", "AlexHost support", "") {
		t.Fatal("expected hosting channel hint")
	}
}
