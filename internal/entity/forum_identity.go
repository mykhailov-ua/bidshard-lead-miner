package entity

import (
	"html"
	"strings"
)

// Known forum source label prefixes (see docs/backlog-entity-heat.md HEAT-P1-02).
// forum:affiliatefix.com/{slug}  -> family forum, host affiliatefix.com
// forum:blackhatworld.com/{slug}  -> family forum
// warrior:{thread-slug}           -> family warrior, host warriorforum
// reddit:{subreddit}              -> family reddit (no forum_user keys)

var skipForumUsernames = map[string]struct{}{
	"":          {},
	"anonymous": {},
	"guest":     {},
	"unknown":   {},
	"deleted":   {},
	"[deleted]": {},
}

// NormalizeForumHost folds a forum hostname into a stable lookup token.
func NormalizeForumHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimSuffix(host, ".")
	return host
}

// NormalizeForumUser returns a stable forum username for host-scoped keys.
func NormalizeForumUser(host, user string) string {
	user = normalizeForumUserLabel(user)
	if user == "" {
		return ""
	}
	host = NormalizeForumHost(host)
	if host == "" {
		return ""
	}
	return host + ":" + user
}

// NormalizeForumUID returns a stable XenForo user id key value host:id.
func NormalizeForumUID(host, uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ""
	}
	for _, r := range uid {
		if r < '0' || r > '9' {
			return ""
		}
	}
	host = NormalizeForumHost(host)
	if host == "" {
		return ""
	}
	return host + ":" + uid
}

func normalizeForumUserLabel(user string) string {
	user = html.UnescapeString(strings.TrimSpace(user))
	user = strings.TrimPrefix(user, "@")
	user = strings.TrimSpace(user)
	user = strings.ToLower(user)
	user = strings.TrimPrefix(user, "forum:user/")
	user = strings.TrimPrefix(user, "warrior:user/")
	if _, skip := skipForumUsernames[user]; skip {
		return ""
	}
	if len(user) < 2 {
		return ""
	}
	return user
}

// ForumHostFromSource extracts the forum host token from a source label.
func ForumHostFromSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(source, "forum:"):
		rest := strings.TrimPrefix(source, "forum:")
		if idx := strings.Index(rest, "/"); idx >= 0 {
			rest = rest[:idx]
		}
		return NormalizeForumHost(rest)
	case strings.HasPrefix(source, "warrior:"):
		return "warriorforum"
	default:
		return ""
	}
}

func isForumIdentitySource(source string) bool {
	switch SourceFamily(source) {
	case "forum", "warrior":
		return true
	default:
		return false
	}
}

func forumUserLabel(in ResolveInput) string {
	if u := normalizeForumUserLabel(in.ForumUser); u != "" {
		return u
	}
	if u := normalizeForumUserLabel(in.DisplayName); u != "" {
		return u
	}
	return ""
}

func appendForumIdentityKeys(keys []EntityKey, in ResolveInput) []EntityKey {
	if !isForumIdentitySource(in.Source) {
		return keys
	}
	host := ForumHostFromSource(in.Source)
	if host == "" {
		return keys
	}
	if uid := NormalizeForumUID(host, in.ForumUID); uid != "" {
		keys = append(keys, EntityKey{Kind: KindForumUID, Value: uid})
	}
	if user := forumUserLabel(in); user != "" {
		if canon := NormalizeForumUser(host, user); canon != "" {
			keys = append(keys, EntityKey{Kind: KindForumUser, Value: canon})
		}
	}
	return keys
}

// EnrichForumIdentity fills forum username and uid from raw crawl fields when absent.
func EnrichForumIdentity(in ResolveInput, username, title, forumUID string) ResolveInput {
	if in.ForumUser == "" {
		in.ForumUser = firstNonEmpty(strings.TrimSpace(username), strings.TrimSpace(title))
	}
	if in.ForumUID == "" {
		in.ForumUID = strings.TrimSpace(forumUID)
	}
	return in
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
