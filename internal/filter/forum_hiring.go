package filter

import (
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/sources/forum"
)

var forumTeamHiringRe = regexp.MustCompile(
	`(?i)(` +
		`media\s+buyer|mediabuyer|медиабайер|` +
		`team\s+lead\s+media|head\s+of\s+acquisition|` +
		`traffic\s+manager|buyer\s+fb|buyer\s+google` +
		`).*(` +
		`hiring|recruiting|вакансия|ищем|open\s+position|job\s+offer|` +
		`набор\s+в\s+команду|#buyer|#head` +
		`)|(` +
		`hiring|recruiting|вакансия|ищем|open\s+position|job\s+offer` +
		`).*(` +
		`media\s+buyer|mediabuyer|медиабайер|team\s+lead\s+media` +
		`)`,
)

// ForumHostFromSource extracts host from forum:host/slug labels.
func ForumHostFromSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	source = strings.TrimPrefix(source, "forum:")
	if source == "" {
		return ""
	}
	if i := strings.Index(source, "/"); i >= 0 {
		return strings.TrimSuffix(source[:i], ".")
	}
	return strings.TrimSuffix(source, ".")
}

// IsForumTeamHiringPost reports media-buying team recruiting (P1 forum OSINT).
func IsForumTeamHiringPost(text, title string) bool {
	combined := strings.TrimSpace(title + " " + text)
	if combined == "" {
		return false
	}
	return forumTeamHiringRe.MatchString(combined)
}

// ForumTeamHiringBypassContextDrop allows hiring threads on allowlisted forum hosts.
func ForumTeamHiringBypassContextDrop(source, text, title string) bool {
	if !IsForumSource(source) {
		return false
	}
	host := ForumHostFromSource(source)
	if host == "" || !forum.IsKnownForumHost(host) {
		return false
	}
	return IsForumTeamHiringPost(text, title)
}
