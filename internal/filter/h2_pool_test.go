package filter

import "testing"

func TestRejectH2CISPool(t *testing.T) {
	t.Parallel()
	cases := []struct {
		user   string
		texts  []string
		reject bool
		reason string
	}{
		{"maximaffiliate", nil, true, "h2_cis_arbitrage"},
		{"zuevaff", nil, true, "h2_cis_arbitrage"},
		{"buyermedia", []string{"media buying WW"}, false, ""},
		{"affiliate_latam_en", []string{"LATAM igaming"}, false, ""},
		{"cpa_rip_chat", []string{"cpa.rip RU funnels 1win"}, true, "h2_cpa_rip_ru"},
		{"affhub_ua", []string{"cpa.rip voluum postback Kyiv"}, false, ""},
	}
	for _, tc := range cases {
		got, reason := RejectH2CISPool(tc.user, tc.texts...)
		if got != tc.reject {
			t.Errorf("RejectH2CISPool(%q) reject=%v want %v", tc.user, got, tc.reject)
			continue
		}
		if tc.reject && tc.reason != "" && reason != tc.reason {
			t.Errorf("RejectH2CISPool(%q) reason=%q want %q", tc.user, reason, tc.reason)
		}
	}
}
