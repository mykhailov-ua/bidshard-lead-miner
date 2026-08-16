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
