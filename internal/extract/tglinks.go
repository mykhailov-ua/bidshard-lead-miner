package extract

import (
	"regexp"
	"strings"
)

var (
	tmePublicRe = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/([a-zA-Z][a-zA-Z0-9_]{4,})`)
	tmeInviteRe = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/\+([A-Za-z0-9_-]{10,})`)
	joinChatRe  = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/joinchat/([A-Za-z0-9_-]+)`)
	atHandleRe  = regexp.MustCompile(`(?:^|[^\w])@([a-zA-Z][a-zA-Z0-9_]{4,})`)
)

// TelegramHandles extracts public channel usernames from t.me links and @handles in text.
func TelegramHandles(text string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, m := range tmePublicRe.FindAllStringSubmatch(text, -1) {
		u := strings.ToLower(m[1])
		if isBlockedTelegramHandle(u) {
			continue
		}
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			out = append(out, u)
		}
	}
	for _, m := range atHandleRe.FindAllStringSubmatch(text, -1) {
		u := strings.ToLower(m[1])
		if isBlockedTelegramHandle(u) {
			continue
		}
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			out = append(out, u)
		}
	}
	return out
}

// TelegramInviteHashes extracts private invite hashes from t.me/+ and joinchat links.
func TelegramInviteHashes(text string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, re := range []*regexp.Regexp{tmeInviteRe, joinChatRe} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			h := m[1]
			if _, ok := seen[h]; !ok {
				seen[h] = struct{}{}
				out = append(out, h)
			}
		}
	}
	return out
}

func isBlockedTelegramHandle(u string) bool {
	switch u {
	case "telegram", "support", "share", "addstickers", "joinchat", "s", "c", "iv":
		return true
	}
	return strings.HasSuffix(u, "bot")
}
