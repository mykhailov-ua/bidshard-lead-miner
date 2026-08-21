package gemini

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bidshard/parser/internal/pretty"
)

const enrichSynthSystem = `You synthesize a lead enrichment profile for BidShard sales ops.
Input: snippet (masked contacts), RDAP/DNS stack signals, optional geo from text/RDAP.
Output concise structured profile - no PII, no invented company names.
company_type: media_buyer | affiliate_network | tool_vendor | agency | unknown
geo_confidence: high | medium | low - agreement between snippet geo hints and RDAP country
summary: 2 sentences max on who they likely are and why they matter for tracker sales`

var enrichSynthSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"company_type": map[string]any{
			"type": "string",
			"enum": []any{"media_buyer", "affiliate_network", "tool_vendor", "agency", "unknown"},
		},
		"geo_confidence": map[string]any{
			"type": "string",
			"enum": []any{"high", "medium", "low"},
		},
		"summary": map[string]any{"type": "string"},
	},
	"required": []any{"company_type", "geo_confidence", "summary"},
}

type EnrichSynthInput struct {
	Snippet        string
	Source         string
	Domain         string
	RDAPCountry    string
	DomainAgeDays  int
	Stack          []string
	DisplayName    string
	GeoCountry     string
	CompanyCountry string
	ICP            string
}

type EnrichSynthResult struct {
	CompanyType   string
	GeoConfidence string
	Summary       string
}

type enrichSynthResponse struct {
	CompanyType   string `json:"company_type"`
	GeoConfidence string `json:"geo_confidence"`
	Summary       string `json:"summary"`
}

func (c *Client) SynthesizeEnrichment(ctx context.Context, in EnrichSynthInput) (EnrichSynthResult, error) {
	payload, err := json.Marshal(map[string]any{
		"snippet":         pretty.Truncate(strings.TrimSpace(in.Snippet), 2000),
		"source":          in.Source,
		"domain":          in.Domain,
		"rdap_country":    in.RDAPCountry,
		"domain_age_days": in.DomainAgeDays,
		"stack":           in.Stack,
		"display_name":    in.DisplayName,
		"geo_country":     in.GeoCountry,
		"company_country": in.CompanyCountry,
		"icp":             in.ICP,
	})
	if err != nil {
		return EnrichSynthResult{}, err
	}

	parsed, err := classifyJSON[enrichSynthResponse](c, ctx, PriorityHigh, enrichSynthSystem,
		"Synthesize enrichment profile:\n\n"+string(payload), enrichSynthSchema)
	if err != nil {
		return EnrichSynthResult{}, err
	}

	return EnrichSynthResult{
		CompanyType:   normalizeCompanyType(parsed.CompanyType),
		GeoConfidence: normalizeConfidence(parsed.GeoConfidence),
		Summary:       strings.TrimSpace(parsed.Summary),
	}, nil
}

func normalizeCompanyType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "media_buyer", "affiliate_network", "tool_vendor", "agency":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "unknown"
	}
}
