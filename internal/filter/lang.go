package filter

import (
	"strings"
	"unicode"
)

const minCyrillicRunes = 40

var latinAllowHints = []string{
	" the ", " and ", " for ", " with ", " tracker", " voluum", " postback",
	" alternative", " affiliate", " traffic", " campaign",
	" de ", " la ", " el ", " para ", " com ", " não ", " tráfico", " afiliado",
	" dla ", " nie ", " mit ", " für ", " pour ", " avec ",
}

// RejectLongCyrillicWithoutLatin rejects long Cyrillic-only posts (README geo/lang rule).
func RejectLongCyrillicWithoutLatin(text string) (bool, string) {
	cyr, lat := scriptCounts(text)
	if cyr < minCyrillicRunes {
		return false, ""
	}
	if lat >= 12 || hasLatinAllowHint(strings.ToLower(text)) {
		return false, ""
	}
	return true, "long cyrillic without EN/ES/PT context"
}

// DetectLanguage returns basic language code based on text markers.
func DetectLanguage(text string) string {
	lower := strings.ToLower(text)
	cyr, lat := scriptCounts(text)

	if cyr > minCyrillicRunes {
		return "ru"
	}
	if lat == 0 {
		return "unknown"
	}

	switch {
	case strings.Contains(lower, "alternatywa") || strings.Contains(lower, "trackera") || strings.Contains(lower, "nie dziala") || strings.Contains(lower, "szukam"):
		return "pl"
	case strings.Contains(lower, "alternativa ao") || strings.Contains(lower, "muito caro") || strings.Contains(lower, "nao funciona") || strings.Contains(lower, "procurando"):
		return "pt"
	case strings.Contains(lower, "alternativa a") || strings.Contains(lower, "muy caro") || strings.Contains(lower, "no funciona") || strings.Contains(lower, "buscando"):
		return "es"
	case strings.Contains(lower, "zu teuer") || strings.Contains(lower, "funktioniert nicht") || strings.Contains(lower, "weiterleitung") || strings.Contains(lower, "suche tracker"):
		return "de"
	case strings.Contains(lower, "trop cher") || strings.Contains(lower, "ne fonctionne pas") || strings.Contains(lower, "rediriger") || strings.Contains(lower, "recherche tracker"):
		return "fr"
	default:
		return "en"
	}
}

func scriptCounts(text string) (cyrillic, latin int) {
	for _, r := range text {
		if unicode.Is(unicode.Cyrillic, r) {
			cyrillic++
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			latin++
		}
	}
	return cyrillic, latin
}

func hasLatinAllowHint(lower string) bool {
	for _, hint := range latinAllowHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}
