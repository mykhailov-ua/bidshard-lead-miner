package sink

import (
	"strings"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type LeadDoc struct {
	HashID         string          `bson:"hash_id" json:"hash_id"`
	TS             time.Time       `bson:"ts" json:"ts"`
	RoundID        string          `bson:"round_id" json:"round_id"`
	Priority       string          `bson:"priority" json:"priority"`
	Score          int             `bson:"score" json:"score"`
	Source         string          `bson:"source" json:"source"`
	Title          string          `bson:"title" json:"title"`
	Contacts       []StoredContact `bson:"contacts" json:"contacts"`
	Matched        []string        `bson:"matched" json:"matched"`
	Snippet        string          `bson:"snippet" json:"snippet"`
	ICP            string          `bson:"icp,omitempty" json:"icp,omitempty"`
	Hot            bool            `bson:"hot,omitempty" json:"hot,omitempty"`
	SpendTier      string          `bson:"spend_tier,omitempty" json:"spend_tier,omitempty"`
	ICPWhy         string          `bson:"icp_why,omitempty" json:"icp_why,omitempty"`
	GeoCountry     string          `bson:"geo_country,omitempty" json:"geo_country,omitempty"`
	CompanyCountry string          `bson:"company_country,omitempty" json:"company_country,omitempty"`
	CompanyName    string          `bson:"company_name,omitempty" json:"company_name,omitempty"`
	GeoSignals     []string        `bson:"geo_signals,omitempty" json:"geo_signals,omitempty"`
	GeoWhy         string          `bson:"geo_why,omitempty" json:"geo_why,omitempty"`
	WhoisCountry   string          `bson:"whois_country,omitempty" json:"whois_country,omitempty"`
	DomainAgeDays  int             `bson:"domain_age_days,omitempty" json:"domain_age_days,omitempty"`
	DisplayName    string          `bson:"display_name,omitempty" json:"display_name,omitempty"`
	GravatarName   string          `bson:"gravatar_name,omitempty" json:"gravatar_name,omitempty"`
	EmailVerified  bool            `bson:"email_verified,omitempty" json:"email_verified,omitempty"`
	PostedAt       time.Time       `bson:"posted_at,omitempty" json:"posted_at,omitempty"`
	Stack           []string        `bson:"stack,omitempty" json:"stack,omitempty"`
	Tags            []string        `bson:"tags,omitempty" json:"tags,omitempty"`
	Lang            string          `bson:"lang,omitempty" json:"lang,omitempty"`
	Status          string          `bson:"status,omitempty" json:"status,omitempty"`
	StatusAt        time.Time       `bson:"status_at,omitempty" json:"status_at,omitempty"`
	OutreachChannel string          `bson:"outreach_channel,omitempty" json:"outreach_channel,omitempty"`
	PilotQualified  bool            `bson:"pilot_qualified,omitempty" json:"pilot_qualified,omitempty"`
}

func ToLeadDoc(lead model.Lead) LeadDoc {
	contacts := ToStoredContacts(ParseFormattedContacts(lead.Contacts))
	ts := lead.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	snippet := strings.Join(strings.Fields(lead.Snippet), " ")
	if len(snippet) > 500 {
		snippet = snippet[:497] + "..."
	}
	status := lead.Status
	if status == "" {
		status = "new"
	}
	statusAt := lead.StatusAt
	if statusAt.IsZero() {
		statusAt = time.Now().UTC()
	}
	return LeadDoc{
		HashID:          lead.HashID,
		TS:              ts,
		RoundID:         lead.RoundID,
		Priority:        lead.Priority,
		Score:           lead.Score,
		Source:          lead.Source,
		Title:           lead.Title,
		Contacts:        contacts,
		Matched:         append([]string(nil), lead.Matched...),
		Snippet:         snippet,
		ICP:             lead.ICP,
		Hot:             lead.Hot,
		SpendTier:       lead.SpendTier,
		ICPWhy:          lead.ICPWhy,
		GeoCountry:      lead.GeoCountry,
		CompanyCountry:  lead.CompanyCountry,
		CompanyName:     lead.CompanyName,
		GeoSignals:      append([]string(nil), lead.GeoSignals...),
		GeoWhy:          lead.GeoWhy,
		WhoisCountry:    lead.WhoisCountry,
		DomainAgeDays:   lead.DomainAgeDays,
		DisplayName:     lead.DisplayName,
		GravatarName:    lead.GravatarName,
		EmailVerified:   lead.EmailVerified,
		PostedAt:        lead.PostedAt,
		Stack:           append([]string(nil), lead.Stack...),
		Tags:            append([]string(nil), lead.Tags...),
		Lang:            lead.Lang,
		Status:          status,
		StatusAt:        statusAt,
		OutreachChannel: lead.OutreachChannel,
		PilotQualified:  lead.PilotQualified,
	}
}

func LeadJSONMap(lead model.Lead) map[string]any {
	doc := ToLeadDoc(lead)
	return map[string]any{
		"hash_id":  doc.HashID,
		"ts":       doc.TS.Format(time.RFC3339),
		"round_id": doc.RoundID,
		"priority": doc.Priority,
		"score":    doc.Score,
		"source":   doc.Source,
		"title":    doc.Title,
		"contacts": doc.Contacts,
		"matched":  doc.Matched,
		"snippet":  doc.Snippet,
	}
}
