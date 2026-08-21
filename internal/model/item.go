package model

import (
	"strings"
	"time"

	"github.com/bidshard/parser/internal/scoring"
)

type RawItem struct {
	Source       string
	Raw          string
	Contact      string
	Title        string
	Username     string
	ForumUserID  string
	MessageID    int64
	ChannelAbout string
	CrawlHTML    string
	PostedAt     time.Time
}

func (r RawItem) Text() string {
	text := r.Title
	if r.Raw != "" {
		text = r.Raw
	}
	if r.CrawlHTML != "" {
		if hint := scoring.FormatStackHint(scoring.DetectCompetitorStack(r.CrawlHTML)); hint != "" {
			text = text + " " + hint
		}
	}
	return text
}

func (r RawItem) ContactTelegram() string {
	if r.Contact != "" {
		contact := strings.TrimSpace(r.Contact)
		if strings.Contains(contact, "@") && !strings.HasPrefix(contact, "@") &&
			!strings.HasPrefix(strings.ToLower(contact), "telegram:") {
			return ""
		}
		return normalizeTelegramContact(contact)
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
	TS                  time.Time
	RoundID             string
	HashID              string
	Priority            string
	Score               int
	Source              string
	Title               string
	Contacts            []string
	Matched             []string
	Snippet             string
	ICP                 string
	Hot                 bool
	SpendTier           string
	ICPWhy              string
	GeoCountry          string
	CompanyCountry      string
	CompanyName         string
	GeoSignals          []string
	GeoWhy              string
	WhoisCountry        string
	DomainAgeDays       int
	DisplayName         string
	GravatarName        string
	EmailVerified       bool
	PostedAt            time.Time
	Stack               []string
	Tags                []string
	Lang                string
	Status              string
	StatusAt            time.Time
	OutreachChannel     string
	OutreachAngle       string
	OutreachDraft       string
	PilotQualified      bool
	PilotWhy            string
	CompanyType         string
	EnrichSummary       string
	GeoConfidence       string
	AnalysisStatus      string
	AnalysisAt          time.Time
	EntityID            string
	EntitySightingCount int
	EntitySourceCount   int
	EntityHeat          float64
	HeatTier            string
}
