package sink

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
)

// StoredContact is a normalized contact used for hash_id and Mongo storage.
type StoredContact struct {
	Type  string `bson:"type" json:"type"`
	Value string `bson:"value" json:"value"`
}

const (
	contactEmail    = "email"
	contactTelegram = "telegram"
	contactReddit   = "reddit"
	contactDiscord  = "discord"
	contactDomain   = "domain"
	contactGitHub   = "github"
	contactReview   = "review"
)

func ToStoredContacts(contacts []extract.Contact) []StoredContact {
	out := make([]StoredContact, 0, len(contacts))
	for _, c := range contacts {
		value := strings.TrimSpace(c.Value)
		if value == "" {
			continue
		}
		switch c.Type {
		case "email":
			out = append(out, StoredContact{
				Type:  contactEmail,
				Value: strings.ToLower(value),
			})
		case "telegram":
			value = strings.TrimPrefix(value, "telegram:")
			value = strings.TrimPrefix(value, "@")
			if value == "" {
				continue
			}
			out = append(out, StoredContact{
				Type:  contactTelegram,
				Value: "@" + strings.ToLower(value),
			})
		case "reddit":
			value = strings.TrimPrefix(value, "reddit:")
			value = strings.TrimPrefix(value, "u/")
			value = strings.TrimPrefix(value, "/u/")
			if value == "" {
				continue
			}
			out = append(out, StoredContact{
				Type:  contactReddit,
				Value: strings.ToLower(value),
			})
		case "discord":
			value = strings.TrimPrefix(value, "discord:")
			value = strings.TrimPrefix(value, "@")
			if value == "" {
				continue
			}
			out = append(out, StoredContact{
				Type:  contactDiscord,
				Value: strings.ToLower(value),
			})
		case "domain":
			value = strings.TrimPrefix(strings.ToLower(value), "domain:")
			if value == "" {
				continue
			}
			out = append(out, StoredContact{
				Type:  contactDomain,
				Value: value,
			})
		case "github":
			value = strings.TrimPrefix(strings.ToLower(value), "github:")
			if value == "" {
				continue
			}
			out = append(out, StoredContact{
				Type:  contactGitHub,
				Value: value,
			})
		case "review":
			value = strings.TrimPrefix(value, "review:")
			if value == "" {
				continue
			}
			out = append(out, StoredContact{
				Type:  contactReview,
				Value: strings.ToLower(value),
			})
		}
	}
	return out
}

func LeadHashIDFromExtract(contacts []extract.Contact) string {
	return LeadHashID(ToStoredContacts(contacts))
}

func ParseFormattedContacts(formatted []string) []extract.Contact {
	out := make([]extract.Contact, 0, len(formatted))
	for _, raw := range formatted {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(raw), "telegram:") {
			out = append(out, extract.Contact{
				Type:  "telegram",
				Value: strings.TrimPrefix(raw, "telegram:"),
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(raw), "reddit:") {
			out = append(out, extract.Contact{
				Type:  "reddit",
				Value: strings.TrimPrefix(raw, "reddit:"),
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(raw), "discord:") {
			out = append(out, extract.Contact{
				Type:  "discord",
				Value: strings.TrimPrefix(raw, "discord:"),
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(raw), "domain:") {
			out = append(out, extract.Contact{
				Type:  "domain",
				Value: strings.TrimPrefix(raw, "domain:"),
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(raw), "github:") {
			out = append(out, extract.Contact{
				Type:  "github",
				Value: strings.TrimPrefix(raw, "github:"),
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(raw), "review:") {
			out = append(out, extract.Contact{
				Type:  "review",
				Value: strings.TrimPrefix(raw, "review:"),
			})
			continue
		}
		if strings.Contains(raw, "@") && !strings.HasPrefix(raw, "@") {
			out = append(out, extract.Contact{Type: "email", Value: strings.ToLower(raw)})
			continue
		}
		if strings.HasPrefix(raw, "@") {
			out = append(out, extract.Contact{Type: "telegram", Value: raw})
		}
	}
	return out
}
