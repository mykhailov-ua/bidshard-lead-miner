package warmpath

import (
	"context"

	"github.com/bidshard/parser/internal/gemini"
)

// LeadBatchAnalyzer runs deferred Gemini geo/ICP batch analysis.
// *gemini.Client implements this; tests inject stubs without HTTP.
type LeadBatchAnalyzer interface {
	AnalyzeLeadBatch(ctx context.Context, items []gemini.LeadBatchInput, geoClassify bool) ([]gemini.LeadBatchResult, error)
}
