package validate

import (
	"regexp"
	"strings"
	"unicode"
)

var commercialPainRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:recommend|suggest|advise)\w*\s+(?:me\s+)?(?:a\s+)?tracker`),
	regexp.MustCompile(`(?i)\b(?:which|what)\s+tracker\b`),
	regexp.MustCompile(`(?i)\b(?:need|looking for)\s+(?:a\s+)?tracker\b`),
	regexp.MustCompile(`(?i)\balternative\s+to\s+(?:keitaro|binom|voluum|redtrack|clickflare|bemob)`),
	regexp.MustCompile(`(?i)\b(?:keitaro|binom|voluum|redtrack)\s+alternative`),
	regexp.MustCompile(`(?i)\bwhat(?:'s| is)\s+better\s+for\s+(?:twa|fb|facebook)`),
	regexp.MustCompile(`(?i)(?:not|doesn't|does not|won't)\s+track\s+clicks?`),
	regexp.MustCompile(`(?i)postback\s+(?:fail(?:ed|ing|s)?|broken|not\s+work(?:ing)?|down)`),
	regexp.MustCompile(`(?i)\bhow\s+to\s+(?:set\s+up|configure|setup)\s+cloak`),
	regexp.MustCompile(`(?i)\bmigrat(?:e|ing|ion)\s+(?:from|to)\s+(?:keitaro|binom|voluum|redtrack)`),
	regexp.MustCompile(`(?i)\bswitching\s+from\s+(?:keitaro|binom|voluum|redtrack)`),
	regexp.MustCompile(`(?i)посоветуйте\s+трекер`),
	regexp.MustCompile(`(?i)какой\s+трекер\s+(?:взять|выбрать|лучше)`),
	regexp.MustCompile(`(?i)альтернатива\s+(?:keitaro|binom|voluum|redtrack)`),
	regexp.MustCompile(`(?i)что\s+лучше\s+для\s+(?:twa|fb|фб)`),
	regexp.MustCompile(`(?i)не\s+трекает\s+клик`),
	regexp.MustCompile(`(?i)отвалился\s+постбек`),
	regexp.MustCompile(`(?i)как\s+настроить\s+клоак`),
}

var buyerQuestionRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:anyone|somebody|who)\s+(?:use|using|recommend|suggest)`),
	regexp.MustCompile(`(?i)\b(?:can|could)\s+(?:someone|anyone)\s+(?:help|recommend|suggest)`),
	regexp.MustCompile(`(?i)\b(?:does|do)\s+anyone\s+(?:use|know|recommend)`),
	regexp.MustCompile(`(?i)\bкакой\s+(?:трекер|tracker)`),
	regexp.MustCompile(`(?i)\bчто\s+(?:лучше|выбрать)`),
}

// HasCommercialPainIntent reports syntactic buyer-pain markers beyond bare keyword hits.
func HasCommercialPainIntent(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, re := range commercialPainRe {
		if re.MatchString(lower) {
			return true
		}
	}
	return false
}

// HasBuyerQuestionPattern reports outreach-style questions without requiring keyword registry hits.
func HasBuyerQuestionPattern(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "?") && intentMeaningfulRunes(lower) >= 24 {
		if HasCommercialPainIntent(lower) || HasPainContext(lower) {
			return true
		}
	}
	for _, re := range buyerQuestionRe {
		if re.MatchString(lower) {
			return true
		}
	}
	return false
}

func intentMeaningfulRunes(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n
}
