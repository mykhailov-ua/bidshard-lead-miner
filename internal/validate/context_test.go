package validate

import "testing"

func TestHasPainContext(t *testing.T) {
	t.Parallel()

	if !HasPainContext("voluum alternative postback failing ops@team.com") {
		t.Fatal("expected pain context")
	}
	if HasPainContext("ops@example.com") {
		t.Fatal("expected no pain context for email only")
	}
	if !HasPainContext("ops@team.com " + stringsRepeat("looking for advice ", 8)) {
		t.Fatal("expected long prose to count as context")
	}
}

func TestEmailWithoutPainContext(t *testing.T) {
	t.Parallel()

	if !EmailWithoutPainContext("contact me ops@example.com") {
		t.Fatal("expected email-only reject")
	}
	if EmailWithoutPainContext("voluum bill too high ops@example.com") {
		t.Fatal("expected pass with pain keyword")
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
