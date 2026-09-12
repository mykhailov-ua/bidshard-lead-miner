package scoring

import (
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/classify"
	"github.com/bidshard/parser/internal/validate"
)

const (
	bidShardBucketInfraBoost      = 40
	bidShardBucketCloakBoost      = 20
	bidShardCommercialIntentBoost = 40
	bidShardVolumeBoost           = 30
)

var volumeSignalRe = regexp.MustCompile(`(?i)(?:\d{2,}\s*k\s*(?:click|clicks)|\d+\s*(?:к|k)\s*клик)`)

// HasVolumeSignal reports high click volume in buyer pain text (H14).
func HasVolumeSignal(text string) bool {
	return volumeSignalRe.MatchString(strings.TrimSpace(text))
}

// BidShardPainBoost applies H10 bucket-aligned score boosts (H14).
func BidShardPainBoost(score int, text string) int {
	if pain := classify.ClassifyBidShardPain(text); pain != nil {
		switch pain.PainBucket {
		case "infra_scale", "shave_discrepancy", "abuse_ddos":
			score += bidShardBucketInfraBoost
		case "cloak_stack_cost":
			score += bidShardBucketCloakBoost
		}
	}
	if validate.HasCommercialPainIntent(text) {
		score += bidShardCommercialIntentBoost
	}
	if HasVolumeSignal(text) {
		score += bidShardVolumeBoost
	}
	return score
}
