package geo

import (
	"regexp"
	"strings"
)

// H1 reject reason codes (SHARDING_SESSION H1).
const (
	RejectCISGreyMarket = "cis_grey_market"
	RejectGeoBlockRUBY  = "geo_block_ru_by"
)

var (
	cisBookmakerRe = regexp.MustCompile(
		`(?i)(?:\b1win\b|\b1xbet\b|\b1хбет\b|\bmelbet\b|\bmostbet\b|\bмостбет\b|pin-?up(?:\.ru)?|\bvavada\b|\bвавада\b)`,
	)
	cpaRipRe         = regexp.MustCompile(`(?i)cpa\.rip`)
	ruSetupContextRe = regexp.MustCompile(
		`(?i)(?:\b(?:ru|rf|рф|рб)\b|россия|russia|рубл|сбер|тинькофф|\.ru\b|\+7)`,
	)
	ruCountryMarkerRe = regexp.MustCompile(`(?i)(?:\bрф\b|\bрб\b|\bRF\b|\bRU\b)`)
	ruRubCardRe       = regexp.MustCompile(`(?i)(?:рублевые?\s+карт|карт[аы]\s+мир|оплата\s+руб)`)
	ruDropRe          = regexp.MustCompile(`(?i)(?:дропы?\s+(?:рф|росси)|рф\s+дроп)`)
	ruFlagEmojiRe     = regexp.MustCompile(`\xF0\x9F\x87\xB7\xF0\x9F\x87\xBA|🇷🇺`)
	ruHandleRe        = regexp.MustCompile(`(?i)(?:^|@)[a-z0-9_][a-z0-9_.]*\.ru\b`)
	wwBuyerVoiceRe    = regexp.MustCompile(
		`(?i)(?:voluum|redtrack|binom|keitaro|self-?hosted|postback|usdt|trc20|erc20|` +
			`clickid|click\s+loss|alternative|migrat|502|nginx|mysql|clickhouse|tracker)`,
	)
)

// FilterH1 is the processor-first gate: CIS grey bookmakers + RU/BY geo/metadata (H1).
func FilterH1(text, username, channelAbout string, contacts ...string) Result {
	body := buildH1Body(text, username, channelAbout, contacts...)
	if body == "" {
		return Result{OK: true}
	}

	if drop, detail := rejectCISGreyMarket(body); drop {
		return Result{Reason: RejectCISGreyMarket + ": " + detail}
	}

	merged := mergeH1Text(text, username, channelAbout)
	res := filterGeoCore(merged, contacts...)
	if !res.OK {
		if hasUAAffinity(body) && isUALocationOnlyReject(res.Reason) {
			// UA teams may mention CIS geography in passing; infra/payment still rejects above.
		} else {
			return Result{Reason: RejectGeoBlockRUBY + ": " + res.Reason}
		}
	}

	if detail := rejectH1Metadata(username, channelAbout, body); detail != "" {
		return Result{Reason: RejectGeoBlockRUBY + ": " + detail}
	}
	if detail := rejectH1GeoMarkers(body); detail != "" {
		return Result{Reason: RejectGeoBlockRUBY + ": " + detail}
	}

	return Result{OK: true}
}

func buildH1Body(text, username, channelAbout string, contacts ...string) string {
	parts := []string{text, username, channelAbout}
	parts = append(parts, contacts...)
	return strings.Join(parts, "\n")
}

func mergeH1Text(text, username, channelAbout string) string {
	return strings.TrimSpace(strings.Join([]string{text, username, channelAbout}, "\n"))
}

func filterGeoCore(text string, contacts ...string) Result {
	return Filter(text, contacts...)
}

func hasUAAffinity(body string) bool {
	return uaAffinityRe.MatchString(body)
}

func isUALocationOnlyReject(reason string) bool {
	return reason == "ru/by location"
}

func rejectCISGreyMarket(body string) (bool, string) {
	lower := strings.ToLower(body)
	if cisBookmakerRe.MatchString(lower) {
		if hasWWBuyerVoice(lower) {
			return false, ""
		}
		return true, "cis bookmaker"
	}
	if cpaRipRe.MatchString(lower) && ruSetupContextRe.MatchString(lower) {
		if hasWWBuyerVoice(lower) {
			return false, ""
		}
		return true, "cpa.rip ru setup"
	}
	return false, ""
}

func rejectH1GeoMarkers(body string) string {
	lower := strings.ToLower(body)
	if ruRubCardRe.MatchString(lower) {
		return "rub card"
	}
	if ruDropRe.MatchString(lower) {
		return "ru drop"
	}
	if ruCountryMarkerRe.MatchString(body) && !hasWWBuyerVoice(lower) {
		return "ru/by country marker"
	}
	return ""
}

func rejectH1Metadata(username, channelAbout, body string) string {
	meta := strings.TrimSpace(username + "\n" + channelAbout)
	if meta == "" {
		meta = body
	}
	if ruFlagEmojiRe.MatchString(meta) {
		return "ru flag in profile"
	}
	if ruHandleRe.MatchString(meta) {
		return "ru handle"
	}
	return ""
}

func hasWWBuyerVoice(lower string) bool {
	return wwBuyerVoiceRe.MatchString(lower)
}
