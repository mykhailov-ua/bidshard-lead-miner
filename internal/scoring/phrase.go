package scoring

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// PhraseMatches finds phrase in lowercased text. Single-token phrases use word boundaries
// so "keitaro" does not match inside "keitaroinc".
func PhraseMatches(text, phrase string) bool {
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	if phrase == "" {
		return false
	}
	text = strings.ToLower(text)
	if strings.Contains(phrase, " ") {
		return strings.Contains(text, phrase)
	}
	start := 0
	for {
		idx := strings.Index(text[start:], phrase)
		if idx < 0 {
			return false
		}
		abs := start + idx
		end := abs + len(phrase)
		if phraseBounded(text, abs, end) {
			return true
		}
		start = abs + 1
	}
}

func phraseBounded(text string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:start])
		if isWordChar(r) {
			return false
		}
	}
	if end < len(text) {
		r, _ := utf8.DecodeRuneInString(text[end:])
		if isWordChar(r) {
			return false
		}
	}
	return true
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func phraseMatchAt(text string, pos int, phrase string) bool {
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	if phrase == "" || pos < 0 || pos+len(phrase) > len(text) {
		return false
	}
	lower := strings.ToLower(text)
	if !strings.HasPrefix(lower[pos:], phrase) {
		return false
	}
	if strings.Contains(phrase, " ") {
		return true
	}
	return phraseBounded(lower, pos, pos+len(phrase))
}
