package pipeline

import "testing"

func TestShouldSampleHardRejectShadow(t *testing.T) {
	t.Parallel()
	text := "casino affiliate program recruiting"
	if !ShouldSampleHardRejectShadow(text, 100) {
		t.Fatal("expected sample at 100%")
	}
	if ShouldSampleHardRejectShadow(text, 0) {
		t.Fatal("expected no sample at 0%")
	}
	// deterministic
	a := ShouldSampleHardRejectShadow(text, 50)
	b := ShouldSampleHardRejectShadow(text, 50)
	if a != b {
		t.Fatal("sample not deterministic")
	}
}
