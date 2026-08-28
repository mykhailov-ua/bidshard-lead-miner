package filter

import (
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/validate"
)

// Programmatic / supply-side / brand OOH markers (not performance-buyer ICP v1).
var programmaticMarkerRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bprogrammatic\b`),
	regexp.MustCompile(`(?i)\bopenrtb\b`),
	regexp.MustCompile(`(?i)\bopen\s+rtb\b`),
	regexp.MustCompile(`(?i)\bheader\s+bidding\b`),
	regexp.MustCompile(`(?i)\bprebid\s+server\b`),
	regexp.MustCompile(`(?i)\bsupply-side\b`),
	regexp.MustCompile(`(?i)\bsupply\s+path\s+optimization\b`),
	regexp.MustCompile(`(?i)\bprogrammatic\s+guaranteed\b`),
	regexp.MustCompile(`(?i)\brtb\s+exchange\b`),
	regexp.MustCompile(`(?i)\breal-time\s+bidding\b`),
	regexp.MustCompile(`(?i)\bwhite-label\s+ssp\b`),
	regexp.MustCompile(`(?i)\bwhite\s+label\s+exchange\b`),
	regexp.MustCompile(`(?i)\bpublishers?\s+monetization\b`),
	regexp.MustCompile(`(?i)\byield\s+optimization\b`),
	regexp.MustCompile(`(?i)\bad\s+server\s+for\s+publishers\b`),
	regexp.MustCompile(`(?i)\bdooh\b`),
	regexp.MustCompile(`(?i)\bpdooh\b`),
	regexp.MustCompile(`(?i)\bout\s+of\s+home\b`),
	regexp.MustCompile(`(?i)\bbillboard\b`),
	regexp.MustCompile(`(?i)\bbrand\s+awareness\b`),
	regexp.MustCompile(`(?i)\bviewability\b`),
	regexp.MustCompile(`(?i)\bbrand\s+lift\b`),
	regexp.MustCompile(`(?i)\bcpm\s+campaign\b`),
	regexp.MustCompile(`(?i)\bhead\s+of\s+programmatic\b`),
	regexp.MustCompile(`(?i)\bopenrtb\s+bidder\b`),
	regexp.MustCompile(`(?i)\bdsp\s+bidder\b`),
	regexp.MustCompile(`(?i)\bssp\s+integration\b`),
	regexp.MustCompile(`(?i)\bprogrammatic\s+stack\b`),
}

var programmaticTokenRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:dsp|ssp)\b`),
}

func hasProgrammaticMarker(text string) bool {
	for _, re := range programmaticMarkerRe {
		if re.MatchString(text) {
			return true
		}
	}
	for _, re := range programmaticTokenRe {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

func performanceBuyerBypass(text string) bool {
	if HasCommercialPainIntent(text) || HasBuyerQuestionPattern(text) {
		return true
	}
	if validate.HasStrictPainContext(text) {
		return true
	}
	lower := strings.ToLower(text)
	// Tracker migration/postback pain with incidental programmatic mention.
	for _, hint := range []string{
		"postback", "voluum", "keitaro", "binom", "redtrack", "clickid",
		"media buy", "media buying", "fb traffic", "igaming",
	} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// RejectProgrammaticContext drops programmatic/supply/brand OOH vertical without performance-buyer pain.
func RejectProgrammaticContext(source, text, title string) (bool, string) {
	combined := strings.TrimSpace(title + " " + text)
	if combined == "" {
		return false, ""
	}
	if !hasProgrammaticMarker(combined) {
		return false, ""
	}
	if performanceBuyerBypass(combined) {
		return false, ""
	}
	return true, "programmatic vertical"
}
