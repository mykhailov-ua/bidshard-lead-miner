package discover

import (
	"testing"
	"time"
)

func TestBlockedDiscoverQueryDenylist(t *testing.T) {
	t.Parallel()

	cases := []struct {
		q      string
		block  bool
		reason string
	}{
		{"site:linkedin.com affiliate", true, "linkedin.com"},
		{"site:t.me voluum alternative", false, ""},
		{"binom migration pain", false, ""},
		{"ab", true, "too_short"},
	}
	for _, tc := range cases {
		got, reason := BlockedDiscoverQuery(tc.q)
		if got != tc.block {
			t.Fatalf("BlockedDiscoverQuery(%q)=%v want %v", tc.q, got, tc.block)
		}
		if tc.block && reason == "" {
			t.Fatalf("expected reason for %q", tc.q)
		}
	}
}

func TestFilterDiscoverAdditionsBlocksJunk(t *testing.T) {
	t.Parallel()

	keepTG, keepSerp, blocked := FilterDiscoverAdditions(
		[]string{"binom migration"},
		[]string{"site:t.me binom", "site:linkedin.com jobs"},
	)
	if len(keepTG) != 1 || keepTG[0] != "binom migration" {
		t.Fatalf("keepTG=%v", keepTG)
	}
	if len(keepSerp) != 1 || keepSerp[0] != "site:t.me binom" {
		t.Fatalf("keepSerp=%v", keepSerp)
	}
	if blocked["site:linkedin.com jobs"] != "linkedin.com" {
		t.Fatalf("blocked=%v", blocked)
	}
}

func TestTrimDiscoverQuota(t *testing.T) {
	t.Parallel()

	tg, sd, deferred := TrimDiscoverQuota([]string{"a", "b"}, []string{"c"}, 2)
	if len(tg) != 2 || len(sd) != 0 || len(deferred) != 1 || deferred[0] != "c" {
		t.Fatalf("tg=%v sd=%v deferred=%v", tg, sd, deferred)
	}
}

func TestAutoApplyStateResetsEachWeek(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC) // Saturday
	prevWeek := AutoApplyState{
		WeekStart: weekStartUTC(now.AddDate(0, 0, -7)).Format(time.RFC3339),
		Applied:   25,
	}
	if rem := RemainingWeeklyQuota(prevWeek, now, 30); rem != 30 {
		t.Fatalf("remaining=%d want fresh week 30", rem)
	}
}
