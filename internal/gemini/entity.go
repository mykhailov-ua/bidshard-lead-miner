package gemini

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/pretty"
)

const entityClassifySystemPrompt = `You validate whether multiple forum/Telegram/web sightings belong to the same affiliate media buyer for BidShard tracker sales.
Focus on affiliate tracker pain (Voluum, Keitaro, Binom, postback, self-hosted tracker, migration).
Ignore CPA accounting/tax contexts, affiliate program recruiting posts, and support staff venting without buying authority.
Decision makers: founders, heads of acquisition, media buyers with spend. Not rank-and-file support or affiliate managers recruiting publishers.
Return split_recommended=true when sightings likely describe different people or incompatible roles (e.g. support@ vs founder@ on same domain, or recruiting vs buyer pain).
buyer_intent: hot (clear active tracker pain + buyer role), warm (some pain), cold (noise/recruiting), none (unrelated).`

var entityClassifySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"same_actor":       map[string]any{"type": "boolean"},
		"actor_confidence": map[string]any{"type": "number"},
		"unified_pain":     map[string]any{"type": "string"},
		"buyer_intent": map[string]any{
			"type": "string",
			"enum": []any{"hot", "warm", "cold", "none"},
		},
		"split_recommended": map[string]any{"type": "boolean"},
		"why":               map[string]any{"type": "string"},
	},
	"required": []any{"same_actor", "actor_confidence", "unified_pain", "buyer_intent", "split_recommended"},
}

// EntitySightingInput is one observation sent to ClassifyEntity.
type EntitySightingInput struct {
	Source          string
	PostedAt        time.Time
	Matched         []string
	Snippet         string
	ContactsSummary string
}

// EntityClassifyInput groups sightings for one entity graph node.
type EntityClassifyInput struct {
	EntityID  string
	Sightings []EntitySightingInput
}

// EntityClassifyResult is Gemini output for entity-level validation.
type EntityClassifyResult struct {
	SameActor        bool
	ActorConfidence  float64
	UnifiedPain      string
	BuyerIntent      string
	SplitRecommended bool
	Why              string
}

type entityClassifyResponse struct {
	SameActor        bool    `json:"same_actor"`
	ActorConfidence  float64 `json:"actor_confidence"`
	UnifiedPain      string  `json:"unified_pain"`
	BuyerIntent      string  `json:"buyer_intent"`
	SplitRecommended bool    `json:"split_recommended"`
	Why              string  `json:"why"`
}

const maxEntityClassifySightings = 5

// ClassifyEntity scores same-actor confidence and unified pain across sightings.
func (c *Client) ClassifyEntity(ctx context.Context, in EntityClassifyInput) (EntityClassifyResult, error) {
	in.EntityID = strings.TrimSpace(in.EntityID)
	if in.EntityID == "" || len(in.Sightings) == 0 {
		return EntityClassifyResult{BuyerIntent: "none"}, nil
	}
	sightings := in.Sightings
	if len(sightings) > maxEntityClassifySightings {
		sightings = sightings[:maxEntityClassifySightings]
	}
	prompt := buildEntityClassifyPrompt(in.EntityID, sightings)
	parsed, err := classifyJSON[entityClassifyResponse](c, ctx, PriorityHigh, entityClassifySystemPrompt, prompt, entityClassifySchema)
	if err != nil {
		return EntityClassifyResult{}, err
	}
	return entityClassifyResultFromResponse(parsed), nil
}

func buildEntityClassifyPrompt(entityID string, sightings []EntitySightingInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Entity id: %s\nSightings (%d):\n", entityID, len(sightings))
	for i, s := range sightings {
		fmt.Fprintf(&b, "\n--- sighting %d ---\n", i+1)
		fmt.Fprintf(&b, "source: %s\n", strings.TrimSpace(s.Source))
		if !s.PostedAt.IsZero() {
			fmt.Fprintf(&b, "posted_at: %s\n", s.PostedAt.UTC().Format(time.RFC3339))
		}
		if len(s.Matched) > 0 {
			fmt.Fprintf(&b, "matched: %s\n", strings.Join(s.Matched, ", "))
		}
		if cs := strings.TrimSpace(s.ContactsSummary); cs != "" {
			fmt.Fprintf(&b, "contacts: %s\n", cs)
		}
		if snip := pretty.Truncate(strings.TrimSpace(s.Snippet), 600); snip != "" {
			fmt.Fprintf(&b, "snippet: %s\n", snip)
		}
	}
	return b.String()
}

func entityClassifyResultFromResponse(parsed entityClassifyResponse) EntityClassifyResult {
	conf := parsed.ActorConfidence
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	return EntityClassifyResult{
		SameActor:        parsed.SameActor,
		ActorConfidence:  conf,
		UnifiedPain:      strings.TrimSpace(parsed.UnifiedPain),
		BuyerIntent:      normalizeBuyerIntent(parsed.BuyerIntent),
		SplitRecommended: parsed.SplitRecommended,
		Why:              strings.TrimSpace(parsed.Why),
	}
}

func normalizeBuyerIntent(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "hot", "warm", "cold":
		return strings.ToLower(v)
	default:
		return "none"
	}
}

// EntitySightingInputsFromSnippets builds classify inputs from raw snippets (tests).
func EntitySightingInputsFromSnippets(sources []string, snippets []string, matched [][]string) []EntitySightingInput {
	n := len(snippets)
	if len(sources) < n {
		n = len(sources)
	}
	out := make([]EntitySightingInput, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, EntitySightingInput{
			Source:  sources[i],
			Snippet: snippets[i],
			Matched: matched[i],
		})
	}
	return out
}
