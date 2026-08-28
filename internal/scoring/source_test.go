package scoring

import "testing"

func TestStaticSourceBoost(t *testing.T) {
	t.Parallel()
	if got := StaticSourceBoost("forum:stm.example/thread"); got != 10 {
		t.Fatalf("boost=%d", got)
	}
	if got := StaticSourceBoost("ads_txt:example.com"); got != 8 {
		t.Fatalf("boost=%d", got)
	}
	if got := StaticSourceBoost("telegram:invite:abc"); got >= 0 {
		t.Fatalf("invite boost=%d want negative", got)
	}
	if got := StaticSourceBoost("reddit:r/affiliatemarketing"); got != 5 {
		t.Fatalf("reddit boost=%d", got)
	}
}

func TestNormalizeSourceKeyInvite(t *testing.T) {
	t.Parallel()
	if got := normalizeSourceKey("telegram:invite:abc"); got != "telegram:invite" {
		t.Fatalf("key=%q", got)
	}
}

func TestSourceReputationDynamic(t *testing.T) {
	t.Parallel()
	rep := NewSourceReputation(nil)
	for i := 0; i < 25; i++ {
		rep.RecordAccepted("telegram:test")
	}
	boost := rep.Boost("telegram:test")
	if boost < 5 {
		t.Fatalf("boost=%d want >=5 with good ratio", boost)
	}
}
