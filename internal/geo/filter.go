package geo

import (
	"regexp"
	"strings"
	"unicode"
)

type Result struct {
	OK     bool
	Reason string
}

var (
	ruDomainRe     = regexp.MustCompile(`(?i)@[^@\s]+\.(ru|рф)([\s.,;]|$)`)
	byDomainRe     = regexp.MustCompile(`(?i)@[^@\s]+\.(by|бел)([\s.,;]|$)`)
	ruMailDomainRe = regexp.MustCompile(`(?i)@(?:[^@\s]+\.)*(?:mail\.ru|yandex\.ru|ya\.ru|bk\.ru|list\.ru|inbox\.ru|rambler\.ru|internet\.ru)([\s.,;]|$)`)
	ruPhoneRe      = regexp.MustCompile(`\+7[\d\s\-()]{8,}`)
	byPhoneRe      = regexp.MustCompile(`\+375[\d\s\-()]{6,}`)
	bioRejectRe    = regexp.MustCompile(`(?i)(europe/moscow|\bmoscow\b|\bminsk\b|\brussia\b|\bbelarus\b|\bроссия\b|\bбеларусь\b)`)
	cyrillicRe     = regexp.MustCompile(`[а-яё]{8,}`)
	latinSignalRe  = regexp.MustCompile(`(?i)[a-z]{3,}`)
)

func Filter(text string, contacts ...string) Result {
	body := strings.Join(append([]string{text}, contacts...), " ")
	lower := strings.ToLower(body)

	if ruMailDomainRe.MatchString(lower) {
		// Check before .ru TLD: user@mail.ru would otherwise match ruDomainRe as *@mail.ru.
		return Result{Reason: "ru mail domain"}
	}
	if ruDomainRe.MatchString(lower) {
		return Result{Reason: "ru domain"}
	}
	if byDomainRe.MatchString(lower) {
		return Result{Reason: "by domain"}
	}
	if ruPhoneRe.MatchString(body) {
		return Result{Reason: "ru phone"}
	}
	if byPhoneRe.MatchString(body) {
		return Result{Reason: "by phone"}
	}
	if shouldCheckBio(text, body) && bioRejectRe.MatchString(body) {
		return Result{Reason: "ru/by bio signal"}
	}
	if longCyrillicWithoutLatin(body) {
		return Result{Reason: "cyrillic-only context"}
	}
	if cyrillicHeavyWithoutLatin(body) {
		return Result{Reason: "cyrillic-heavy context"}
	}

	return Result{OK: true}
}

// Hostname-only strings (seed domains) skip bio keyword heuristics.
func shouldCheckBio(text string, body string) bool {
	if strings.Contains(body, "@") {
		return true
	}
	if ruPhoneRe.MatchString(body) || byPhoneRe.MatchString(body) {
		return true
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return len(strings.Fields(body)) > 1
	}
	return strings.Contains(trimmed, " ")
}

func longCyrillicWithoutLatin(text string) bool {
	if !cyrillicRe.MatchString(text) {
		return false
	}
	return !latinSignalRe.MatchString(text)
}

func cyrillicHeavyWithoutLatin(text string) bool {
	cyr, lat := scriptCounts(text)
	if cyr < 20 {
		return false
	}
	if lat >= 20 {
		return false
	}
	total := cyr + lat
	if total == 0 {
		return false
	}
	return cyr*100/total >= 35
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

func IsBlockedCountry(code string, blocked []string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, b := range blocked {
		if strings.ToUpper(strings.TrimSpace(b)) == code {
			return true
		}
	}
	return false
}

var blockedTLDS = map[string]struct{}{
	"ru": {}, "рф": {}, "by": {}, "бел": {}, "su": {},
}

// IsBlockedTLD reports RU/BY country-code TLDs used for geo hard reject.
func IsBlockedTLD(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	host = strings.TrimPrefix(host, "www.")
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return false
	}
	tld := parts[len(parts)-1]
	_, blocked := blockedTLDS[tld]
	return blocked
}

func HasCyrillicRun(text string, min int) bool {
	run := 0
	for _, r := range text {
		if unicode.In(r, unicode.Cyrillic) {
			run++
			if run >= min {
				return true
			}
			continue
		}
		run = 0
	}
	return false
}
