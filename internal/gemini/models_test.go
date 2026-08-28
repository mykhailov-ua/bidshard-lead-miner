package gemini

import "testing"

func TestDeprecatedModelWarning(t *testing.T) {
	if got := DeprecatedModelWarning("gemini-2.5-flash"); got == "" {
		t.Fatal("expected warning for 2.5-flash")
	}
	if got := DeprecatedModelWarning(DefaultModel); got != "" {
		t.Fatalf("unexpected warning for default: %q", got)
	}
}

func TestDefaultModelConstant(t *testing.T) {
	if DefaultModel != "gemini-3.6-flash" {
		t.Fatalf("DefaultModel=%q", DefaultModel)
	}
}
