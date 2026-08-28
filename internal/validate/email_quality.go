package validate

import (
	"net/mail"
	"strings"
)

var plusTagBlockDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
	"outlook.com":    {},
	"hotmail.com":    {},
	"live.com":       {},
	"yahoo.com":      {},
	"icloud.com":     {},
}

// executiveLocalParts are role mailboxes that still indicate a decision maker.
var executiveLocalParts = map[string]struct{}{
	"ceo":     {},
	"cto":     {},
	"cio":     {},
	"cfo":     {},
	"cmo":     {},
	"coo":     {},
	"founder": {},
	"owner":   {},
}

// affiliateB2BLocalParts are partnership-facing mailboxes we accept for cold outreach.
// Aligned with tgweb preferredLPRLocals; not generic support/info role rejects.
var affiliateB2BLocalParts = map[string]struct{}{
	"ads":          {},
	"partners":     {},
	"partnerships": {},
	"partner":      {},
	"affiliate":    {},
	"affiliates":   {},
	"bizdev":       {},
	"business":     {},
	"bd":           {},
	"growth":       {},
	"media":        {},
	"mediation":    {},
	"publisher":    {},
	"publishers":   {},
	"advertisers":  {},
	"advertising":  {},
	"programmatic": {},
}

var roleLocalParts = map[string]struct{}{
	"support":          {},
	"info":             {},
	"contact":          {},
	"hello":            {},
	"admin":            {},
	"sales":            {},
	"noreply":          {},
	"no-reply":         {},
	"donotreply":       {},
	"marketing":        {},
	"help":             {},
	"hr":               {},
	"careers":          {},
	"jobs":             {},
	"billing":          {},
	"service":          {},
	"customerservice":  {},
	"customer-service": {},
	"team":             {},
	"office":           {},
	"press":            {},
	"accounts":         {},
	"accounting":       {},
	"finance":          {},
	"feedback":         {},
	"enquiries":        {},
	"inquiry":          {},
	"recruitment":      {},
	"recruiting":       {},
	"legal":            {},
	"compliance":       {},
	"newsletter":       {},
	"notifications":    {},
	"notify":           {},
	"postmaster":       {},
	"webmaster":        {},
	"mail":             {},
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
	if IsDisposableDomain(parts[1]) {
		return false
	}
	if IsBlacklistedDomain(parts[1]) {
		return false
	}
	if strings.Contains(parts[1], "ingest.") && strings.Contains(parts[1], "sentry") {
		return false
	}
	if hasBlockedPlusTag(parts[0], parts[1]) {
		return false
	}
	if IsRoleEmail(email) {
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
	if _, ok := executiveLocalParts[local]; ok {
		return false
	}
	if _, ok := affiliateB2BLocalParts[local]; ok {
		return false
	}
	if _, ok := roleLocalParts[local]; ok {
		return true
	}
	if strings.HasPrefix(local, "support") ||
		strings.HasPrefix(local, "sales") ||
		strings.HasPrefix(local, "help") ||
		strings.HasPrefix(local, "marketing") {
		return true
	}
	return strings.Contains(local, "noreply")
}
