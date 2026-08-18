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
		{"week old", now.Add(-8 * 24 * time.Hour), 40},
		{"mid age", now.Add(-3 * 24 * time.Hour), 50},
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
