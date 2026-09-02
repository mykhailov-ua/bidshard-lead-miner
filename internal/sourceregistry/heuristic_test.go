package sourceregistry

import "testing"

func TestHeuristicTriageDropsSocialHosts(t *testing.T) {
	t.Parallel()
	cases := []string{
		"facebook.com",
		"www.twitter.com",
		"t.me",
		"bit.ly",
		"netflix.com",
		"www.solscan.io",
		"gmgn.ai",
		"solanacoinpump.top",
		"t.co",
		"etherscan.io",
		"pump.fun",
	}
	for _, domain := range cases {
		action, why, ok := HeuristicTriage(DomainMeta{Domain: domain})
		if !ok || action != "drop" {
			t.Fatalf("domain=%s action=%s why=%s ok=%v want heuristic drop", domain, action, why, ok)
		}
	}
}

func TestHeuristicTriageKeepsAffiliateDomain(t *testing.T) {
	t.Parallel()
	action, _, ok := HeuristicTriage(DomainMeta{
		Domain:  "buylink.pro",
		Channel: "aff_net",
		Source:  "cross_mention",
	})
	if ok {
		t.Fatalf("expected no heuristic decision, got %s", action)
	}
}
