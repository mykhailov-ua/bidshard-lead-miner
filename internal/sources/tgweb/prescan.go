package tgweb

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
)

// AggressivePrescanFromContact passes thin pages when crawl already found site LPR.
// Gate hard-reject, keyword prescan, and email-without-pain bypasses in the processor.
func AggressivePrescanFromContact(source, primaryContact string) bool {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "tgweb:") {
		return false
	}
	domain := SiteDomainFromSource(source)
	if domain == "" {
		return false
	}
	primaryContact = strings.TrimSpace(primaryContact)
	if primaryContact == "" {
		return false
	}
	lower := strings.ToLower(primaryContact)
	if strings.HasPrefix(lower, "skype:") {
		return len(strings.TrimPrefix(lower, "skype:")) > 0
	}
	if strings.Contains(primaryContact, "@") {
		return emailMatchesSite(primaryContact, domain)
	}
	return false
}

// HasSiteLPRContact reports on-domain email or skype in extracted contacts.
func HasSiteLPRContact(source string, contacts []extract.Contact) bool {
	domain := SiteDomainFromSource(source)
	if domain == "" {
		return false
	}
	_, ok := pickSiteLPR(extract.Result{Contacts: contacts}, "", domain)
	return ok
}
