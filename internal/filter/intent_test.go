package filter

import "testing"

func TestHasCommercialPainIntent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want bool
	}{
		{"voluum media buyer igaming", false},
		{"postback failing on voluum migration", true},
		{"looking for a tracker alternative to keitaro", true},
	}
	for _, tc := range cases {
		if got := HasCommercialPainIntent(tc.text); got != tc.want {
			t.Fatalf("text=%q got=%v want=%v", tc.text, got, tc.want)
		}
	}
}
