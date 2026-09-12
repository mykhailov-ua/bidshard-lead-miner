package filter

import (
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/validate"
)

var (
	// Cyrillic terms omit \b: Go RE2 word boundaries are ASCII-only.
	h8FOPTOVRe = regexp.MustCompile(
		`(?i)(?:фоп|тов|гривн|grivna|\bfop\b|\btov\b|\buah\b|privatbank|monobank)`,
	)
	h8InfobizRe = regexp.MustCompile(
		`(?i)(?:инфобиз|infobiz|white\s+goods|курсы?\s+по\s+арбитраж|course\s+seller|` +
			`how\s+to\s+buy\s+domain|купить\s+домен)`,
	)
	h8CryptoPayoutRe = regexp.MustCompile(
		`(?i)(?:\busdt\b|\btrc20\b|\berc20\b|crypto\s+payout|crypto\s+settlement|weekly\s+usdt|daily\s+usdt)`,
	)
)

// RejectH8PaymentVertical drops local payment rails and off-vertical noise (H8/M11).
func RejectH8PaymentVertical(text, title string) (bool, string) {
	combined := strings.ToLower(strings.TrimSpace(title + " " + text))
	if combined == "" {
		return false, ""
	}
	if h8FOPTOVRe.MatchString(combined) && !hasH8BuyerVoice(combined) {
		return true, "h8 fop/tov/grivna"
	}
	if h8InfobizRe.MatchString(combined) && !validate.HasTrackerPainMessage(text) {
		return true, "h8 infobiz/off-vertical"
	}
	return false, ""
}

// RejectCryptoPayoutOnly rejects USDT/crypto mention without tracker/operational pain (H8).
func RejectCryptoPayoutOnly(text string) (bool, string) {
	body := strings.TrimSpace(text)
	if body == "" {
		return false, ""
	}
	lower := strings.ToLower(body)
	if !h8CryptoPayoutRe.MatchString(lower) {
		return false, ""
	}
	if validate.HasTrackerPainMessage(body) {
		return false, ""
	}
	if validate.HasCommercialPainIntent(body) || validate.HasBuyerQuestionPattern(body) {
		return false, ""
	}
	return true, "h8 crypto payout without pain"
}

func hasH8BuyerVoice(lower string) bool {
	for _, hint := range []string{
		"voluum", "keitaro", "binom", "redtrack", "postback", "tracker", "usdt",
		"trc20", "clickid", "self-hosted", "502", "nginx", "mysql",
	} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}
