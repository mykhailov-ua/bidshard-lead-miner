package model

import (
	"strings"
	"time"
)

type RawItem struct {
	Source    string
	Raw       string
	Contact   string
	Title     string
	Username  string
	MessageID int64
}

func (r RawItem) Text() string {
	if r.Raw != "" {
		return r.Raw
	}
	return r.Title
}

func (r RawItem) ContactTelegram() string {
	if r.Contact != "" {
		return normalizeTelegramContact(r.Contact)
	}
	if r.Username != "" {
		return normalizeTelegramContact(r.Username)
	}
	return ""
}

func normalizeTelegramContact(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(v), "telegram:") {
		return v
	}
	if strings.HasPrefix(v, "@") {
		return "telegram:" + v
	}
	return "telegram:@" + v
}

func (r RawItem) MaskedContact() string {
	contact := r.ContactTelegram()
	if contact == "" {
		contact = r.Contact
	}
	if contact == "" {
		return ""
	}
	if len(contact) <= 3 {
		return "***"
	}
	return contact[:1] + "***"
}

type Lead struct {
	TS       time.Time
	RoundID  string
	HashID   string
	Priority string
	Score    int
	Source   string
	Title    string
	Contacts []string
	Matched  []string
	Snippet  string
}
