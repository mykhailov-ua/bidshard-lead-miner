package gemini

import "testing"

func TestDeprecatedModelConfigError(t *testing.T) {
	for _, model := range []string{"gemini-2.5-flash", "gemini-2.0-flash", "gemini-2.5-flash-preview"} {
		if err := DeprecatedModelConfigError(model); err == nil {
			t.Fatalf("expected error for %s", model)
		}
	}
	if err := DeprecatedModelConfigError(DefaultModel); err != nil {
		t.Fatalf("unexpected error for default: %v", err)
	}
}

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
