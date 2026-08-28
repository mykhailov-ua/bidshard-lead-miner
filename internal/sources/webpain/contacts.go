package webpain

import (
	"net/url"
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/validate"
)

var junkSkypeHandles = map[string]struct{}{
	"skype": {}, "live": {}, "username": {}, "user": {}, "contact": {},
	"none": {}, "null": {}, "undefined": {}, "example": {}, "test": {},
}

var preferredLPRLocals = []string{
	"partners", "partnerships", "partner", "affiliate", "affiliates",
	"bizdev", "business", "bd", "growth", "ads", "media",
}

// HostFromURL returns lowercase hostname without www.
func HostFromURL(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(u.Host, "www."))
}

// HostFromSource parses webpain:host or webpain:host/path labels.
func HostFromSource(source string) string {
	source = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(source)), "webpain:")
	if source == "" || source == "unknown" {
		return ""
	}
	if idx := strings.Index(source, "/"); idx > 0 {
		return source[:idx]
	}
	return source
}

func isJunkSkypeHandle(value string) bool {
	v := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "skype:")))
	if v == "" || len(v) < 4 {
		return true
	}
	if extract.IsJunkTelegramHandle(v) {
		return true
	}
	_, junk := junkSkypeHandles[v]
	return junk
}

func filterPageContacts(contacts []extract.Contact) []extract.Contact {
	out := make([]extract.Contact, 0, len(contacts))
	for _, c := range contacts {
		switch c.Type {
		case "telegram":
			if extract.IsJunkTelegramHandle(c.Value) || extract.IsFalseTelegramHandle(c.Value) {
				continue
			}
		case "email":
			lower := strings.ToLower(c.Value)
			if strings.Contains(lower, "ingest.") && strings.Contains(lower, "sentry") {
				continue
			}
			if validate.IsRoleEmail(c.Value) {
				continue
			}
		case "skype":
			if isJunkSkypeHandle(c.Value) {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

func hasOnDomainLPR(contacts []extract.Contact, siteDomain string) bool {
	for _, c := range contacts {
		if c.Type == "email" && emailMatchesSite(c.Value, siteDomain) && !validate.IsRoleEmail(c.Value) {
			return true
		}
	}
	for _, c := range contacts {
		if c.Type == "skype" && c.Value != "" && !isJunkSkypeHandle(c.Value) {
			return true
		}
	}
	return false
}

func pickPrimaryContact(contacts []extract.Contact, siteDomain string) extract.Contact {
	var emails []string
	for _, c := range contacts {
		if c.Type == "email" && c.Value != "" && emailMatchesSite(c.Value, siteDomain) && !validate.IsRoleEmail(c.Value) {
			emails = append(emails, c.Value)
		}
	}
	if best, ok := pickBestSiteEmail(emails); ok {
		return extract.Contact{Type: "email", Value: best}
	}
	for _, c := range contacts {
		if c.Type == "skype" && c.Value != "" && !isJunkSkypeHandle(c.Value) {
			return c
		}
	}
	return extract.Contact{}
}

func pickBestSiteEmail(emails []string) (string, bool) {
	if len(emails) == 0 {
		return "", false
	}
	for _, pref := range preferredLPRLocals {
		for _, email := range emails {
			local := emailLocalPart(email)
			if local == pref || strings.HasPrefix(local, pref) {
				return email, true
			}
		}
	}
	return emails[0], true
}

func emailLocalPart(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return ""
	}
	local := email[:at]
	if plus := strings.Index(local, "+"); plus >= 0 {
		local = local[:plus]
	}
	return local
}

func emailMatchesSite(email, siteDomain string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	siteDomain = strings.ToLower(strings.TrimSpace(siteDomain))
	if email == "" || siteDomain == "" {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	host := email[at+1:]
	return host == siteDomain || strings.HasSuffix(host, "."+siteDomain)
}

// PickPageLPR requires on-domain email or skype before emitting a webpain lead.
func PickPageLPR(page extract.Result, siteDomain string) (extract.Contact, bool) {
	siteDomain = strings.ToLower(strings.TrimSpace(siteDomain))
	filtered := filterPageContacts(page.Contacts)
	if !hasOnDomainLPR(filtered, siteDomain) {
		return extract.Contact{}, false
	}
	primary := pickPrimaryContact(filtered, siteDomain)
	if primary.Value == "" {
		return extract.Contact{}, false
	}
	return primary, true
}

func formatPrimaryContact(c extract.Contact) string {
	switch c.Type {
	case "skype":
		return "skype:" + strings.TrimPrefix(strings.ToLower(c.Value), "skype:")
	default:
		return c.Value
	}
}

// FilterPipelineContacts collapses webpain leads to one on-domain LPR (processor hot path).
func FilterPipelineContacts(source, primaryHint string, extracted []extract.Contact) []extract.Contact {
	domain := HostFromSource(source)
	if domain == "" {
		return nil
	}
	filtered := filterPageContacts(extracted)

	primaryHint = strings.TrimSpace(primaryHint)
	if primaryHint != "" {
		lower := strings.ToLower(primaryHint)
		if strings.HasPrefix(lower, "skype:") {
			skype := strings.TrimPrefix(lower, "skype:")
			if !isJunkSkypeHandle(skype) {
				filtered = append(filtered, extract.Contact{Type: "skype", Value: skype})
			}
		} else if strings.Contains(primaryHint, "@") && emailMatchesSite(primaryHint, domain) && !validate.IsRoleEmail(primaryHint) {
			filtered = append(filtered, extract.Contact{Type: "email", Value: primaryHint})
		}
	}

	primary := pickPrimaryContact(filtered, domain)
	if primary.Value == "" {
		return nil
	}
	return []extract.Contact{primary}
}
