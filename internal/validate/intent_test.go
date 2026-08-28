package validate

import "testing"

func TestHasCommercialPainIntent(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"посоветуйте трекер для FB", true},
		{"alternative to voluum for igaming team", true},
		{"postback not working after migration from keitaro", true},
		{"как настроить клоаку в binom?", true},
		{"join our channel for vip signals", false},
		{"we are hiring a media buyer", false},
	}
	for _, tc := range cases {
		if got := HasCommercialPainIntent(tc.text); got != tc.want {
			t.Fatalf("text=%q got=%v want=%v", tc.text, got, tc.want)
		}
	}
}
