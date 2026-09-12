package classify

import "testing"

func TestClassifyBidShardPain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text   string
		bucket string
		tier   string
	}{
		{"Adspect too expensive for FB cloak stack", "cloak_stack_cost", "pro"},
		{"PP shaves leads, how to prove discrepancy", "shave_discrepancy", "pro"},
		{"keitaro on 200k clicks/day hangs server, admin 2 min load", "infra_scale", "network"},
		{"competitors bot link flood on tracking domain", "abuse_ddos", "enterprise"},
	}
	for _, tc := range cases {
		got := ClassifyBidShardPain(tc.text)
		if got == nil {
			t.Fatalf("nil for %q", tc.text)
		}
		if got.PainBucket != tc.bucket {
			t.Fatalf("bucket %q want %q for %q", got.PainBucket, tc.bucket, tc.text)
		}
		if got.TierHint != tc.tier {
			t.Fatalf("tier %q want %q for %q", got.TierHint, tc.tier, tc.text)
		}
	}
	if ClassifyBidShardPain("good morning team") != nil {
		t.Fatal("expected nil for noise")
	}
}
