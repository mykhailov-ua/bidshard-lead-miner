package filter

import "testing"

func TestSourceMixTier(t *testing.T) {
	t.Parallel()
	if SourceMixTier("telegram:@aff") != SourceMixHot {
		t.Fatal("telegram should be hot")
	}
	if SourceMixTier("forum:afflift") != SourceMixHot {
		t.Fatal("forum should be hot")
	}
	if SourceMixTier("reddit:r/affiliatemarketing") != SourceMixDeprioritize {
		t.Fatal("reddit should be deprioritize")
	}
}

func TestTGFirstSources(t *testing.T) {
	t.Parallel()
	if TGFirstSources() == "" {
		t.Fatal("expected non-empty profile")
	}
}
