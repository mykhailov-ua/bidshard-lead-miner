package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/bidshard/parser/internal/pretty"
)

const leadBatchCoreSystemPrompt = `You analyze accepted affiliate/iGaming leads for BidShard (self-hosted ad tracker).
Return geo compliance (when requested), ICP, and pilot checklist signals only. No outreach drafts.
Contacts are masked - never invent PII.

Geo: blocked=true only with credible RU/BY evidence. person_country/company_country ISO alpha-2 or unknown.

ICP: starter | pro | none; hot=true only for clear buyer pain with spend or competitor stack.

Pilot signals: spend_budget, competitor_stack, tracker_pain, infra_vps, usdt_ok, buyer_role, high_volume, migration_intent.
pilot_qualified=true when >=3 independent signals.`

const leadBatchCoreICPSystemPrompt = `You analyze accepted affiliate/iGaming leads for BidShard (self-hosted ad tracker).
Return ICP and pilot checklist signals only. Skip geo. No outreach drafts.
Contacts are masked - never invent PII.

ICP: starter | pro | none; hot=true only for clear buyer pain with spend or competitor stack.

Pilot signals: spend_budget, competitor_stack, tracker_pain, infra_vps, usdt_ok, buyer_role, high_volume, migration_intent.
pilot_qualified=true when >=3 independent signals.`

const leadBatchEngageSystemPrompt = `Draft outreach for High-priority accepted affiliate/iGaming leads (BidShard tracker).
Contacts are masked - never invent PII.

Outreach channel rules: pick from contact_types only.
Prefer email for supply/lander/ads_txt/tgweb; telegram for forum/reddit when both exist.
outreach_draft: max 2 short sentences. outreach_subject: max 60 chars when email.

Enrichment: company_type, geo_confidence, summary (1 sentence max).`

var pilotSignalEnum = []any{
	"spend_budget", "competitor_stack", "tracker_pain", "infra_vps",
	"usdt_ok", "buyer_role", "high_volume", "migration_intent",
}

func leadBatchCoreSchema(geoClassify bool) map[string]any {
	required := []any{"id", "icp", "hot", "spend_tier"}
	if geoClassify {
		required = []any{"id", "blocked", "geo_confidence", "icp", "hot", "spend_tier"}
	}
	props := map[string]any{
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
		"hot":        map[string]any{"type": "boolean"},
		"spend_tier": map[string]any{"type": "string", "enum": []any{"15k-150k", "unknown"}},
		"icp_why":    map[string]any{"type": "string"},
		"pilot_signals": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string", "enum": pilotSignalEnum},
		},
		"pilot_qualified": map[string]any{"type": "boolean"},
		"pilot_why":       map[string]any{"type": "string"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		},
		"required": []any{"items"},
	}
}

var leadBatchEngageSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
					"outreach_channel": map[string]any{
						"type": "string",
						"enum": []any{"telegram", "email", "forum", "other"},
					},
					"outreach_subject": map[string]any{"type": "string"},
					"outreach_angle":   map[string]any{"type": "string"},
					"outreach_draft":   map[string]any{"type": "string"},
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
				"required": []any{"id", "outreach_channel", "outreach_angle", "outreach_draft"},
			},
		},
	},
	"required": []any{"items"},
}

// LeadBatchOptions controls warm-path Gemini batch phases (RPM-aware).
type LeadBatchOptions struct {
	GeoClassify     bool
	Engage          bool
	Enrich          bool
	EngageBatchSize int // 0 = default 4
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
	OutreachSubject     string   `json:"outreach_subject"`
	OutreachAngle       string   `json:"outreach_angle"`
	OutreachDraft       string   `json:"outreach_draft"`
	CompanyType         string   `json:"company_type"`
	EnrichGeoConfidence string   `json:"enrich_geo_confidence"`
	EnrichSummary       string   `json:"enrich_summary"`
}

// AnalyzeLeadBatch runs deferred warm-path analysis (legacy: engage+enrich enabled).
func (c *Client) AnalyzeLeadBatch(ctx context.Context, items []LeadBatchInput, geoClassify bool) ([]LeadBatchResult, error) {
	return c.AnalyzeLeadBatchOpts(ctx, items, LeadBatchOptions{
		GeoClassify: geoClassify,
		Engage:      true,
		Enrich:      true,
	})
}

// AnalyzeLeadBatchOpts uses a core pass (ICP/geo/pilot) then optional engage pass for High leads only.
func (c *Client) AnalyzeLeadBatchOpts(ctx context.Context, items []LeadBatchInput, opts LeadBatchOptions) ([]LeadBatchResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	core, err := c.analyzeLeadBatchWithRetry(ctx, items, opts, c.analyzeLeadBatchCoreOnce)
	if err != nil {
		return nil, err
	}
	if !opts.Engage && !opts.Enrich {
		return core, nil
	}
	high := filterHighPriorityInputs(items)
	if len(high) == 0 {
		return core, nil
	}
	chunkSize := opts.EngageBatchSize
	if chunkSize <= 0 {
		chunkSize = 4
	}
	byID := indexLeadBatchInputs(items)
	for i := 0; i < len(high); i += chunkSize {
		end := i + chunkSize
		if end > len(high) {
			end = len(high)
		}
		chunk := high[i:end]
		engageItems, err := c.analyzeLeadBatchWithRetry(ctx, chunk, opts, func(ctx context.Context, batch []LeadBatchInput, o LeadBatchOptions) ([]LeadBatchResult, error) {
			return c.analyzeLeadBatchEngageOnce(ctx, batch, o)
		})
		if err != nil {
			return nil, err
		}
		mergeEngageResults(core, engageItems, byID)
	}
	return core, nil
}

type leadBatchOnceFn func(context.Context, []LeadBatchInput, LeadBatchOptions) ([]LeadBatchResult, error)

func (c *Client) analyzeLeadBatchWithRetry(ctx context.Context, items []LeadBatchInput, opts LeadBatchOptions, fn leadBatchOnceFn) ([]LeadBatchResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if len(items) == 1 {
		return fn(ctx, items, opts)
	}
	out, err := fn(ctx, items, opts)
	if err == nil {
		return out, nil
	}
	if !isBatchRetryableError(err) {
		return nil, err
	}
	mid := len(items) / 2
	left, err1 := c.analyzeLeadBatchWithRetry(ctx, items[:mid], opts, fn)
	if err1 != nil {
		return nil, err1
	}
	right, err2 := c.analyzeLeadBatchWithRetry(ctx, items[mid:], opts, fn)
	if err2 != nil {
		return nil, err2
	}
	return append(left, right...), nil
}

func (c *Client) analyzeLeadBatchCoreOnce(ctx context.Context, items []LeadBatchInput, opts LeadBatchOptions) ([]LeadBatchResult, error) {
	payload, err := marshalLeadBatchPayload(items)
	if err != nil {
		return nil, err
	}
	sysPrompt := leadBatchCoreSystemPrompt
	if !opts.GeoClassify {
		sysPrompt = leadBatchCoreICPSystemPrompt
	}
	schema := leadBatchCoreSchema(opts.GeoClassify)
	prompt := "Analyze each lead. Return one output item per input id.\n\n" + string(payload)
	parsed, err := classifyJSON[leadBatchResponse](c, ctx, PriorityHigh, sysPrompt, prompt, schema)
	if err != nil {
		return nil, err
	}
	byID := indexLeadBatchInputs(items)
	out := make([]LeadBatchResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, leadBatchResultFromItem(item, opts.GeoClassify, byID[strings.TrimSpace(item.ID)]))
	}
	return out, nil
}

func (c *Client) analyzeLeadBatchEngageOnce(ctx context.Context, items []LeadBatchInput, opts LeadBatchOptions) ([]LeadBatchResult, error) {
	payload, err := marshalLeadBatchPayload(items)
	if err != nil {
		return nil, err
	}
	prompt := "Draft outreach for each High-priority lead id.\n\n" + string(payload)
	parsed, err := classifyJSON[leadBatchResponse](c, ctx, PriorityHigh, leadBatchEngageSystemPrompt, prompt, leadBatchEngageSchema)
	if err != nil {
		return nil, err
	}
	byID := indexLeadBatchInputs(items)
	out := make([]LeadBatchResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		base := leadBatchResultFromItem(leadBatchItem{ID: item.ID}, false, byID[strings.TrimSpace(item.ID)])
		base.Engagement = EngagementResult{
			OutreachChannel: normalizeOutreachChannel(item.OutreachChannel),
			OutreachSubject: strings.TrimSpace(item.OutreachSubject),
			OutreachAngle:   strings.TrimSpace(item.OutreachAngle),
			OutreachDraft:   strings.TrimSpace(item.OutreachDraft),
		}
		base.Engagement = ReconcileEngagement(base.Engagement, byID[strings.TrimSpace(item.ID)].ContactTypes, byID[strings.TrimSpace(item.ID)].Source)
		if opts.Enrich {
			base.Enrichment = EnrichSynthResult{
				CompanyType:   normalizeCompanyType(item.CompanyType),
				GeoConfidence: normalizeConfidence(item.EnrichGeoConfidence),
				Summary:       strings.TrimSpace(item.EnrichSummary),
			}
		}
		out = append(out, base)
	}
	return out, nil
}

func filterHighPriorityInputs(items []LeadBatchInput) []LeadBatchInput {
	out := make([]LeadBatchInput, 0, len(items))
	for _, in := range items {
		if strings.EqualFold(strings.TrimSpace(in.Priority), "high") {
			out = append(out, in)
		}
	}
	return out
}

func indexLeadBatchInputs(items []LeadBatchInput) map[string]LeadBatchInput {
	byID := make(map[string]LeadBatchInput, len(items))
	for _, in := range items {
		byID[in.ID] = in
	}
	return byID
}

func mergeEngageResults(core []LeadBatchResult, engage []LeadBatchResult, byID map[string]LeadBatchInput) {
	engageByID := make(map[string]LeadBatchResult, len(engage))
	for _, r := range engage {
		engageByID[r.HashID] = r
	}
	for i := range core {
		id := core[i].HashID
		ex, ok := engageByID[id]
		if !ok {
			continue
		}
		in := byID[id]
		eng := ReconcileEngagement(ex.Engagement, in.ContactTypes, in.Source)
		core[i].Engagement = eng
		if ex.Enrichment.CompanyType != "" || ex.Enrichment.Summary != "" {
			core[i].Enrichment = ex.Enrichment
		}
	}
}

func marshalLeadBatchPayload(items []LeadBatchInput) ([]byte, error) {
	rows := make([]LeadBatchInput, 0, len(items))
	for _, in := range items {
		row := in
		row.Snippet = pretty.Truncate(strings.TrimSpace(in.Snippet), 1500)
		rows = append(rows, row)
	}
	return json.Marshal(map[string]any{"items": rows})
}

func isBatchRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrOutputTruncated) {
		return true
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "decode model json") {
		return false
	}
	return strings.Contains(msg, "unexpected end") || strings.Contains(msg, "unexpected eof")
}

func leadBatchResultFromItem(item leadBatchItem, geoClassify bool, in LeadBatchInput) LeadBatchResult {
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
		OutreachSubject: strings.TrimSpace(item.OutreachSubject),
		OutreachAngle:   strings.TrimSpace(item.OutreachAngle),
		OutreachDraft:   strings.TrimSpace(item.OutreachDraft),
	}
	engage = ReconcileEngagement(engage, in.ContactTypes, in.Source)
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
