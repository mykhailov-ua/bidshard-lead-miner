package filter

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/validate"
)

// IsSerpSource reports DuckDuckGo SERP harvest items.
func IsSerpSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "serp:")
}

// SerpDomainFromSource returns hostname from serp:example.com items.
func SerpDomainFromSource(source string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(source)), "serp:")
}

// SerpBlacklistedSource rejects competitor tracker SEO domains from SERP hits.
func SerpBlacklistedSource(source string) bool {
	if !IsSerpSource(source) {
		return false
	}
	return validate.IsBlacklistedDomain(SerpDomainFromSource(source))
}

// SerpPlaceholderContact is the fake @serp:domain fallback when no handle exists.
func SerpPlaceholderContact(contact string) bool {
	lower := strings.ToLower(strings.TrimSpace(contact))
	if strings.HasPrefix(lower, "serp:") {
		return true
	}
	lower = strings.TrimPrefix(lower, "telegram:")
	lower = strings.TrimPrefix(lower, "@")
	return strings.HasPrefix(lower, "serp:") || extract.IsSyntheticContact(contact)
}

// SerpRequiresBuyerSignal blocks channel descriptions and listicles without buyer voice.
func SerpRequiresBuyerSignal(text string) bool {
	return validate.HasCommercialPainIntent(text) ||
		validate.HasBuyerQuestionPattern(text) ||
		scoring.HasBuyerIntentSignal(text)
}

// SerpForumSnippetOnly reports SERP hits on forum hosts without a forum_user handle.
func SerpForumSnippetOnly(source string, contacts []extract.Contact) bool {
	if !IsSerpSource(source) {
		return false
	}
	host := SerpDomainFromSource(source)
	if !forum.IsKnownForumHost(host) {
		return false
	}
	for _, c := range contacts {
		if c.Type == "forum_user" {
			return false
		}
	}
	return true
}

// SerpHasReachableContact rejects SERP rows that only have serp:domain placeholders.
func SerpHasReachableContact(contactHint string, contacts []extract.Contact) bool {
	if extract.HasReachableContact(contacts) || extract.HasEnrichableIdentity(contacts) {
		return true
	}
	if SerpPlaceholderContact(contactHint) {
		return false
	}
	return len(contacts) > 0
}
