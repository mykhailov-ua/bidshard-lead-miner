package scoring

import "time"

const (
	timeDecayFreshBoost   = 10
	timeDecayStalePenalty = 10
)

type ScoreOpts struct {
	PostedAt  time.Time
	TimeDecay bool
}

func ApplyTimeDecay(score int, postedAt time.Time, now time.Time) int {
	if postedAt.IsZero() {
		return score
	}
	age := now.Sub(postedAt)
	switch {
	case age <= 24*time.Hour:
		return score + timeDecayFreshBoost
	case age > 7*24*time.Hour:
		if score > timeDecayStalePenalty {
			return score - timeDecayStalePenalty
		}
		return 0
	default:
		return score
	}
}
