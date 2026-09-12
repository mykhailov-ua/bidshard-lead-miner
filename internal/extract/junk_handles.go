package extract

import "strings"

// junkTelegramHandles are CSS at-rules and HTML scrape tokens, not Telegram usernames.
var junkTelegramHandles = map[string]struct{}{
	"media": {}, "keyframes": {}, "starting": {}, "supports": {},
	"import": {}, "charset": {}, "font-face": {},
	"layer": {}, "page": {}, "scope": {}, "container": {},
	"namespace": {}, "document": {}, "property": {}, "counter-style": {},
	"viewport": {}, "skype": {}, "trustpilot": {}, "boost": {},
}

var falseTelegramHandles = map[string]struct{}{
	"github": {}, "reddit": {}, "serp": {},
}

// IsJunkTelegramHandle reports CSS/placeholder @handles from HTML scrape noise.
func IsJunkTelegramHandle(value string) bool {
	handle := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "@")))
	if handle == "" {
		return true
	}
	_, junk := junkTelegramHandles[handle]
	return junk
}

// IsFalseTelegramHandle reports regex false positives (e.g. @github from github.com).
func IsFalseTelegramHandle(value string) bool {
	handle := telegramHandleCore(value)
	if handle == "" {
		return true
	}
	if _, bad := falseTelegramHandles[handle]; bad {
		return true
	}
	if strings.HasPrefix(handle, "serp:") {
		return true
	}
	// Telegram usernames cannot contain dots; domain-shaped handles are SERP scrape noise.
	if strings.Contains(handle, ".") {
		return true
	}
	return false
}

func telegramHandleCore(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	v = strings.TrimPrefix(v, "telegram:")
	return strings.TrimPrefix(v, "@")
}

// IsSyntheticContact reports serp:domain and other non-outreach placeholders.
func IsSyntheticContact(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	v = strings.TrimPrefix(v, "telegram:")
	v = strings.TrimPrefix(v, "@")
	return strings.HasPrefix(v, "serp:") || strings.Contains(v, "@serp:")
}

// FilterJunkContacts drops CSS telegram handles and false-positive @mentions.
func FilterJunkContacts(contacts []Contact) []Contact {
	if len(contacts) == 0 {
		return contacts
	}
	out := make([]Contact, 0, len(contacts))
	for _, c := range contacts {
		if c.Type == "telegram" && (IsJunkTelegramHandle(c.Value) || IsFalseTelegramHandle(c.Value) || IsSyntheticContact(c.Value)) {
			continue
		}
		out = append(out, c)
	}
	return out
}
