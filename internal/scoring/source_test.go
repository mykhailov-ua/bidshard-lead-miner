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
