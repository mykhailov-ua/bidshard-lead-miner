package filter

import (
	"regexp"
	"strings"
)

var hostingIncidentRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b502\b`),
	regexp.MustCompile(`(?i)\b503\b`),
	regexp.MustCompile(`(?i)\b504\b`),
	regexp.MustCompile(`(?i)\bupstream\b`),
	regexp.MustCompile(`(?i)\babuse\s+suspend`),
	regexp.MustCompile(`(?i)\bhost(?:er|ing)\b.{0,40}\b(?:block|suspend|down)\b`),
	regexp.MustCompile(`(?i)\bredirect\b.{0,30}\bdown\b`),
	regexp.MustCompile(`(?i)\bnginx\b.{0,40}\b(?:timeout|error|down)\b`),
}

var hostingVendorHints = []string{
	"alexhost", "pq.hosting", "pq hosting", "zomro", "friendhosting", "bulletproof",
	"offshore hosting", "abuse ticket",
}

var hostingTrackerHints = []string{
	"keitaro", "binom", "voluum", "redtrack", "tracker", "postback", "clickid",
}

// HasHostingIncidentPain reports bulletproof hosting incident + tracker stack (H4).
func HasHostingIncidentPain(text string) bool {
	body := strings.TrimSpace(text)
	if body == "" {
		return false
	}
	lower := strings.ToLower(body)
	hasIncident := false
	for _, rx := range hostingIncidentRe {
		if rx.MatchString(body) {
			hasIncident = true
			break
		}
	}
	if !hasIncident {
		return false
	}
	for _, hint := range hostingTrackerHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// IsHostingChannelHint reports discover hints for hosting vendor communities.
func IsHostingChannelHint(username, title, query string) bool {
	combined := strings.ToLower(strings.TrimSpace(username + " " + title + " " + query))
	if combined == "" {
		return false
	}
	for _, hint := range hostingVendorHints {
		if strings.Contains(combined, hint) {
			return true
		}
	}
	return false
}
