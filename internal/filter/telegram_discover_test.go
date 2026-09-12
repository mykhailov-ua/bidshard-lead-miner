package filter

import "testing"

func TestTelegramDiscoverReject(t *testing.T) {
	t.Parallel()
	cases := []struct {
		username string
		texts    []string
		reject   bool
		reason   string
	}{
		{"igaming_news", nil, true, "block_handle"},
		{"partnerkin_job", nil, true, "block_handle"},
		{"daily_affiliate_news", nil, true, "block_username"},
		{"cpa_jobs_board", nil, true, "block_username"},
		{"soltrending", nil, true, "block_handle"},
		{"pumpspy_alerts", nil, true, "block_username"},
		{"randomchannel", nil, true, "intel_only"},
		{"voluum", nil, false, ""},
		{"buyermedia", []string{"media buying community"}, false, ""},
		{"vip_signals", []string{"VIP signal course mentorship paid tips only"}, true, "spam_channel"},
		{"keitaro_chat", []string{"affiliate tracker discussion"}, false, ""},
		{"", []string{"weekly news digest"}, true, "intel_only"},
	}
	for _, tc := range cases {
		got, reason := TelegramDiscoverReject(tc.username, tc.texts...)
		if got != tc.reject {
			t.Errorf("TelegramDiscoverReject(%q, %v) reject=%v want %v reason=%q", tc.username, tc.texts, got, tc.reject, reason)
			continue
		}
		if tc.reject && tc.reason != "" && reason != tc.reason {
			t.Errorf("TelegramDiscoverReject(%q) reason=%q want %q", tc.username, reason, tc.reason)
		}
	}
}

func TestTelegramIntelOnlyChannelBlockPatternsOnly(t *testing.T) {
	t.Parallel()
	if !TelegramIntelOnlyChannel("telegram:@igaming_news") {
		t.Fatal("expected igaming_news blocked")
	}
	if !TelegramIntelOnlyChannel("telegram:@cpa_jobs_board") {
		t.Fatal("expected job board channel blocked")
	}
	if TelegramIntelOnlyChannel("telegram:@voluum") {
		t.Fatal("expected voluum to pass")
	}
	if TelegramIntelOnlyChannel("telegram:@affnet") {
		t.Fatal("expected affnet to pass (no username block pattern)")
	}
}
