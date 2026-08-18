package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/geo"
)

const geoSystemPrompt = `You are a geo-compliance analyst for an affiliate/iGaming lead parser (BidShard).
Goal: detect if the lead operator or their company is based in or legally registered in blocked countries (Russia/Belarus by default).

Analyze:
1. Company legal registration — ООО/АО/ИП/UNP, INN/KPP, "registered in", юрлицо, corporate domain TLD, legal address
2. Operator location — city/region/timezone, phone codes, bank names (Sberbank, Tinkoff, Belarusbank), Mir card, RUB/BYN pricing
3. Language/context — RU/BY market focus, Cyrillic business identity without international footprint

Rules:
- blocked=true only when there is credible evidence the person OR company is in a blocked country
- confidence=high: explicit location/registration (e.g. "ООО из Москвы", +7 phone, .ru corporate email)
- confidence=medium: strong indirect signals (Belarus bank, Minsk office, BY legal form)
- confidence=low: weak/ambiguous Cyrillic or generic Eastern Europe hints — do NOT set blocked=true
- person_country / company_country: ISO 3166-1 alpha-2 (RU, BY, US, ...) or "unknown"
- List concrete signals in ru_by_signals and registration_signals arrays
- Contacts may be masked; never invent PII`

var geoSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"blocked": map[string]any{"type": "boolean"},
		"confidence": map[string]any{
			"type": "string",
			"enum": []any{"high", "medium", "low"},
		},
		"person_country": map[string]any{
			"type":        "string",
			"description": "ISO 3166-1 alpha-2 or unknown",
		},
		"company_country": map[string]any{
			"type":        "string",
			"description": "Country of legal entity registration, ISO alpha-2 or unknown",
		},
		"company_name": map[string]any{"type": "string"},
		"registration_signals": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "Evidence about company registration jurisdiction",
		},
		"ru_by_signals": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"why": map[string]any{"type": "string"},
	},
	"required": []any{"blocked", "confidence", "why"},
}

type GeoResult struct {
	Blocked             bool
	Confidence          string
	PersonCountry       string
	CompanyCountry      string
	CompanyName         string
	RegistrationSignals []string
	RUBYSignals         []string
	Why                 string
}

type geoResponse struct {
	Blocked             bool     `json:"blocked"`
	Confidence          string   `json:"confidence"`
	PersonCountry       string   `json:"person_country"`
	CompanyCountry      string   `json:"company_country"`
	CompanyName         string   `json:"company_name"`
	RegistrationSignals []string `json:"registration_signals"`
	RUBYSignals         []string `json:"ru_by_signals"`
	Why                 string   `json:"why"`
}

func (c *Client) ClassifyGeo(ctx context.Context, text string, contacts []string, blockedCountries []string) (GeoResult, error) {
	text = strings.TrimSpace(text)
	if text == "" && len(contacts) == 0 {
		return GeoResult{Confidence: "low"}, nil
	}

	payload, err := json.Marshal(map[string]any{
		"blocked_countries": blockedCountries,
		"snippet":           truncate(text, 2500),
		"contacts":          contacts,
	})
	if err != nil {
		return GeoResult{}, err
	}

	userPrompt := fmt.Sprintf(
		"Blocked countries (ISO alpha-2): %s\n\nClassify geo and company registration:\n\n%s",
		formatBlockedList(blockedCountries),
		string(payload),
	)

	raw, err := c.generateJSON(ctx, geoSystemPrompt, userPrompt, geoSchema)
	if err != nil {
		return GeoResult{}, err
	}

	var parsed geoResponse
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return GeoResult{}, err
	}

	return normalizeGeoResult(parsed), nil
}

func normalizeGeoResult(parsed geoResponse) GeoResult {
	return GeoResult{
		Blocked:             parsed.Blocked,
		Confidence:          normalizeConfidence(parsed.Confidence),
		PersonCountry:       normalizeCountryCode(parsed.PersonCountry),
		CompanyCountry:      normalizeCountryCode(parsed.CompanyCountry),
		CompanyName:         strings.TrimSpace(parsed.CompanyName),
		RegistrationSignals: append([]string(nil), parsed.RegistrationSignals...),
		RUBYSignals:         append([]string(nil), parsed.RUBYSignals...),
		Why:                 strings.TrimSpace(parsed.Why),
	}
}

func normalizeConfidence(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "high", "medium":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "low"
	}
}

func normalizeCountryCode(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" || v == "UNKNOWN" || v == "N/A" {
		return "unknown"
	}
	if len(v) == 2 {
		return v
	}
	return "unknown"
}

func formatBlockedList(blocked []string) string {
	if len(blocked) == 0 {
		return "RU, BY"
	}
	parts := make([]string, 0, len(blocked))
	for _, c := range blocked {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c != "" {
			parts = append(parts, c)
		}
	}
	if len(parts) == 0 {
		return "RU, BY"
	}
	return strings.Join(parts, ", ")
}

// ShouldReject applies geo result against configured block list.
func (r GeoResult) ShouldReject(blockedCountries []string) bool {
	if r.Confidence == "low" {
		return false
	}
	if r.Blocked && (r.Confidence == "high" || r.Confidence == "medium") {
		return true
	}
	if geo.IsBlockedCountry(r.PersonCountry, blockedCountries) {
		return true
	}
	if geo.IsBlockedCountry(r.CompanyCountry, blockedCountries) {
		return true
	}
	return false
}

func (r GeoResult) Detail() string {
	var parts []string
	if r.CompanyName != "" {
		parts = append(parts, "company="+r.CompanyName)
	}
	if r.CompanyCountry != "unknown" && r.CompanyCountry != "" {
		parts = append(parts, "reg="+r.CompanyCountry)
	}
	if r.PersonCountry != "unknown" && r.PersonCountry != "" {
		parts = append(parts, "person="+r.PersonCountry)
	}
	if len(r.RegistrationSignals) > 0 {
		parts = append(parts, "reg_signals="+strings.Join(r.RegistrationSignals, "; "))
	}
	if len(r.RUBYSignals) > 0 {
		parts = append(parts, "ru_by="+strings.Join(r.RUBYSignals, "; "))
	}
	if r.Why != "" {
		parts = append(parts, r.Why)
	}
	if len(parts) == 0 {
		return "gemini geo block (" + r.Confidence + ")"
	}
	return strings.Join(parts, " | ")
}
