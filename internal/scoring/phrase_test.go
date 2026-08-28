package scoring

import "testing"

func TestPhraseMatchesKeitaroNotKeitaroInc(t *testing.T) {
	t.Parallel()
	if PhraseMatches("keitaroinc/docker-ckan install", "keitaro") {
		t.Fatal("keitaro should not match inside keitaroinc")
	}
	if !PhraseMatches("switching from keitaro to binom", "keitaro") {
		t.Fatal("expected bounded keitaro match")
	}
}

func TestApplySpendGateKeitaroIncNoBypass(t *testing.T) {
	t.Parallel()
	score := ApplySpendGate(25, "github.com/keitaroinc/docker-ckan ckan helm install", 15)
	if score > 14 {
		t.Fatalf("score=%d want capped without competitor proof", score)
	}
}
