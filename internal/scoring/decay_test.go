package scoring

import (
	"testing"
	"time"
)

func TestApplyTimeDecay(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		at   time.Time
		want int
	}{
		{"fresh", now.Add(-2 * time.Hour), 60},
		{"recent week", now.Add(-3 * 24 * time.Hour), 55},
		{"mid age", now.Add(-20 * 24 * time.Hour), 50},
		{"aged", now.Add(-60 * 24 * time.Hour), 45},
		{"stale", now.Add(-200 * 24 * time.Hour), 40},
		{"zero time", time.Time{}, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyTimeDecay(50, tc.at, now)
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}
