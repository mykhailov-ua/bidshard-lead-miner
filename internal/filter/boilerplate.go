package filter

import (
	"strings"
	"unicode"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/validate"
)

var htmlBoilerplateSignals = []string{
	"viewport width=device-width",
	"meta charset",
	"x-ua-compatible",
	"script nonce",
	"hreflang=",
	"data-theme=",
	"data-head-attrs",
	"content-type text/html",
	"initial-scale=1",
	"theme-color",
	"font-face",
	"async></script",
	"<!--",
}

// RejectHTMLBoilerplate drops tgweb crawl text that is mostly HTML head/meta noise.
func RejectHTMLBoilerplate(text string) (bool, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, ""
	}
	// Pages with real outreach contacts and buyer/affiliate context are not boilerplate.
	contacts := extract.Extract(text)
	contacts.Contacts = extract.FilterJunkContacts(contacts.Contacts)
	if extract.HasReachableContact(contacts.Contacts) && hasOutreachContext(text) {
		return false, ""
	}
	lower := strings.ToLower(text)
	hits := 0
	for _, sig := range htmlBoilerplateSignals {
		if strings.Contains(lower, sig) {
			hits++
		}
	}
	if hits >= 2 {
		return true, "html boilerplate"
	}
	if strings.Count(text, "<") >= 2 && strings.Count(text, ">") >= 2 && naturalWordCount(text) < 10 {
		return true, "html markup noise"
	}
	return false, ""
}

func hasOutreachContext(text string) bool {
	return scoring.HasBuyerIntentSignal(text) || validate.HasCommercialPainIntent(text) || validate.HasBuyerQuestionPattern(text) || validate.HasAffiliateNetworkContext(text)
}

func naturalWordCount(text string) int {
	fields := strings.Fields(text)
	n := 0
	for _, f := range fields {
		letters := 0
		for _, r := range f {
			if unicode.IsLetter(r) {
				letters++
			}
		}
		if letters >= 3 {
			n++
		}
	}
	return n
}
