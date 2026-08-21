package gemini

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bidshard/parser/internal/pretty"
)

const engageSystemPrompt = `You evaluate accepted affiliate/iGaming leads for BidShard (self-hosted ad tracker).
Audience: media buyers outside Russia/Belarus with tracker pain or migration intent.

Pilot checklist - emit only signals clearly supported by the snippet (max 8):
- spend_budget: explicit spend/budget ($5k+, enterprise, high spend) or spend_tier from context
- competitor_stack: named tracker (Voluum, Keitaro, RedTrack, Binom, etc.)
- tracker_pain: postback/click loss/billing/down/broken tracker
- infra_vps: self-hosted, VPS, Docker, Hetzner, AWS infra
- usdt_ok: USDT/crypto payment willingness
- buyer_role: founder, CEO, head of acquisition/media - decision maker
- high_volume: scale signals (100+ FTD, high clicks, scaling)
- migration_intent: switching/migrating/alternative to current tracker

pilot_qualified=true only when at least 3 independent checklist signals are present.

Outreach:
- outreach_channel: best first-touch channel given available contact types (telegram > email > forum > other)
- outreach_angle: one sentence hook referencing their pain/stack (no PII, no fake familiarity)
- outreach_draft: 1-2 short sentences for first message; professional, specific to their pain; no links unless snippet mentions one

Contacts are masked. Never invent PII.`

var engageSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
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
	},
	"required": []any{"pilot_signals", "pilot_qualified", "outreach_channel", "outreach_angle", "outreach_draft"},
}

type EngagementInput struct {
	Text         string
	Stack        []string
	ICP          string
	SpendTier    string
	Hot          bool
	ContactTypes []string
	Source       string
}

type EngagementResult struct {
	PilotSignals    []string
	PilotQualified  bool
	PilotWhy        string
	OutreachChannel string
	OutreachAngle   string
	OutreachDraft   string
}

type engagementResponse struct {
	PilotSignals    []string `json:"pilot_signals"`
	PilotQualified  bool     `json:"pilot_qualified"`
	PilotWhy        string   `json:"pilot_why"`
	OutreachChannel string   `json:"outreach_channel"`
	OutreachAngle   string   `json:"outreach_angle"`
	OutreachDraft   string   `json:"outreach_draft"`
}

var pilotSignalTags = map[string]string{
	"spend_budget":     "pilot-spend-budget",
	"competitor_stack": "pilot-competitor-stack",
	"tracker_pain":     "pilot-tracker-pain",
	"infra_vps":        "pilot-infra-vps",
	"usdt_ok":          "pilot-usdt-ok",
	"buyer_role":       "pilot-buyer-role",
	"high_volume":      "pilot-high-volume",
	"migration_intent": "pilot-migration-intent",
}

func (c *Client) ClassifyEngagement(ctx context.Context, in EngagementInput) (EngagementResult, error) {
	payload, err := json.Marshal(map[string]any{
		"snippet":       pretty.Truncate(strings.TrimSpace(in.Text), 2500),
		"stack":         in.Stack,
		"icp":           in.ICP,
		"spend_tier":    in.SpendTier,
		"hot":           in.Hot,
		"contact_types": in.ContactTypes,
		"source":        in.Source,
	})
	if err != nil {
		return EngagementResult{}, err
	}

	parsed, err := classifyJSON[engagementResponse](c, ctx, PriorityHigh, engageSystemPrompt,
		"Classify pilot signals and draft outreach for this accepted lead:\n\n"+string(payload),
		engageSchema)
	if err != nil {
		return EngagementResult{}, err
	}

	signals := make([]string, 0, len(parsed.PilotSignals))
	seen := make(map[string]struct{}, len(parsed.PilotSignals))
	for _, sig := range parsed.PilotSignals {
		sig = normalizePilotSignal(sig)
		if sig == "" {
			continue
		}
		if _, ok := seen[sig]; ok {
			continue
		}
		seen[sig] = struct{}{}
		signals = append(signals, sig)
	}

	return EngagementResult{
		PilotSignals:    signals,
		PilotQualified:  parsed.PilotQualified,
		PilotWhy:        strings.TrimSpace(parsed.PilotWhy),
		OutreachChannel: normalizeOutreachChannel(parsed.OutreachChannel),
		OutreachAngle:   strings.TrimSpace(parsed.OutreachAngle),
		OutreachDraft:   strings.TrimSpace(parsed.OutreachDraft),
	}, nil
}

func normalizePilotSignal(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if _, ok := pilotSignalTags[v]; ok {
		return v
	}
	return ""
}

func normalizeOutreachChannel(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "telegram", "email", "forum":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "other"
	}
}

// ApplyEngagementPilot maps LLM pilot signals to legacy pilot tags.
// Treat qualified when LLM says so OR >=3 independent signals; empty signals -> pilot-nurture.
func ApplyEngagementPilot(r EngagementResult) (qualified bool, tags []string) {
	seen := make(map[string]struct{}, len(r.PilotSignals))
	for _, sig := range r.PilotSignals {
		tag, ok := pilotSignalTags[sig]
		if !ok {
			continue
		}
		if _, dup := seen[tag]; dup {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}

	qualified = r.PilotQualified || len(tags) >= 3
	if qualified {
		return true, append([]string{"pilot-qualified"}, tags...)
	}
	if len(tags) == 0 {
		// No signals: nurture bucket for warm-path re-review, not hard reject.
		return false, []string{"pilot-nurture"}
	}
	return false, append([]string{"pilot-nurture"}, tags...)
}
