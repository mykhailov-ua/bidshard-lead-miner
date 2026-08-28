package scoring

import "testing"

func TestDetectDisplacementTierHot(t *testing.T) {
	t.Parallel()
	text := "We run internal campaigns and I'm switching from voluum due to postback pain"
	tier, boost := DetectDisplacementTier(text, []string{"voluum"})
	if tier != DisplacementHot {
		t.Fatalf("tier=%q want hot", tier)
	}
	if boost != displacementHotBoost {
		t.Fatalf("boost=%d", boost)
	}
}

func TestDetectDisplacementTierWarm(t *testing.T) {
	t.Parallel()
	text := "We're looking at migrating to a new tracker next month"
	tier, boost := DetectDisplacementTier(text, nil)
	if tier != DisplacementWarm {
		t.Fatalf("tier=%q want warm", tier)
	}
	if boost != displacementWarmBoost {
		t.Fatalf("boost=%d", boost)
	}
}

func TestDetectDisplacementTierEditorialNone(t *testing.T) {
	t.Parallel()
	tier, boost := DetectDisplacementTier("companies migrating from voluum in 2025 industry report", []string{"voluum"})
	if tier != DisplacementNone || boost != 0 {
		t.Fatalf("tier=%q boost=%d want none", tier, boost)
	}
}

func TestScoreWithBoostsStructuredStack(t *testing.T) {
	t.Parallel()
	reg := NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	text := &LeadText{Context: "contact us at partnerships@example.com"}
	p := ScoreWithBoosts(reg, text, "tgweb:example.com", []string{"keitaro"}, nil, ScoreOpts{StructuredStack: true})
	if p == PriorityLow {
		t.Fatalf("expected medium+ from stack-only boost, score=%d priority=%s", text.Score, p)
	}
}
