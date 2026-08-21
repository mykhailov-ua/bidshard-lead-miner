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
	ruDomainRe    = regexp.MustCompile(`(?i)@[^@\s]+\.(ru|рф)([\s.,;]|$)`)
	byDomainRe    = regexp.MustCompile(`(?i)@[^@\s]+\.(by|бел)([\s.,;]|$)`)
	ruPhoneRe     = regexp.MustCompile(`\+7[\d\s\-()]{8,}`)
	byPhoneRe     = regexp.MustCompile(`\+375[\d\s\-()]{6,}`)
	bioRejectRe   = regexp.MustCompile(`(?i)(europe/moscow|\bmoscow\b|\bminsk\b|\brussia\b|\bbelarus\b|\bроссия\b|\bбеларусь\b)`)
	cyrillicRe    = regexp.MustCompile(`[а-яё]{8,}`)
	latinSignalRe = regexp.MustCompile(`(?i)[a-z]{3,}`)
)

func Filter(text string, contacts ...string) Result {
	body := strings.Join(append([]string{text}, contacts...), " ")
	lower := strings.ToLower(body)

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

func IsBlockedCountry(code string, blocked []string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, b := range blocked {
		if strings.ToUpper(strings.TrimSpace(b)) == code {
			return true
		}
	}
	return false
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
