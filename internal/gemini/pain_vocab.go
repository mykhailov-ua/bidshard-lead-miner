package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

// PainVocabDiff reuses keyword diff schema for manual-approve pain vocabulary mining.
type PainVocabDiff = KeywordDiff

const painVocabSystem = `You propose keyword registry additions from affiliate buyer pain phrases and false-negative junk snippets.
Output JSON only. Suggest additions to keywords (intent/pain) or hard_reject (anti-ICP).
Weight 15-25 for intent/pain phrases. Include negative examples in summary when uncertain.`

// BuildPainVocabDiff proposes keyword candidates from entity pains and false-negative junk.
func (c *Client) BuildPainVocabDiff(ctx context.Context, unifiedPains, falseNegativeSamples []string) (PainVocabDiff, error) {
	payload, err := json.Marshal(map[string]any{
		"unified_pains":           unifiedPains,
		"false_negative_snippets": falseNegativeSamples,
	})
	if err != nil {
		return PainVocabDiff{}, err
	}
	diff, err := classifyJSON[PainVocabDiff](c, ctx, PriorityLow, painVocabSystem,
		"Propose pain vocabulary keyword diff for manual approve:\n\n"+string(payload), keywordDiffSchema)
	if err != nil {
		return PainVocabDiff{}, err
	}
	diff.Status = "pending"
	diff.Summary = strings.TrimSpace(diff.Summary)
	return diff, nil
}
