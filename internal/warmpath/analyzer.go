package warmpath

import (
	"context"

	"github.com/bidshard/parser/internal/gemini"
)

// LeadBatchAnalyzer runs deferred Gemini geo/ICP batch analysis.
// *gemini.Client implements this; tests inject stubs without HTTP.
type LeadBatchAnalyzer interface {
	AnalyzeLeadBatchOpts(ctx context.Context, items []gemini.LeadBatchInput, opts gemini.LeadBatchOptions) ([]gemini.LeadBatchResult, error)
}
