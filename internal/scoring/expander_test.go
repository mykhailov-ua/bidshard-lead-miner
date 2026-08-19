package scoring

import (
	"testing"
)

func TestKeywordExpanderPermutations(t *testing.T) {
	exp := NewKeywordExpander()
	perms := exp.GeneratePermutations()
	if len(perms) == 0 {
		t.Fatal("expected non-empty permutations")
	}
	expectedMin := len(competitors) * len(painPhrases)
	if len(perms) != expectedMin {
		t.Fatalf("expected %d permutations, got %d", expectedMin, len(perms))
	}
}

func TestKeywordExpanderMatch(t *testing.T) {
	exp := NewKeywordExpander()
	sampleText := "Hello team, Voluum is too expensive for our media buying operations, looking for self-hosted option."
	matched, key := exp.MatchPermutation(sampleText)
	if !matched {
		t.Fatalf("expected match for sample text, got none")
	}
	if key != "voluum too expensive" && key != "voluum self-hosted" {
		t.Fatalf("unexpected matched key: %s", key)
	}
}

func BenchmarkKeywordExpanderMatch(b *testing.B) {
	exp := NewKeywordExpander()
	sampleText := "We are experiencing postback failing errors on RedTrack and considering a self-hosted tracker."

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exp.MatchPermutation(sampleText)
	}
}
