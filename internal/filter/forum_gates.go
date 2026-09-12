package filter

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
)

// IsForumSource reports forum thread crawl items (forum:host/slug).
func IsForumSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "forum:")
}

// ForumHasForumUserContact reports whether contacts include a forum author handle.
func ForumHasForumUserContact(contacts []extract.Contact) bool {
	for _, c := range contacts {
		if c.Type == "forum_user" && strings.TrimSpace(c.Value) != "" {
			return true
		}
	}
	return false
}
