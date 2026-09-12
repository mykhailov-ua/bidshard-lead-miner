package validate

import (
	"regexp"
	"strings"
)

var githubSourceCodeSignals = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bimport\s+[\w*{]+\s+from\s+['"]`),
	regexp.MustCompile(`(?i)\bexport\s+default\s+(?:function|class)\b`),
	regexp.MustCompile(`(?i)\buse(?:State|Effect|Callback|Memo|Ref)\s*\(`),
	regexp.MustCompile(`(?i)\b(?:const|let|var)\s+\w+\s*=\s*`),
	regexp.MustCompile(`(?i)\bframer-motion\b`),
	regexp.MustCompile(`(?i)\bnext/(?:js|image|link|head)\b`),
	regexp.MustCompile(`(?i)\breact(?:\.|/|-dom)\b`),
	regexp.MustCompile(`(?i)\.tsx\b|\.jsx\b`),
	regexp.MustCompile(`(?i)<(?:div|span|script|html|head|body)\b`),
	regexp.MustCompile(`(?i)\bpackage\.json\b`),
	regexp.MustCompile(`(?i)\bnode_modules/`),
	regexp.MustCompile(`(?i)\bclassName\s*=`),
}

// IsGitHubSourceCodePaste reports pasted frontend/repo code in issue bodies.
func IsGitHubSourceCodePaste(text, title string) bool {
	combined := strings.TrimSpace(title + " " + text)
	if combined == "" {
		return false
	}
	if HasCommercialPainIntent(combined) || HasBuyerQuestionPattern(combined) {
		return false
	}
	hits := 0
	for _, re := range githubSourceCodeSignals {
		if re.MatchString(combined) {
			hits++
		}
	}
	if hits >= 2 {
		return true
	}
	if hits >= 1 && strings.Count(combined, ";") >= 3 {
		return true
	}
	return false
}
