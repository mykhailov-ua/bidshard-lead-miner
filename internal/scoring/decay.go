package scoring

import "time"

const (
	timeDecayFreshBoost   = 10
	timeDecayRecentBoost  = 5
	timeDecayAgedPenalty  = 5
	timeDecayStalePenalty = 10
)

type ScoreOpts struct {
	PostedAt        time.Time
	TimeDecay       bool
	StructuredStack bool
}

func ApplyTimeDecay(score int, postedAt time.Time, now time.Time) int {
	if postedAt.IsZero() {
		return score
	}
	age := now.Sub(postedAt)
	switch {
	case age <= 48*time.Hour:
		return score + timeDecayFreshBoost
	case age <= 7*24*time.Hour:
		return score + timeDecayRecentBoost
	case age <= 30*24*time.Hour:
		return score
	case age <= 180*24*time.Hour:
		return clampNonNegative(score - timeDecayAgedPenalty)
	default:
		return clampNonNegative(score - timeDecayStalePenalty)
	}
}

func clampNonNegative(score int) int {
	if score < 0 {
		return 0
	}
	return score
}
