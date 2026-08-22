package admin

import "context"

// LeadExplainer returns a one-line inbox summary for a lead.
type LeadExplainer interface {
	Explain(ctx context.Context, hashID string) (string, error)
}
