package validate

import "strings"

// Crypto-gray buyer signals (docs/ICP.md M11): infra/tracker AND (payout OR antifraud pain).

var cryptoPayoutHints = []string{
	"trc20", "erc20", "usdt", "crypto payout", "crypto settlement",
	"weekly usdt", "daily usdt", "no kyc", "capitalist",
}

var infraStackHints = []string{
	"keitaro", "binom", "zeustrack", "hideclick", "cloaking.house", "cloaking house",
	"fraudfilter", "js fingerprint", "maxmind", "ipqualityscore", "ipqs",
	"voluum", "redtrack", "clickflare", "bemob",
}

var antifraudPainHints = []string{
	"shaving", "scrubbing", "bot click", "fake lead", "fake leads",
	"auto-fill", "autofill", "trash deposit", "cr drop", "incentivized traffic",
	"fraudulent traffic", "balance frozen", "network froze", "network frozen",
}

var trackerPainHints = []string{
	"voluum", "keitaro", "binom", "redtrack", "postback", "tracker", "clickid", "cloak", "s2s",
}

// HasCryptoGrayBuyerSignal reports M11 crypto-gray ICP in message body.
func HasCryptoGrayBuyerSignal(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	infra := containsAny(lower, infraStackHints) || containsAny(lower, trackerPainHints)
	if !infra {
		return false
	}
	return containsAny(lower, cryptoPayoutHints) || containsAny(lower, antifraudPainHints)
}

func containsAny(lower string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

// CryptoGrayScoreBoost adds prescan weight when M11 gate matches.
const CryptoGrayScoreBoost = 18

var operationalPainHints = []string{
	"failing", "failed", "fail", "crash", "crashed", "timeout", "timed out",
	"migration", "migrate", "broken", "not working", "doesn't work", "does not work",
	"502", "503", "504", "upstream", "discrepancy", "mismatch", "nginx",
}

// HasTrackerPainMessage mirrors Python message_has_tracker_pain (M3/M8 telegram gate).
func HasTrackerPainMessage(text string) bool {
	if HasCryptoGrayBuyerSignal(text) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if !containsAny(lower, trackerPainHints) {
		return false
	}
	return containsAny(lower, operationalPainHints) || HasCommercialPainIntent(text)
}
