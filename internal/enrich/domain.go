package enrich

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/sources/tgweb"
)

// TargetDomain picks the best domain for RDAP/DNS enrichment.
func TargetDomain(source string, contacts []extract.Contact) string {
	if d := emailDomain(primaryEmail(contacts)); d != "" {
		return d
	}
	for _, c := range contacts {
		if c.Type == "domain" && c.Value != "" {
			return strings.ToLower(strings.TrimPrefix(c.Value, "domain:"))
		}
	}
	return domainFromSource(source)
}

func emailDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[1] == "" {
		return ""
	}
	return parts[1]
}

func domainFromSource(source string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	switch {
	case strings.HasPrefix(source, "ads_txt:"):
		return strings.TrimPrefix(source, "ads_txt:")
	case strings.HasPrefix(source, "lander:"):
		return strings.TrimPrefix(source, "lander:")
	case strings.HasPrefix(source, "supply:"):
		return strings.TrimPrefix(source, "supply:")
	case strings.HasPrefix(source, "ct:"):
		return strings.TrimPrefix(source, "ct:")
	case strings.HasPrefix(source, "tgweb:"):
		return tgweb.SiteDomainFromSource(source)
	}
	return ""
}

func primaryEmail(contacts []extract.Contact) string {
	for _, c := range contacts {
		if c.Type == "email" && c.Value != "" {
			return c.Value
		}
	}
	return ""
}
