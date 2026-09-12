package filter

import "strings"

// banContextPhrases: ad-platform ban / restriction signals. Never hard-reject; route to LLM.
var banContextPhrases = []string{
	"facebook ban", "fb ban", "meta ban", "ad account banned", "account disabled",
	"ads account restricted", "business manager disabled", "bm disabled",
	"account flagged", "ad account suspended", "разбан", "бан фб", "бан аккаунта",
	"штормит фб", "штормит facebook", "ограничение рекламы", "ads rejected",
	"campaign rejected", "ad rejected", "policy violation", "restricted ad account",
	"appeal rejected", "разблокировка аккаунта", "аккаунт слетел", "акки слетают",
	"pixel not firing", "pixel not seeing", "capi error", "conversions not arriving",
	"conversions not coming", "не доходят конверсии", "конверсии не доходят",
	"lead rejected", "postback timeout", "postback not received",
}

// HasBanContextSignal reports ban/restriction pain that needs LLM context, not keyword reject.
func HasBanContextSignal(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, phrase := range banContextPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// InfraPainBypassPrescan extends keyword prescan pass for runtime/infra pain markers.
func InfraPainBypassPrescan(text string) bool {
	if HasBanContextSignal(text) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, hint := range infraPainHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// infraPainHints: priority ingest markers (network, tracker, load).
var infraPainHints = []string{
	"502 bad gateway", "504 gateway timeout", "upstream timed out",
	"too many open files", "connection reset by peer", "ssl handshake failed",
	"keitaro упал", "keitaro down", "keitaro crashed", "binom завис", "binom stuck",
	"binom hangs", "clickhouse cpu", "mysql lock", "mysql slow",
	"redirect delay", "redirect latency", "duplicate clicks", "duplicate click",
	"click_id lost", "click id lost", "clickid lost", "click_id потерялся",
	"postback timeout", "capi error 400", "capi error", "lead rejected",
	"rps limit", "million hits", "traffic spike", "swap full", "swap забился",
	"oom kill", "out of memory", "hetzner abuse", "hetzner abusing",
	"штормит трафик", "пик трафика", "спайк трафика",
}
