package scoring

import "testing"

func TestPrescanFailsProgrammaticWithoutPain(t *testing.T) {
	reg := NewRegistry("../../data/keywords.json")
	if reg == nil {
		t.Fatal("registry nil")
	}
	if PrescanPasses("forum:test", reg, "Programmatic guaranteed CPM campaign for display buyers") {
		t.Fatal("expected prescan fail for programmatic vertical")
	}
}

func TestPrescanPassesOpenRTBWithPostbackPain(t *testing.T) {
	reg := NewRegistry("../../data/keywords.json")
	text := "OpenRTB postback failing after voluum migration on igaming traffic"
	if !PrescanPasses("forum:test", reg, text) {
		t.Fatal("expected prescan pass with tracker pain bypass")
	}
}
