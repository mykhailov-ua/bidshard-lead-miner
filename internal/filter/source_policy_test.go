package filter

import "testing"

func TestCollectPriorityFamily(t *testing.T) {
	t.Parallel()
	if CollectPriorityFamily("reddit") <= CollectPriorityFamily("lander") {
		t.Fatal("reddit should outrank lander")
	}
	if CollectPriorityFamily("forum") <= CollectPriorityFamily("github") {
		t.Fatal("forum should outrank github")
	}
}

func TestIsIntelOnlySource(t *testing.T) {
	t.Parallel()
	if !IsIntelOnlySource("lander:voluum.com", false) {
		t.Fatal("lander default intel only")
	}
	if IsIntelOnlySource("lander:voluum.com", true) {
		t.Fatal("lander outreach opt-in")
	}
	if IsIntelOnlySource("reddit:r/test", false) {
		t.Fatal("reddit is outreach")
	}
	if !IsIntelOnlySource("telegram:@igaming_news", false) {
		t.Fatal("telegram news channel intel only")
	}
}

func TestSourceRequiresIntentGate(t *testing.T) {
	t.Parallel()
	if SourceRequiresIntentGate("reddit:r/affiliatemarketing", "") {
		t.Fatal("reddit should skip intent gate")
	}
	if !SourceRequiresIntentGate("github:org/repo", "") {
		t.Fatal("github should require intent gate")
	}
	if !SourceRequiresIntentGate("telegram:@news", "channel") {
		t.Fatal("telegram channel should require intent gate")
	}
	if SourceRequiresIntentGate("telegram:@buyers", "supergroup") {
		t.Fatal("telegram supergroup should skip intent gate")
	}
}
