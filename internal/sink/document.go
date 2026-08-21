package sink

import (
	"strings"
	"time"

	"github.com/bidshard/parser/internal/model"
	"go.mongodb.org/mongo-driver/bson"
)

type LeadDoc struct {
	HashID              string          `bson:"hash_id" json:"hash_id"`
	TS                  time.Time       `bson:"ts" json:"ts"`
	RoundID             string          `bson:"round_id" json:"round_id"`
	Priority            string          `bson:"priority" json:"priority"`
	Score               int             `bson:"score" json:"score"`
	Source              string          `bson:"source" json:"source"`
	Title               string          `bson:"title" json:"title"`
	Contacts            []StoredContact `bson:"contacts" json:"contacts"`
	Matched             []string        `bson:"matched" json:"matched"`
	Snippet             string          `bson:"snippet" json:"snippet"`
	ICP                 string          `bson:"icp,omitempty" json:"icp,omitempty"`
	Hot                 bool            `bson:"hot,omitempty" json:"hot,omitempty"`
	SpendTier           string          `bson:"spend_tier,omitempty" json:"spend_tier,omitempty"`
	ICPWhy              string          `bson:"icp_why,omitempty" json:"icp_why,omitempty"`
	GeoCountry          string          `bson:"geo_country,omitempty" json:"geo_country,omitempty"`
	CompanyCountry      string          `bson:"company_country,omitempty" json:"company_country,omitempty"`
	CompanyName         string          `bson:"company_name,omitempty" json:"company_name,omitempty"`
	GeoSignals          []string        `bson:"geo_signals,omitempty" json:"geo_signals,omitempty"`
	GeoWhy              string          `bson:"geo_why,omitempty" json:"geo_why,omitempty"`
	WhoisCountry        string          `bson:"whois_country,omitempty" json:"whois_country,omitempty"`
	DomainAgeDays       int             `bson:"domain_age_days,omitempty" json:"domain_age_days,omitempty"`
	DisplayName         string          `bson:"display_name,omitempty" json:"display_name,omitempty"`
	GravatarName        string          `bson:"gravatar_name,omitempty" json:"gravatar_name,omitempty"`
	EmailVerified       bool            `bson:"email_verified,omitempty" json:"email_verified,omitempty"`
	PostedAt            time.Time       `bson:"posted_at,omitempty" json:"posted_at,omitempty"`
	Stack               []string        `bson:"stack,omitempty" json:"stack,omitempty"`
	Tags                []string        `bson:"tags,omitempty" json:"tags,omitempty"`
	Lang                string          `bson:"lang,omitempty" json:"lang,omitempty"`
	Status              string          `bson:"status,omitempty" json:"status,omitempty"`
	StatusAt            time.Time       `bson:"status_at,omitempty" json:"status_at,omitempty"`
	OutreachChannel     string          `bson:"outreach_channel,omitempty" json:"outreach_channel,omitempty"`
	OutreachAngle       string          `bson:"outreach_angle,omitempty" json:"outreach_angle,omitempty"`
	OutreachDraft       string          `bson:"outreach_draft,omitempty" json:"outreach_draft,omitempty"`
	PilotQualified      bool            `bson:"pilot_qualified,omitempty" json:"pilot_qualified,omitempty"`
	PilotWhy            string          `bson:"pilot_why,omitempty" json:"pilot_why,omitempty"`
	CompanyType         string          `bson:"company_type,omitempty" json:"company_type,omitempty"`
	EnrichSummary       string          `bson:"enrich_summary,omitempty" json:"enrich_summary,omitempty"`
	GeoConfidence       string          `bson:"geo_confidence,omitempty" json:"geo_confidence,omitempty"`
	AnalysisStatus      string          `bson:"analysis_status,omitempty" json:"analysis_status,omitempty"`
	AnalysisAt          time.Time       `bson:"analysis_at,omitempty" json:"analysis_at,omitempty"`
	EntityID            string          `bson:"entity_id,omitempty" json:"entity_id,omitempty"`
	EntitySightingCount int             `bson:"entity_sighting_count,omitempty" json:"entity_sighting_count,omitempty"`
	EntitySourceCount   int             `bson:"entity_source_count,omitempty" json:"entity_source_count,omitempty"`
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
	doc := LeadDoc{
		HashID:              lead.HashID,
		TS:                  ts,
		RoundID:             lead.RoundID,
		Priority:            lead.Priority,
		Score:               lead.Score,
		Source:              lead.Source,
		Title:               lead.Title,
		Contacts:            contacts,
		Matched:             append([]string(nil), lead.Matched...),
		Snippet:             snippet,
		ICP:                 lead.ICP,
		Hot:                 lead.Hot,
		SpendTier:           lead.SpendTier,
		ICPWhy:              lead.ICPWhy,
		GeoCountry:          lead.GeoCountry,
		CompanyCountry:      lead.CompanyCountry,
		CompanyName:         lead.CompanyName,
		GeoSignals:          append([]string(nil), lead.GeoSignals...),
		GeoWhy:              lead.GeoWhy,
		WhoisCountry:        lead.WhoisCountry,
		DomainAgeDays:       lead.DomainAgeDays,
		DisplayName:         lead.DisplayName,
		GravatarName:        lead.GravatarName,
		EmailVerified:       lead.EmailVerified,
		PostedAt:            lead.PostedAt,
		Stack:               append([]string(nil), lead.Stack...),
		Tags:                append([]string(nil), lead.Tags...),
		Lang:                lead.Lang,
		Status:              status,
		StatusAt:            statusAt,
		OutreachChannel:     lead.OutreachChannel,
		OutreachAngle:       lead.OutreachAngle,
		OutreachDraft:       lead.OutreachDraft,
		PilotQualified:      lead.PilotQualified,
		PilotWhy:            lead.PilotWhy,
		CompanyType:         lead.CompanyType,
		EnrichSummary:       lead.EnrichSummary,
		GeoConfidence:       lead.GeoConfidence,
		AnalysisStatus:      lead.AnalysisStatus,
		AnalysisAt:          lead.AnalysisAt,
		EntityID:            lead.EntityID,
		EntitySightingCount: lead.EntitySightingCount,
		EntitySourceCount:   lead.EntitySourceCount,
	}
	return doc
}

// LeadDocUpdateFields are mutable on re-sight; status fields stay on insert only.
type LeadDocUpdateFields struct {
	RoundID             string          `bson:"round_id"`
	Priority            string          `bson:"priority"`
	Score               int             `bson:"score"`
	Source              string          `bson:"source"`
	Title               string          `bson:"title"`
	Contacts            []StoredContact `bson:"contacts"`
	Matched             []string        `bson:"matched"`
	Snippet             string          `bson:"snippet"`
	ICP                 string          `bson:"icp,omitempty"`
	Hot                 bool            `bson:"hot,omitempty"`
	SpendTier           string          `bson:"spend_tier,omitempty"`
	ICPWhy              string          `bson:"icp_why,omitempty"`
	GeoCountry          string          `bson:"geo_country,omitempty"`
	CompanyCountry      string          `bson:"company_country,omitempty"`
	CompanyName         string          `bson:"company_name,omitempty"`
	GeoSignals          []string        `bson:"geo_signals,omitempty"`
	GeoWhy              string          `bson:"geo_why,omitempty"`
	WhoisCountry        string          `bson:"whois_country,omitempty"`
	DomainAgeDays       int             `bson:"domain_age_days,omitempty"`
	DisplayName         string          `bson:"display_name,omitempty"`
	GravatarName        string          `bson:"gravatar_name,omitempty"`
	EmailVerified       bool            `bson:"email_verified,omitempty"`
	PostedAt            time.Time       `bson:"posted_at,omitempty"`
	Stack               []string        `bson:"stack,omitempty"`
	Tags                []string        `bson:"tags,omitempty"`
	Lang                string          `bson:"lang,omitempty"`
	OutreachChannel     string          `bson:"outreach_channel,omitempty"`
	OutreachAngle       string          `bson:"outreach_angle,omitempty"`
	OutreachDraft       string          `bson:"outreach_draft,omitempty"`
	PilotQualified      bool            `bson:"pilot_qualified,omitempty"`
	PilotWhy            string          `bson:"pilot_why,omitempty"`
	CompanyType         string          `bson:"company_type,omitempty"`
	EnrichSummary       string          `bson:"enrich_summary,omitempty"`
	GeoConfidence       string          `bson:"geo_confidence,omitempty"`
	EntityID            string          `bson:"entity_id,omitempty"`
	EntitySightingCount int             `bson:"entity_sighting_count,omitempty"`
	EntitySourceCount   int             `bson:"entity_source_count,omitempty"`
}

func ToLeadDocUpdateBSON(doc LeadDoc) (bson.M, error) {
	raw, err := bson.Marshal(LeadDocUpdateFields{
		RoundID:             doc.RoundID,
		Priority:            doc.Priority,
		Score:               doc.Score,
		Source:              doc.Source,
		Title:               doc.Title,
		Contacts:            doc.Contacts,
		Matched:             doc.Matched,
		Snippet:             doc.Snippet,
		ICP:                 doc.ICP,
		Hot:                 doc.Hot,
		SpendTier:           doc.SpendTier,
		ICPWhy:              doc.ICPWhy,
		GeoCountry:          doc.GeoCountry,
		CompanyCountry:      doc.CompanyCountry,
		CompanyName:         doc.CompanyName,
		GeoSignals:          doc.GeoSignals,
		GeoWhy:              doc.GeoWhy,
		WhoisCountry:        doc.WhoisCountry,
		DomainAgeDays:       doc.DomainAgeDays,
		DisplayName:         doc.DisplayName,
		GravatarName:        doc.GravatarName,
		EmailVerified:       doc.EmailVerified,
		PostedAt:            doc.PostedAt,
		Stack:               doc.Stack,
		Tags:                doc.Tags,
		Lang:                doc.Lang,
		OutreachChannel:     doc.OutreachChannel,
		OutreachAngle:       doc.OutreachAngle,
		OutreachDraft:       doc.OutreachDraft,
		PilotQualified:      doc.PilotQualified,
		PilotWhy:            doc.PilotWhy,
		CompanyType:         doc.CompanyType,
		EnrichSummary:       doc.EnrichSummary,
		GeoConfidence:       doc.GeoConfidence,
		EntityID:            doc.EntityID,
		EntitySightingCount: doc.EntitySightingCount,
		EntitySourceCount:   doc.EntitySourceCount,
	})
	if err != nil {
		return nil, err
	}
	var fields bson.M
	if err := bson.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func leadDocUpsertUpdate(doc LeadDoc) (bson.M, error) {
	set, err := ToLeadDocUpdateBSON(doc)
	if err != nil {
		return nil, err
	}
	return bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			"hash_id":   doc.HashID,
			"ts":        doc.TS,
			"status":    doc.Status,
			"status_at": doc.StatusAt,
		},
	}, nil
}
