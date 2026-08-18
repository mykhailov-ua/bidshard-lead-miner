package filter

import (
	"strings"
	"unicode"
)

const minTelegramTextRunes = 40

var telegramSpamPhrases = []string{
	"join our channel",
	"subscribe to",
	"subscribе to", // cyrillic е
	"via @",
	"forwarded from",
	"dm me for course",
	"free course",
	"affiliate course",
	"mentorship program",
	"signal group",
	"vip signals",
	"paid group",
	"click the link below",
	"limited slots",
}

// TelegramSpam reports low-quality Telegram messages before expensive scoring.
func TelegramSpam(source, text string) (bool, string) {
	if !isTelegramSource(source) {
		return false, ""
	}
	body := strings.TrimSpace(text)
	if body == "" {
		return true, "empty message"
	}

	lower := strings.ToLower(body)
	for _, phrase := range telegramSpamPhrases {
		if strings.Contains(lower, phrase) {
			return true, "spam phrase: " + phrase
		}
	}

	if runeLen(body) < minTelegramTextRunes && !hasPainKeyword(lower) {
		return true, "short message without pain signal"
	}

	return false, ""
}

func isTelegramSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(source), "telegram:")
}

// IsTelegramSource reports whether the raw item came from Telegram.
func IsTelegramSource(source string) bool {
	return isTelegramSource(source)
}

func hasPainKeyword(lower string) bool {
	hints := []string{
		"voluum", "keitaro", "binom", "redtrack", "postback", "tracker",
		"alternative", "afterburn", "ftd", "self-hosted", "cloak", "safe page",
	}
	for _, h := range hints {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

func runeLen(s string) int {
	return len([]rune(s))
}

// OnlyContactNoContext is true when text is just a contact hint with no substance.
func OnlyContactNoContext(text string) bool {
	body := strings.TrimSpace(text)
	if body == "" {
		return true
	}
	stripped := body
	for _, prefix := range []string{"telegram:", "@"} {
		stripped = strings.TrimPrefix(stripped, prefix)
	}
	stripped = strings.TrimSpace(stripped)
	if strings.Contains(stripped, " ") {
		return false
	}
	if strings.Contains(stripped, "@") && !strings.Contains(stripped, " ") {
		return runeCountLetters(stripped) < 8
	}
	return runeLen(body) < 12
}

func runeCountLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}
