package validate

import "testing"

func TestIsSEOMarketingCopy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title string
		text  string
		want  bool
	}{
		{"6 Best Keitaro Alternatives in 2026 For Media Buyers", "bot filtering real time reports", true},
		{"Best Affiliate Tracking Software in 2026", "Top picks for media buyers", true},
		{"Keitaro alternative", "voluum too expensive sign up for pricing plans", true},
		{"", "looking for voluum alternative because postback is failing", false},
		{"", "Is there a keitaro alternative that handles 50k clicks?", false},
		{"Keitaro alternative", "voluum too expensive sign up for pricing plans", true},
	}
	for _, tc := range cases {
		if got := IsSEOMarketingCopy(tc.text, tc.title); got != tc.want {
			t.Errorf("IsSEOMarketingCopy(%q,%q)=%v want %v", tc.text, tc.title, got, tc.want)
		}
	}
}
