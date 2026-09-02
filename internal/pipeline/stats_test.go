package pipeline

import (
	"testing"
)

func TestTopRejectReasons(t *testing.T) {
	t.Parallel()

	stats := RoundStats{
		RejectedGeo:        12,
		RejectedLang:       5,
		RejectedContext:    8,
		RejectedBlacklist:  3,
		RejectedLowPriority: 1,
	}
	got := TopRejectReasons(stats, 3)
	want := []string{"geo:12", "context:8", "lang:5"}
	if len(got) != len(want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}

func TestRecordRoundRejectMapsAllBuckets(t *testing.T) {
	t.Parallel()

	var state RoundState
	cases := []struct {
		reason string
		check  func(RoundStats) int
	}{
		{"geo", func(s RoundStats) int { return s.RejectedGeo }},
		{"lang", func(s RoundStats) int { return s.RejectedLang }},
		{"context", func(s RoundStats) int { return s.RejectedContext }},
		{"contact", func(s RoundStats) int { return s.RejectedContact }},
		{"no_contacts", func(s RoundStats) int { return s.RejectedNoContacts }},
		{"email_no_context", func(s RoundStats) int { return s.RejectedEmailNoContext }},
		{"role_email", func(s RoundStats) int { return s.RejectedRoleEmail }},
		{"empty_hash", func(s RoundStats) int { return s.RejectedEmptyHash }},
		{"mx", func(s RoundStats) int { return s.RejectedMX }},
		{"dedup", func(s RoundStats) int { return s.Dedup }},
		{"hard_reject", func(s RoundStats) int { return s.HardRejected }},
		{"blacklist", func(s RoundStats) int { return s.RejectedBlacklist }},
		{"intel_only", func(s RoundStats) int { return s.RejectedIntelOnly }},
		{"lander_no_buyer_signal", func(s RoundStats) int { return s.RejectedLanderNoBuyer }},
		{"github_vendor", func(s RoundStats) int { return s.RejectedGitHubVendor }},
		{"telegram_spam", func(s RoundStats) int { return s.RejectedTelegramSpam }},
		{"low_priority", func(s RoundStats) int { return s.RejectedLowPriority }},
		{"icp", func(s RoundStats) int { return s.RejectedICP }},
		{"intent", func(s RoundStats) int { return s.RejectedIntent }},
	}
	for _, tc := range cases {
		recordRoundReject(&state, tc.reason)
	}
	recordRoundReject(&state, "unknown_reason")

	stats := state.Snapshot("round", 0)
	for _, tc := range cases {
		if got := tc.check(stats); got != 1 {
			t.Fatalf("reason=%s count=%d want 1", tc.reason, got)
		}
	}
	if stats.Dropped != 1 {
		t.Fatalf("dropped=%d want 1 for unknown reason", stats.Dropped)
	}
	if len(TopRejectReasons(stats, 0)) != len(cases)+1 {
		t.Fatalf("top reject count=%d want %d", len(TopRejectReasons(stats, 0)), len(cases)+1)
	}
}
