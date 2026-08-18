package validate

import (
	"net/mail"
	"strings"
)

var disposableDomains = map[string]struct{}{
	"mailinator.com":      {},
	"guerrillamail.com":   {},
	"tempmail.com":        {},
	"10minutemail.com":    {},
	"throwaway.email":     {},
	"example.com":         {},
	"test.com":            {},
	"yopmail.com":         {},
	"sharklasers.com":     {},
	"trashmail.com":       {},
	"getnada.com":         {},
	"temp-mail.org":       {},
	"dispostable.com":     {},
	"maildrop.cc":         {},
	"fakeinbox.com":       {},
	"spamgourmet.com":     {},
	"mailnesia.com":       {},
	"tempail.com":         {},
	"moakt.com":           {},
	"emailondeck.com":     {},
	"mohmal.com":          {},
	"burnermail.io":       {},
	"mailcatch.com":       {},
	"mytemp.email":        {},
	"tmpmail.net":         {},
	"tmpmail.org":         {},
	"mail-temp.com":       {},
}

var plusTagBlockDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
	"outlook.com":    {},
	"hotmail.com":    {},
	"live.com":       {},
	"yahoo.com":      {},
	"icloud.com":     {},
}

var roleLocalParts = map[string]struct{}{
	"ads":        {},
	"support":    {},
	"info":       {},
	"contact":    {},
	"hello":      {},
	"admin":      {},
	"sales":      {},
	"noreply":    {},
	"no-reply":   {},
	"donotreply": {},
}

func AcceptEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	if strings.HasSuffix(email, ".png") || strings.HasSuffix(email, ".jpg") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	email = strings.ToLower(addr.Address)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if strings.HasSuffix(parts[1], "users.noreply.github.com") {
		return false
	}
	if _, ok := disposableDomains[parts[1]]; ok {
		return false
	}
	if hasBlockedPlusTag(parts[0], parts[1]) {
		return false
	}
	return true
}

func hasBlockedPlusTag(local, domain string) bool {
	if _, ok := plusTagBlockDomains[domain]; !ok {
		return false
	}
	plus := strings.Index(local, "+")
	return plus > 0
}

func IsRoleEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local := parts[0]
	if idx := strings.Index(local, "+"); idx >= 0 {
		local = local[:idx]
	}
	if _, ok := roleLocalParts[local]; ok {
		return true
	}
	return strings.Contains(local, "noreply")
}
