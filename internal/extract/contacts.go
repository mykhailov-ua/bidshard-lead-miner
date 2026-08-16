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
		add("telegram", "@"+m[1])
	}
	for _, m := range tmeRe.FindAllStringSubmatch(scratch, -1) {
		add("telegram", "@"+m[1])
	}

	for _, hint := range hints {
		hint = strings.TrimSpace(hint)
		if hint == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(hint), "telegram:") {
			add("telegram", strings.TrimPrefix(hint, "telegram:"))
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
