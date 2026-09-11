package geo

import (
	"regexp"
	"strings"
)

type Result struct {
	OK     bool
	Reason string
}

var (
	ruDomainRe      = regexp.MustCompile(`(?i)@[^@\s]+\.(ru|рф)([\s.,;]|$)`)
	byDomainRe      = regexp.MustCompile(`(?i)@[^@\s]+\.(by|бел)([\s.,;]|$)`)
	ruMailDomainRe  = regexp.MustCompile(`(?i)@(?:[^@\s]+\.)*(?:mail\.ru|yandex\.ru|ya\.ru|bk\.ru|list\.ru|inbox\.ru|rambler\.ru|internet\.ru)([\s.,;]|$)`)
	ruPhoneRe       = regexp.MustCompile(`\+7[\d\s\-()]{8,}`)
	byPhoneRe       = regexp.MustCompile(`\+375[\d\s\-()]{6,}`)
	blockedTLDRe    = regexp.MustCompile(`(?i)\.(ru|by|su|рф|бел)\b`)
	ruInfraAlwaysRe = regexp.MustCompile(`(?i)(reg\.ru|beget\.|timeweb\.|сбер(?:банк)?|тинькофф|т-банк|сбп|юmoney|qiwi|юмани|карта мир|оплата руб|рубл)`)
	ruLocationRe    = regexp.MustCompile(`(?i)(europe/moscow|\bmoscow\b|\bмосква\b|\bпитер\b|\bспб\b|санкт-петербург|\bminsk\b|\bминск\b|\brussia\b|\bроссия\b|\bbelarus\b|\bбеларусь\b)`)
	uaAffinityRe    = regexp.MustCompile(`(?i)(киев|київ|днепр|дніпро|одесса|одеса|львов|львів|подол|\+380|монобанк|monobank|privat24|приват|remote ua|mac kyiv|sempro)`)
)

func Filter(text string, contacts ...string) Result {
	body := strings.Join(append([]string{text}, contacts...), " ")
	lower := strings.ToLower(body)

	if ruMailDomainRe.MatchString(lower) {
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
	if blockedTLDRe.MatchString(body) {
		return Result{Reason: "ru/by tld"}
	}
	if ruInfraAlwaysRe.MatchString(body) {
		return Result{Reason: "ru/by infrastructure"}
	}
	if shouldCheckLocationMarkers(text, body) && ruLocationRe.MatchString(body) {
		return Result{Reason: "ru/by location"}
	}
	if uaAffinityRe.MatchString(body) {
		return Result{OK: true}
	}

	return Result{OK: true}
}

// Hostname-only strings skip location heuristics (e.g. traffic-moscow.example.com seed).
func shouldCheckLocationMarkers(text string, body string) bool {
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
		if r >= 'а' && r <= 'я' || r >= 'А' && r <= 'Я' || r == 'ё' || r == 'Ё' {
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
