package entity

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/bidshard/parser/internal/validate"
)

var (
	companyNoiseRe = regexp.MustCompile(`(?i)\b(llc|l\.l\.c|ltd|limited|inc|corp|corporation|gmbh|oy|ab|s\.r\.o|srl|pte|co)\b`)
	companyPipeRe  = regexp.MustCompile(`\s*[|/\\]\s*.*$`)
)

var freeMailDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
	"yahoo.com":      {},
	"yahoo.co.uk":    {},
	"hotmail.com":    {},
	"outlook.com":    {},
	"live.com":       {},
	"icloud.com":     {},
	"protonmail.com": {},
	"proton.me":      {},
	"mail.ru":        {},
	"yandex.ru":      {},
	"yandex.com":     {},
	"bk.ru":          {},
	"list.ru":        {},
	"inbox.ru":       {},
	"me.com":         {},
	"aol.com":        {},
	"gmx.com":        {},
	"gmx.de":         {},
	"web.de":         {},
	"mail.com":       {},
	"zoho.com":       {},
	"fastmail.com":   {},
	"tutanota.com":   {},
	"hey.com":        {},
	"qq.com":         {},
	"163.com":        {},
}

// NormalizeCompany folds network/company names into a stable lookup token.
func NormalizeCompany(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ToLower(name)
	if idx := strings.Index(name, "|"); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	name = companyPipeRe.ReplaceAllString(name, "")
	name = companyNoiseRe.ReplaceAllString(name, "")
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return -1
	}, name)
	name = strings.Join(strings.Fields(name), " ")
	return strings.TrimSpace(name)
}

// NormalizeDomain lowercases and trims a hostname key.
func NormalizeDomain(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "domain:")
	if host == "" || strings.Contains(host, " ") {
		return ""
	}
	if validate.IsBlacklistedDomain(host) {
		return ""
	}
	if _, free := freeMailDomains[host]; free {
		// Do not entity-link on gmail/yandex; use company name or tgweb site domain instead.
		return ""
	}
	return host
}

// EmailRootDomain returns the corporate email domain or "" for free/disposable mail.
func EmailRootDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return NormalizeDomain(parts[1])
}

// NormalizeTelegram folds telegram handles to @username form.
func NormalizeTelegram(handle string) string {
	handle = strings.TrimSpace(handle)
	handle = strings.TrimPrefix(strings.ToLower(handle), "telegram:")
	handle = strings.TrimPrefix(handle, "@")
	handle = strings.TrimSpace(handle)
	if handle == "" || len(handle) < 5 {
		return ""
	}
	return "@" + handle
}

// TelegramChannelFromSource extracts the channel username from telegram: source labels.
func TelegramChannelFromSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if !strings.HasPrefix(source, "telegram:") {
		return ""
	}
	rest := strings.TrimPrefix(source, "telegram:")
	rest = strings.TrimPrefix(rest, "@")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	return "@" + rest
}
