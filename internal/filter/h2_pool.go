package filter

import "strings"

// H2 WW/UA pool curation: drop RU/CIS arbitrage channels at discover ingress.
// Keep aligned with sources/telegram/h2_pool.py.

var h2CISArbitrageHandles = map[string]struct{}{
	"maximaffiliate":     {},
	"zuevaff":            {},
	"zuevchatcpa":        {},
	"traffic_trickster":  {},
	"bakalov_info":       {},
	"seodroplab":         {},
	"po_ushi_v_gambling": {},
	"frbs_team":          {},
	"g_gate_media":       {},
	"affwriter":          {},
	"m2ensenchannel":     {},
	"cpalenta":           {},
	"cpa_lenta":          {},
}

var h2CISArbitrageSubstrings = []string{
	"zuevaff",
	"maximaff",
	"cpalent",
	"cparip",
	"arbitrage_ru",
	"_ru_arb",
	"ru_arbitrage",
}

// RejectH2CISPool drops known CIS/RU arbitrage pool noise (SHARDING_SESSION H2).
func RejectH2CISPool(username string, texts ...string) (bool, string) {
	user := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
	if user != "" {
		if _, ok := h2CISArbitrageHandles[user]; ok {
			return true, "h2_cis_arbitrage"
		}
		for _, sub := range h2CISArbitrageSubstrings {
			if strings.Contains(user, sub) {
				return true, "h2_cis_username"
			}
		}
	}
	var parts []string
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t != "" {
			parts = append(parts, t)
		}
	}
	blob := strings.ToLower(strings.Join(parts, " "))
	if blob == "" {
		return false, ""
	}
	if strings.Contains(blob, "cpa.rip") && cisRUSetupContext(blob) && !hasWWBuyerVoiceH2(blob) {
		return true, "h2_cpa_rip_ru"
	}
	return false, ""
}

func cisRUSetupContext(lower string) bool {
	return strings.Contains(lower, "россия") ||
		strings.Contains(lower, "russia") ||
		strings.Contains(lower, " ru ") ||
		strings.Contains(lower, " рф ") ||
		strings.Contains(lower, ".ru") ||
		strings.Contains(lower, "+7") ||
		strings.Contains(lower, "сбер") ||
		strings.Contains(lower, "рубл")
}

func hasWWBuyerVoiceH2(lower string) bool {
	hints := []string{
		"voluum", "redtrack", "binom", "keitaro", "self-hosted", "postback",
		"usdt", "trc20", "tracker", "alternative", "clickhouse",
	}
	for _, h := range hints {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}
