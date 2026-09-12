package filter

import "strings"

// Keep username block lists aligned with sources/telegram/prefilter.py channel_discover_reject.

var telegramDiscoverBlockHandles = map[string]struct{}{
	"igaming_news":             {},
	"partnerkin_job":           {},
	"affiliatechannel_igaming": {},
	"partneroff_pro":           {},
	"soltrending":              {},
	"pumpspy":                  {},
	"pumpalert":                {},
	"cryptopumpsignals":        {},
	"moonshotgems":             {},
}

var telegramDiscoverBlockSubstrings = []string{
	"_news", "_jobs", "_job_", "jobboard", "job_board", "vacancy", "partnerkin",
	"soltrending", "pumpspy", "pumpalert", "cryptopump", "moonshot", "gemsalert",
	"migrationspam", "migrate2us", "usdtairdrop",
}

var telegramDiscoverSpamHints = []string{
	"signal", "signals", "course", "mentorship", "vip group", "paid tips", "casino tips",
}

var telegramDiscoverPositiveHints = []string{
	"affiliate", "igaming", "media buy", "mediabuy", "media_buy", "mediabuying",
	"arbitrage", "tracker", "keitaro", "binom", "voluum", "redtrack", "clickflare",
	"cpa", "acquisition", "performance marketing", "buyer", "postback", "s2s",
	"clickid", "ftd", "cloak",
}

// TelegramDiscoverReject reports whether a discovered channel should be dropped before registry/chats.
// Username and optional title/query/snippet texts are scanned for block patterns and buyer signals.
func TelegramDiscoverReject(username string, texts ...string) (reject bool, reason string) {
	if reject, reason := RejectH2CISPool(username, texts...); reject {
		return true, reason
	}
	user := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
	parts := make([]string, 0, 1+len(texts))
	if user != "" {
		parts = append(parts, user)
	}
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t != "" {
			parts = append(parts, strings.ToLower(t))
		}
	}
	blob := strings.Join(parts, " ")
	if blob == "" {
		return true, "empty"
	}

	if user != "" {
		if _, ok := telegramDiscoverBlockHandles[user]; ok {
			return true, "block_handle"
		}
		for _, sub := range telegramDiscoverBlockSubstrings {
			if strings.Contains(user, sub) {
				return true, "block_username"
			}
		}
	}

	spamHits := 0
	posHits := 0
	for _, h := range telegramDiscoverSpamHints {
		if strings.Contains(blob, h) {
			spamHits++
		}
	}
	for _, h := range telegramDiscoverPositiveHints {
		if strings.Contains(blob, h) {
			posHits++
		}
	}
	if spamHits >= 2 && posHits == 0 {
		return true, "spam_channel"
	}
	if posHits == 0 {
		return true, "intel_only"
	}
	return false, ""
}
