package validate

import (
	"regexp"
	"strings"
)

// H3/H4 segment pain mirrors internal/filter/pwa_pain.go and hosting_incident.go.

var pwaPainRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bpostback\s+from\s+app\b`),
	regexp.MustCompile(`(?i)\bwebview\s+stuck\b`),
	regexp.MustCompile(`(?i)\bsub[_-]?id\b.{0,40}\b(?:keitaro|binom|voluum|tracker)\b`),
	regexp.MustCompile(`(?i)\b(?:keitaro|binom|voluum|tracker)\b.{0,40}\bsub[_-]?id\b`),
	regexp.MustCompile(`(?i)\bredirect\s+delay\b`),
	regexp.MustCompile(`(?i)\bs2s\s+from\s+pwa\b`),
	regexp.MustCompile(`(?i)\bpwa\b.{0,40}\b(?:postback|tracker|keitaro)\b`),
}

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

func hasPWAPainSignal(text string) bool {
	body := strings.TrimSpace(text)
	if body == "" {
		return false
	}
	for _, rx := range pwaPainRe {
		if rx.MatchString(body) {
			return true
		}
	}
	lower := strings.ToLower(body)
	if !strings.Contains(lower, "pwa") && !strings.Contains(lower, "webview") {
		return false
	}
	for _, hint := range []string{"postback", "keitaro", "binom", "voluum", "tracker", "sub_id", "subid", "s2s"} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

func hasHostingIncidentPain(text string) bool {
	body := strings.TrimSpace(text)
	if body == "" {
		return false
	}
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
	lower := strings.ToLower(body)
	for _, hint := range trackerPainHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}
