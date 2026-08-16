package validate

import (
	"net/mail"
	"strings"
)

var disposableDomains = map[string]struct{}{
	"mailinator.com":     {},
	"guerrillamail.com":  {},
	"tempmail.com":       {},
	"10minutemail.com":   {},
	"throwaway.email":    {},
	"example.com":        {},
	"test.com":           {},
}

var roleLocalParts = map[string]struct{}{
	"ads":       {},
	"support":   {},
	"info":      {},
	"contact":   {},
	"hello":     {},
	"admin":     {},
	"sales":     {},
	"noreply":   {},
	"no-reply":  {},
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
	return true
}

func IsRoleEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local := parts[0]
	if _, ok := roleLocalParts[local]; ok {
		return true
	}
	return strings.Contains(local, "noreply")
}
