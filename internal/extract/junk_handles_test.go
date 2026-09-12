package extract

import "testing"

func TestIsJunkTelegramHandle(t *testing.T) {
	t.Parallel()
	for _, h := range []string{"@media", "keyframes", "@supports"} {
		if !IsJunkTelegramHandle(h) {
			t.Fatalf("expected junk %q", h)
		}
	}
	if IsJunkTelegramHandle("@media_buyer") {
		t.Fatal("expected real handle")
	}
}

func TestFilterJunkContactsDropsCSS(t *testing.T) {
	t.Parallel()
	in := []Contact{
		{Type: "telegram", Value: "@media"},
		{Type: "email", Value: "ops@example.com"},
	}
	out := FilterJunkContacts(in)
	if len(out) != 1 || out[0].Type != "email" {
		t.Fatalf("out=%v", out)
	}
}

func TestIsFalseTelegramHandleSerpDomain(t *testing.T) {
	t.Parallel()
	if !IsFalseTelegramHandle("@serp:www.redtrack.io") {
		t.Fatal("expected domain-shaped serp handle to be false")
	}
	if !IsSyntheticContact("telegram:@serp:www.redtrack.io") {
		t.Fatal("expected synthetic serp contact")
	}
}

func TestIsFalseTelegramHandle(t *testing.T) {
	t.Parallel()
	if !IsFalseTelegramHandle("@github") {
		t.Fatal("expected false positive handle")
	}
}
