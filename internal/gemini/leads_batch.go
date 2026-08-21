package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

const leadBatchSystemPrompt = `You analyze accepted affiliate/iGaming leads for BidShard (self-hosted ad tracker).
For each lead id return geo compliance, ICP classification, and (for high-priority leads) pilot/outreach and enrichment synthesis.
Contacts are masked - never invent PII.

Geo: blocked=true only with credible RU/BY evidence (see geo rules). person_country/company_country ISO alpha-2 or unknown.

ICP: starter | pro | none; hot=true only for clear buyer pain with spend or competitor stack.

Pilot signals (high priority only): spend_budget, competitor_stack, tracker_pain, infra_vps, usdt_ok, buyer_role, high_volume, migration_intent.
pilot_qualified=true when >=3 independent signals.

Enrichment (high priority only): company_type, geo_confidence, summary (2 sentences max).`

const leadBatchICPSystemPrompt = `You analyze accepted affiliate/iGaming leads for BidShard (self-hosted ad tracker).
For each lead id return ICP classification and (for high-priority leads) pilot/outreach and enrichment synthesis.
Contacts are masked - never invent PII. Skip geo compliance (handled elsewhere).

ICP: starter | pro | none; hot=true only for clear buyer pain with spend or competitor stack.

Pilot signals (high priority only): spend_budget, competitor_stack, tracker_pain, infra_vps, usdt_ok, buyer_role, high_volume, migration_intent.
pilot_qualified=true when >=3 independent signals.

Enrichment (high priority only): company_type, geo_confidence, summary (2 sentences max).`

var leadBatchSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "string"},
					"blocked": map[string]any{"type": "boolean"},
					"geo_confidence": map[string]any{
						"type": "string",
						"enum": []any{"high", "medium", "low"},
					},
					"person_country":  map[string]any{"type": "string"},
					"company_country": map[string]any{"type": "string"},
					"company_name":    map[string]any{"type": "string"},
					"registration_signals": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"ru_by_signals": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"geo_why": map[string]any{"type": "string"},
					"icp": map[string]any{
						"type": "string",
						"enum": []any{"starter", "pro", "none"},
					},
					"hot": map[string]any{"type": "boolean"},
					"spend_tier": map[string]any{
						"type": "string",
						"enum": []any{"15k-150k", "unknown"},
					},
					"icp_why": map[string]any{"type": "string"},
					"pilot_signals": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
							"enum": []any{
								"spend_budget", "competitor_stack", "tracker_pain", "infra_vps",
								"usdt_ok", "buyer_role", "high_volume", "migration_intent",
							},
						},
					},
					"pilot_qualified": map[string]any{"type": "boolean"},
					"pilot_why":       map[string]any{"type": "string"},
					"outreach_channel": map[string]any{
						"type": "string",
						"enum": []any{"telegram", "email", "forum", "other"},
					},
					"outreach_angle": map[string]any{"type": "string"},
					"outreach_draft": map[string]any{"type": "string"},
					"company_type": map[string]any{
						"type": "string",
						"enum": []any{"media_buyer", "affiliate_network", "tool_vendor", "agency", "unknown"},
					},
					"enrich_geo_confidence": map[string]any{
						"type": "string",
						"enum": []any{"high", "medium", "low"},
					},
					"enrich_summary": map[string]any{"type": "string"},
				},
				"required": []any{"id", "blocked", "geo_confidence", "icp", "hot", "spend_tier"},
			},
		},
	},
	"required": []any{"items"},
}

var leadBatchICPSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
					"icp": map[string]any{
						"type": "string",
						"enum": []any{"starter", "pro", "none"},
					},
					"hot": map[string]any{"type": "boolean"},
					"spend_tier": map[string]any{
						"type": "string",
						"enum": []any{"15k-150k", "unknown"},
					},
					"icp_why": map[string]any{"type": "string"},
					"pilot_signals": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
							"enum": []any{
								"spend_budget", "competitor_stack", "tracker_pain", "infra_vps",
								"usdt_ok", "buyer_role", "high_volume", "migration_intent",
							},
						},
					},
					"pilot_qualified": map[string]any{"type": "boolean"},
					"pilot_why":       map[string]any{"type": "string"},
					"outreach_channel": map[string]any{
						"type": "string",
						"enum": []any{"telegram", "email", "forum", "other"},
					},
					"outreach_angle": map[string]any{"type": "string"},
					"outreach_draft": map[string]any{"type": "string"},
					"company_type": map[string]any{
						"type": "string",
						"enum": []any{"media_buyer", "affiliate_network", "tool_vendor", "agency", "unknown"},
					},
					"enrich_geo_confidence": map[string]any{
						"type": "string",
						"enum": []any{"high", "medium", "low"},
					},
					"enrich_summary": map[string]any{"type": "string"},
				},
				"required": []any{"id", "icp", "hot", "spend_tier"},
			},
		},
	},
	"required": []any{"items"},
}

type LeadBatchInput struct {
	ID               string   `json:"id"`
	Source           string   `json:"source"`
	Priority         string   `json:"priority"`
	Score            int      `json:"score,omitempty"`
	Snippet          string   `json:"snippet"`
	Contacts         []string `json:"contacts,omitempty"`
	ContactTypes     []string `json:"contact_types,omitempty"`
	Stack            []string `json:"stack,omitempty"`
	Domain           string   `json:"domain,omitempty"`
	RDAPCountry      string   `json:"rdap_country,omitempty"`
	DomainAgeDays    int      `json:"domain_age_days,omitempty"`
	DisplayName      string   `json:"display_name,omitempty"`
	BlockedCountries []string `json:"blocked_countries"`
}

type LeadBatchResult struct {
	HashID         string
	Geo            GeoResult
	ICP            ICPResult
	Engagement     EngagementResult
	Enrichment     EnrichSynthResult
	PilotQualified bool
	PilotTags      []string
	Score          int
	Priority       string
	GeoRejected    bool
	ICPRejected    bool
}

type leadBatchResponse struct {
	Items []leadBatchItem `json:"items"`
}

type leadBatchItem struct {
	ID                  string   `json:"id"`
	Blocked             bool     `json:"blocked"`
	GeoConfidence       string   `json:"geo_confidence"`
	PersonCountry       string   `json:"person_country"`
	CompanyCountry      string   `json:"company_country"`
	CompanyName         string   `json:"company_name"`
	RegistrationSignals []string `json:"registration_signals"`
	RUBYSignals         []string `json:"ru_by_signals"`
	GeoWhy              string   `json:"geo_why"`
	ICP                 string   `json:"icp"`
	Hot                 bool     `json:"hot"`
	SpendTier           string   `json:"spend_tier"`
	ICPWhy              string   `json:"icp_why"`
	PilotSignals        []string `json:"pilot_signals"`
	PilotQualified      bool     `json:"pilot_qualified"`
	PilotWhy            string   `json:"pilot_why"`
	OutreachChannel     string   `json:"outreach_channel"`
	OutreachAngle       string   `json:"outreach_angle"`
	OutreachDraft       string   `json:"outreach_draft"`
	CompanyType         string   `json:"company_type"`
	EnrichGeoConfidence string   `json:"enrich_geo_confidence"`
	EnrichSummary       string   `json:"enrich_summary"`
}

func (c *Client) AnalyzeLeadBatch(ctx context.Context, items []LeadBatchInput, geoClassify bool) ([]LeadBatchResult, error) {
	if len(items) == 0 {
		return nil, nil
	}

	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return nil, err
	}

	sysPrompt := leadBatchSystemPrompt
	schema := leadBatchSchema
	if !geoClassify {
		sysPrompt = leadBatchICPSystemPrompt
		schema = leadBatchICPSchema
	}

	prompt := "Analyze each accepted lead. Return one output item per input id.\n\n" + string(payload)
	parsed, err := classifyJSON[leadBatchResponse](c, ctx, PriorityHigh, sysPrompt, prompt, schema)
	if err != nil {
		return nil, err
	}

	out := make([]LeadBatchResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, leadBatchResultFromItem(item, geoClassify))
	}
	return out, nil
}

func leadBatchResultFromItem(item leadBatchItem, geoClassify bool) LeadBatchResult {
	var geo GeoResult
	if geoClassify {
		geo = normalizeGeoResult(geoResponse{
			Blocked:             item.Blocked,
			Confidence:          item.GeoConfidence,
			PersonCountry:       item.PersonCountry,
			CompanyCountry:      item.CompanyCountry,
			CompanyName:         item.CompanyName,
			RegistrationSignals: item.RegistrationSignals,
			RUBYSignals:         item.RUBYSignals,
			Why:                 item.GeoWhy,
		})
	}
	engage := EngagementResult{
		PilotSignals:    normalizePilotSignals(item.PilotSignals),
		PilotQualified:  item.PilotQualified,
		PilotWhy:        strings.TrimSpace(item.PilotWhy),
		OutreachChannel: normalizeOutreachChannel(item.OutreachChannel),
		OutreachAngle:   strings.TrimSpace(item.OutreachAngle),
		OutreachDraft:   strings.TrimSpace(item.OutreachDraft),
	}
	qualified, tags := ApplyEngagementPilot(engage)
	return LeadBatchResult{
		HashID: strings.TrimSpace(item.ID),
		Geo:    geo,
		ICP: ICPResult{
			ICP:       normalizeICP(item.ICP),
			Hot:       item.Hot,
			SpendTier: normalizeSpendTier(item.SpendTier),
			Why:       strings.TrimSpace(item.ICPWhy),
		},
		Engagement: engage,
		Enrichment: EnrichSynthResult{
			CompanyType:   normalizeCompanyType(item.CompanyType),
			GeoConfidence: normalizeConfidence(item.EnrichGeoConfidence),
			Summary:       strings.TrimSpace(item.EnrichSummary),
		},
		PilotQualified: qualified,
		PilotTags:      tags,
	}
}

func normalizePilotSignals(signals []string) []string {
	out := make([]string, 0, len(signals))
	seen := make(map[string]struct{}, len(signals))
	for _, sig := range signals {
		sig = normalizePilotSignal(sig)
		if sig == "" {
			continue
		}
		if _, ok := seen[sig]; ok {
			continue
		}
		seen[sig] = struct{}{}
		out = append(out, sig)
	}
	return out
}
