package filter

import "strings"

// IsTgWebSource reports leads crawled from Telegram-linked affiliate network websites.
func IsTgWebSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "tgweb:")
}
