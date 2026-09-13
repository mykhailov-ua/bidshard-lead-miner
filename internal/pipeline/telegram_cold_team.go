package pipeline

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

func isTelegramColdTeam(item model.RawItem) bool {
	return strings.EqualFold(strings.TrimSpace(item.LeadType), "cold_team")
}

// telegramColdTeamContacts prefer the message author over the HR channel handle.
func telegramColdTeamContacts(item model.RawItem, contacts []extract.Contact) []extract.Contact {
	poster := strings.TrimSpace(item.Username)
	if poster == "" {
		return contacts
	}
	poster = strings.TrimPrefix(poster, "@")
	handle := "telegram:@" + poster

	out := make([]extract.Contact, 0, len(contacts)+1)
	out = append(out, extract.Contact{Type: "telegram", Value: handle})

	seen := map[string]struct{}{handle: {}}
	for _, c := range contacts {
		val := strings.TrimSpace(c.Value)
		if val == "" {
			continue
		}
		if strings.EqualFold(val, handle) {
			continue
		}
		// Drop channel self-contact from source when we have a human poster.
		if c.Type == "telegram" && strings.Contains(strings.ToLower(item.Source), strings.TrimPrefix(strings.ToLower(val), "telegram:")) {
			continue
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, c)
	}
	return out
}
