package filter

import "testing"

func TestHasPWAPainSignal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want bool
	}{
		{"postback from app not firing in keitaro", true},
		{"webview stuck on redirect delay", true},
		{"sub_id not in keitaro after PWA install", true},
		{"good morning team", false},
	}
	for _, tc := range cases {
		if got := HasPWAPainSignal(tc.text); got != tc.want {
			t.Fatalf("HasPWAPainSignal(%q)=%v want %v", tc.text, got, tc.want)
		}
	}
}

func TestIsPWAChannelHint(t *testing.T) {
	t.Parallel()
	if !IsPWAChannelHint("pwa_group_support", "PWA.Group client chat", "") {
		t.Fatal("expected PWA channel hint")
	}
	if IsPWAChannelHint("affhub", "Affiliate hub", "") {
		t.Fatal("expected false for generic affiliate")
	}
}
