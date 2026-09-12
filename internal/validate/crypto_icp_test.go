package validate

import "testing"

func TestHasCryptoGrayBuyerSignal(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"keitaro shaving on USDT payouts weekly", true},
		{"just usdt pump signal join vip", false},
		{"binom bot click fraud on our ftd", true},
		{"good morning team", false},
	}
	for _, tc := range cases {
		if got := HasCryptoGrayBuyerSignal(tc.text); got != tc.want {
			t.Fatalf("text=%q got=%v want=%v", tc.text, got, tc.want)
		}
	}
}

func TestHasTrackerPainMessage(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"Weekly digest: voluum keitaro binom market news", false},
		{"keitaro postback failing again", true},
		{"looking for voluum alternative", true},
	}
	for _, tc := range cases {
		if got := HasTrackerPainMessage(tc.text); got != tc.want {
			t.Fatalf("text=%q got=%v want=%v", tc.text, got, tc.want)
		}
	}
}
