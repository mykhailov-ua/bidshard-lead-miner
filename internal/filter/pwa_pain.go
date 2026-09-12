package filter

import (
	"regexp"
	"strings"
)

var pwaPainRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bpostback\s+from\s+app\b`),
	regexp.MustCompile(`(?i)\bwebview\s+stuck\b`),
	regexp.MustCompile(`(?i)\bsub[_-]?id\b.{0,40}\b(?:keitaro|binom|voluum|tracker)\b`),
	regexp.MustCompile(`(?i)\b(?:keitaro|binom|voluum|tracker)\b.{0,40}\bsub[_-]?id\b`),
	regexp.MustCompile(`(?i)\bredirect\s+delay\b`),
	regexp.MustCompile(`(?i)\bs2s\s+from\s+pwa\b`),
	regexp.MustCompile(`(?i)\bpwa\b.{0,40}\b(?:postback|tracker|keitaro)\b`),
	regexp.MustCompile(`(?i)\b(?:postback|tracker|keitaro)\b.{0,40}\bpwa\b`),
}

var pwaProviderHints = []string{
	"pwa.group", "pwa.market", "irent", "td apps", "pwa rent", "ios rent",
	"webview", "progressive web app",
}

// HasPWAPainSignal reports PWA/iOS rent client tracker pain (H3).
func HasPWAPainSignal(text string) bool {
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
	hasPWA := strings.Contains(lower, "pwa") || strings.Contains(lower, "webview")
	if !hasPWA {
		return false
	}
	for _, hint := range []string{"postback", "keitaro", "binom", "voluum", "tracker", "sub_id", "subid", "s2s"} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// IsPWAChannelHint reports discover/registry hints for PWA provider chats.
func IsPWAChannelHint(username, title, query string) bool {
	combined := strings.ToLower(strings.TrimSpace(username + " " + title + " " + query))
	if combined == "" {
		return false
	}
	for _, hint := range pwaProviderHints {
		if strings.Contains(combined, hint) {
			return true
		}
	}
	for _, token := range []string{"pwa", "webview", "sub_id", "ios rent"} {
		if strings.Contains(combined, token) {
			return true
		}
	}
	return false
}
