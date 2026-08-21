package sink

import (
	"log/slog"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/model"
)

// WebhookHeatGate filters CRM webhook delivery by entity heat tier.
type WebhookHeatGate struct {
	MinTier string
}

// Allows reports whether lead heat_tier meets the configured minimum.
func (g WebhookHeatGate) Allows(lead model.Lead) bool {
	if g.MinTier == "" {
		return true
	}
	return entity.HeatTierMeetsMin(lead.HeatTier, g.MinTier)
}

func (c *WebhookClient) allowsHeat(lead model.Lead) bool {
	if c == nil {
		return true
	}
	return c.heatGate.Allows(lead)
}

func (c *WebhookClient) logHeatSkip(lead model.Lead) {
	slog.Debug("crm webhook skipped heat gate",
		"hash_id", lead.HashID,
		"heat_tier", lead.HeatTier,
		"min_tier", c.heatGate.MinTier,
	)
}
