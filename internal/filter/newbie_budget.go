package filter

import (
	"strings"

	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/validate"
)

var newbieNoBudgetPhrases = []string{
	"can't afford", "cant afford", "cannot afford",
	"want to start my journey", "start my journey",
	"want to start promoting", "starting my journey",
	"want to start my", "just starting out",
	"first cpa offer", "first offer",
	"good and cheap", "cheap tracker",
	"i'm a beginner", "i am a beginner", "total beginner",
	"no budget", "zero budget", "no money for",
}

// RejectNewbieNoBudget drops broke beginner intent without spend or infra pain signals.
func RejectNewbieNoBudget(text, title string) (bool, string) {
	combined := strings.ToLower(strings.TrimSpace(title + " " + text))
	if combined == "" {
		return false, ""
	}
	if scoring.HasSpendSignal(combined) {
		return false, ""
	}
	if hasInfraPainBypass(combined) {
		return false, ""
	}
	for _, phrase := range newbieNoBudgetPhrases {
		if strings.Contains(combined, phrase) {
			return true, "newbie no budget"
		}
	}
	return false, ""
}

func hasInfraPainBypass(lower string) bool {
	if validate.HasCommercialPainIntent(lower) {
		for _, hint := range []string{
			"postback", "not working", "failing", "broken", "oom", "out of memory",
			"crash", "migrate", "switching from", "clicks per day", "million clicks",
		} {
			if strings.Contains(lower, hint) {
				return true
			}
		}
	}
	return false
}
