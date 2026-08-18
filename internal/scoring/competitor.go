package scoring

import (
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/validate"
)

const competitorPainBoost = 20

var competitorStackPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)voluumtrk`),
	regexp.MustCompile(`(?i)voluum\.`),
	regexp.MustCompile(`(?i)keitaro`),
	regexp.MustCompile(`(?i)binom`),
	regexp.MustCompile(`(?i)redtrack`),
	regexp.MustCompile(`(?i)clickflare`),
	regexp.MustCompile(`(?i)bemob`),
}

// DetectCompetitorStack finds tracker fingerprints in crawl HTML/text.
func DetectCompetitorStack(body string) []string {
	if body == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, re := range competitorStackPatterns {
		hit := re.FindString(body)
		if hit == "" {
			continue
		}
		key := strings.ToLower(hit)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// CompetitorPainBoost adds stack bonus when competitor fingerprint coexists with pain language.
func CompetitorPainBoost(score int, text string, stack []string) int {
	if len(stack) == 0 {
		stack = DetectCompetitorStack(text)
	}
	if len(stack) == 0 {
		return score
	}
	if !validate.HasPainContext(text) && !HasCompetitorMention(strings.ToLower(text)) {
		return score
	}
	return score + competitorPainBoost
}

func FormatStackHint(stack []string) string {
	if len(stack) == 0 {
		return ""
	}
	return "[stack:" + strings.Join(stack, ",") + "]"
}
