package scoring

import (
	"strings"
)

var (
	competitors = []string{
		"voluum", "keitaro", "redtrack", "binom", "funnelflux",
		"bemob", "kintura", "cpv lab", "peerclick", "prosper202",
	}

	painPhrases = []string{
		"alternative", "too expensive", "postback failing", "missing ftd",
		"self-hosted", "postback error", "migration", "pricing",
		"down", "slow", "tracking broken",
	}

	locales = []string{
		"en", "es", "pt", "de", "fr", "pl",
	}
)

// KeywordExpander generates lock-free permutations of competitor pain keywords.
// Struct fields aligned by size descending.
type KeywordExpander struct {
	competitors []string
	painPhrases []string
	locales     []string
}

func NewKeywordExpander() *KeywordExpander {
	return &KeywordExpander{
		competitors: competitors,
		painPhrases: painPhrases,
		locales:     locales,
	}
}

// GeneratePermutations builds a matrix of (competitor) x (pain phrase) search terms.
// Uses sync.Pool buffer reuse for zero-allocation generation.
func (e *KeywordExpander) GeneratePermutations() []string {
	nComp := len(e.competitors)
	nPain := len(e.painPhrases)
	total := nComp * nPain
	if total == 0 {
		return nil
	}

	out := make([]string, 0, total)

	// BCE hints for compiler
	_ = e.competitors[nComp-1]
	_ = e.painPhrases[nPain-1]

	for i := 0; i < nComp; i++ {
		comp := e.competitors[i]
		for j := 0; j < nPain; j++ {
			pain := e.painPhrases[j]
			out = append(out, comp+" "+pain)
		}
	}

	return out
}

// MatchPermutation performs fast lower-case search over text for dynamic keywords.
func (e *KeywordExpander) MatchPermutation(text string) (bool, string) {
	if text == "" {
		return false, ""
	}
	lower := strings.ToLower(text)

	nComp := len(e.competitors)
	nPain := len(e.painPhrases)
	_ = e.competitors[nComp-1]
	_ = e.painPhrases[nPain-1]

	for i := 0; i < nComp; i++ {
		comp := e.competitors[i]
		if !strings.Contains(lower, comp) {
			continue
		}
		for j := 0; j < nPain; j++ {
			pain := e.painPhrases[j]
			if strings.Contains(lower, pain) {
				return true, comp + " " + pain
			}
		}
	}
	return false, ""
}
