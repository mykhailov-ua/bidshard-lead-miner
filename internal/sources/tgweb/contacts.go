package tgweb

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/validate"
)

var junkSkypeHandles = map[string]struct{}{
	"skype": {}, "live": {}, "username": {}, "user": {}, "contact": {},
	"none": {}, "null": {}, "undefined": {}, "example": {}, "test": {},
}

var preferredLPRLocals = []string{
	// Prefer partnership-facing mailboxes over generic contact@ when multiple on-domain emails exist.
	"partners", "partnerships", "partner", "affiliate", "affiliates",
	"bizdev", "business", "bd", "growth", "ads", "media",
}

var junkTelegramHandles = map[string]struct{}{
	"media": {}, "keyframes": {}, "starting": {}, "skype": {},
	"supports": {}, "import": {}, "charset": {}, "font-face": {},
	"trustpilot": {}, "boost": {},
}

func isJunkSkypeHandle(value string) bool {
	v := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "skype:")))
	if v == "" || len(v) < 4 {
		return true
	}
	if isJunkTelegramHandle(v) {
		return true
	}
	_, junk := junkSkypeHandles[v]
	return junk
}

func isJunkTelegramHandle(handle string) bool {
	handle = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(handle, "@")))
	if handle == "" {
		return true
	}
	_, junk := junkTelegramHandles[handle]
	return junk
}

func pickSiteLPR(page extract.Result, channel, siteDomain string) (extract.Contact, bool) {
	// Require at least one on-domain email or skype before accepting the page as a site lead.
	channel = normalizeChannel(channel)
	siteDomain = strings.ToLower(strings.TrimSpace(siteDomain))
	filtered := filterSiteContacts(page.Contacts, channel)
	if !hasSiteLPR(filtered, siteDomain) {
		return extract.Contact{}, false
	}
	primary := pickPrimarySiteContact(filtered, siteDomain)
	if primary.Value == "" {
		return extract.Contact{}, false
	}
	return primary, true
}

func filterSiteContacts(contacts []extract.Contact, channel string) []extract.Contact {
	out := make([]extract.Contact, 0, len(contacts))
	for _, c := range contacts {
		switch c.Type {
		case "telegram":
			handle := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(c.Value)), "@")
			if isJunkTelegramHandle(handle) {
				continue
			}
			// Drop the discovery channel handle; tgweb leads target site operators, not the Telegram poster.
			if channel != "" && handle == channel {
				continue
			}
		case "email":
			lower := strings.ToLower(c.Value)
			// Strip Sentry ingest noise often embedded in Next.js bundles.
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

func hasSiteLPR(contacts []extract.Contact, siteDomain string) bool {
	// On-domain non-role email counts as LPR; skype is a fallback when no qualifying email exists.
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

func pickPrimarySiteContact(contacts []extract.Contact, siteDomain string) extract.Contact {
	var emails []string
	for _, c := range contacts {
		if c.Type == "email" && c.Value != "" && emailMatchesSite(c.Value, siteDomain) && !validate.IsRoleEmail(c.Value) {
			emails = append(emails, c.Value)
		}
	}
	// Prefer partnership local-parts; fall back to first on-domain email, then skype.
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

// emailMatchesSite reports whether the email host equals siteDomain or is its subdomain.
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

func formatPrimaryContact(c extract.Contact) string {
	switch c.Type {
	case "skype":
		return "skype:" + strings.TrimPrefix(strings.ToLower(c.Value), "skype:")
	default:
		return c.Value
	}
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(channel, "@")))
}

func provenanceTitle(channel, domain string) string {
	channel = strings.TrimSpace(strings.TrimPrefix(channel, "@"))
	if channel != "" {
		return "site " + domain + " via telegram @" + channel
	}
	return "site " + domain
}

// ChannelFromSource parses tgweb:@channel:domain labels.
func ChannelFromSource(source string) string {
	source = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(source), "tgweb:"))
	if source == "" {
		return ""
	}
	parts := strings.Split(source, ":")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(parts[0]), "@")
}

// FilterPipelineContacts collapses tgweb leads to a single on-domain LPR for dedup and scoring.
// Non-tgweb sources pass contacts through unchanged.
func FilterPipelineContacts(source, primaryHint string, extracted []extract.Contact) []extract.Contact {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "tgweb:") {
		return extracted
	}
	domain := SiteDomainFromSource(source)
	if domain == "" {
		return nil
	}
	channel := ChannelFromSource(source)
	filtered := filterSiteContacts(extracted, channel)

	primaryHint = strings.TrimSpace(primaryHint)
	if primaryHint != "" {
		// Re-merge crawler primary hint in case extract pass dropped it from page text.
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

	primary := pickPrimarySiteContact(filtered, domain)
	if primary.Value == "" {
		return nil
	}
	return []extract.Contact{primary}
}
