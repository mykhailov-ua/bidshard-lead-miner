package extract

import (
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/validate"
)

var (
	emailRe    = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	linkedInRe = regexp.MustCompile(`https?://(?:www\.)?linkedin\.com/in/[a-zA-Z0-9\-_%]+/?`)
	telegramRe = regexp.MustCompile(`(?:telegram:)?@([a-zA-Z][a-zA-Z0-9_]{4,})`)
	tmeRe      = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/([a-zA-Z][a-zA-Z0-9_]{4,})`)
	skypeRe    = regexp.MustCompile(`(?i)(?:skype|live)\s*[:@]\s*([a-zA-Z][a-zA-Z0-9.,\-_]{3,32})`)
)

type Contact struct {
	Type  string
	Value string
}

type Result struct {
	Contacts []Contact
	Rejected bool
	Reason   string
}

func Extract(text string, hints ...string) Result {
	combined := strings.Join(append([]string{text}, hints...), " ")

	seen := map[string]struct{}{}
	var contacts []Contact

	add := func(typ, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if typ == "email" && !validate.AcceptEmail(value) {
			return
		}
		key := typ + ":" + strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		contacts = append(contacts, Contact{Type: typ, Value: value})
	}

	for _, email := range emailRe.FindAllString(combined, -1) {
		if linkedInRe.MatchString(email) {
			continue
		}
		add("email", strings.ToLower(email))
	}

	scratch := combined
	for _, email := range emailRe.FindAllString(combined, -1) {
		scratch = strings.ReplaceAll(scratch, email, " ")
	}

	for _, m := range telegramRe.FindAllStringSubmatch(scratch, -1) {
		handle := "@" + m[1]
		if IsFalseTelegramHandle(handle) || IsJunkTelegramHandle(handle) {
			continue
		}
		add("telegram", handle)
	}
	for _, m := range tmeRe.FindAllStringSubmatch(scratch, -1) {
		handle := "@" + m[1]
		if IsFalseTelegramHandle(handle) || IsJunkTelegramHandle(handle) {
			continue
		}
		add("telegram", handle)
	}
	for _, m := range skypeRe.FindAllStringSubmatch(scratch, -1) {
		add("skype", strings.ToLower(m[1]))
	}

	for _, hint := range hints {
		hint = strings.TrimSpace(hint)
		if hint == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "reddit:") {
			value := strings.TrimSpace(hint[len("reddit:"):])
			if value != "" {
				add("reddit", value)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "discord:") {
			value := strings.TrimSpace(hint[len("discord:"):])
			if value != "" {
				add("discord", value)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "domain:") {
			value := strings.TrimSpace(hint[len("domain:"):])
			if value != "" {
				add("domain", strings.ToLower(value))
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "github:") {
			value := strings.TrimSpace(hint[len("github:"):])
			if value != "" {
				add("github", strings.ToLower(value))
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "forum:user/") {
			value := strings.TrimSpace(hint[len("forum:user/"):])
			if value != "" {
				add("forum_user", value)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "warrior:user/") {
			value := strings.TrimSpace(hint[len("warrior:user/"):])
			if value != "" {
				add("forum_user", value)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "review:") {
			value := strings.TrimSpace(hint[len("review:"):])
			if value != "" {
				add("review", value)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "telegram:user_id:") {
			val := strings.TrimSpace(hint[len("telegram:user_id:"):])
			if val != "" {
				add("telegram_user_id", val)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "telegram:") {
			val := strings.TrimPrefix(hint, "telegram:")
			if !IsJunkTelegramHandle(val) && !IsFalseTelegramHandle(val) {
				add("telegram", val)
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "skype:") {
			add("skype", strings.TrimPrefix(strings.ToLower(hint), "skype:"))
			continue
		}
		if strings.Contains(hint, "@") && emailRe.MatchString(hint) {
			add("email", strings.ToLower(emailRe.FindString(hint)))
		}
	}

	if len(contacts) == 0 && linkedInRe.MatchString(combined) {
		return Result{Rejected: true, Reason: "linkedin-only contact"}
	}

	return Result{Contacts: contacts}
}

func Primary(contacts []Contact) string {
	for _, c := range contacts {
		if c.Type == "email" && c.Value != "" {
			return c.Value
		}
	}
	for _, c := range contacts {
		if c.Value != "" {
			return c.Value
		}
	}
	return ""
}

func FormatAll(contacts []Contact) []string {
	out := make([]string, 0, len(contacts))
	for _, c := range contacts {
		if c.Type == "telegram" && !strings.HasPrefix(c.Value, "telegram:") {
			out = append(out, "telegram:"+c.Value)
			continue
		}
		if c.Type == "reddit" {
			value := strings.TrimPrefix(c.Value, "reddit:")
			out = append(out, "reddit:"+value)
			continue
		}
		if c.Type == "discord" {
			value := strings.TrimPrefix(c.Value, "discord:")
			out = append(out, "discord:"+value)
			continue
		}
		if c.Type == "domain" {
			value := strings.TrimPrefix(strings.ToLower(c.Value), "domain:")
			out = append(out, "domain:"+value)
			continue
		}
		if c.Type == "github" {
			value := strings.TrimPrefix(strings.ToLower(c.Value), "github:")
			out = append(out, "github:"+value)
			continue
		}
		if c.Type == "forum_user" {
			value := strings.TrimPrefix(c.Value, "forum:user/")
			value = strings.TrimPrefix(value, "warrior:user/")
			out = append(out, "forum:user/"+value)
			continue
		}
		if c.Type == "review" {
			value := strings.TrimPrefix(c.Value, "review:")
			out = append(out, "review:"+value)
			continue
		}
		if c.Type == "skype" {
			value := strings.TrimPrefix(strings.ToLower(c.Value), "skype:")
			out = append(out, "skype:"+value)
			continue
		}
		out = append(out, c.Value)
	}
	return out
}

func OnlyRoleEmails(contacts []Contact) bool {
	hasEmail := false
	for _, c := range contacts {
		if c.Type != "email" {
			return false
		}
		hasEmail = true
		if !validate.IsRoleEmail(c.Value) {
			return false
		}
	}
	return hasEmail
}

// HasReachableContact reports email, telegram, or skype suitable for outreach.
func HasReachableContact(contacts []Contact) bool {
	for _, c := range contacts {
		switch c.Type {
		case "email", "telegram", "skype":
			if strings.TrimSpace(c.Value) != "" {
				return true
			}
		}
	}
	return false
}

// MergeContacts dedupes by type:value, preserving order of base then append.
func MergeContacts(base, add []Contact) []Contact {
	if len(add) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(add))
	out := make([]Contact, 0, len(base)+len(add))
	for _, c := range base {
		key := c.Type + ":" + strings.ToLower(strings.TrimSpace(c.Value))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	for _, c := range add {
		key := c.Type + ":" + strings.ToLower(strings.TrimSpace(c.Value))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}
