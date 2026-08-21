package extract

import "testing"

func TestWebDomains(t *testing.T) {
	t.Parallel()
	text := "Site https://www.topxpartners.com/contact and also spinbetter.io/about @noise"
	got := WebDomains(text)
	want := map[string]bool{"topxpartners.com": true, "spinbetter.io": true}
	if len(got) != len(want) {
		t.Fatalf("got %v want %d", got, len(want))
	}
	for _, d := range got {
		if !want[d] {
			t.Fatalf("unexpected %q in %v", d, got)
		}
	}
}

func TestWebDomainsSkipsSocial(t *testing.T) {
	t.Parallel()
	got := WebDomains("https://t.me/foo https://instagram.com/bar https://real-aff.net")
	if len(got) != 1 || got[0] != "real-aff.net" {
		t.Fatalf("got %v", got)
	}
}

func TestIsValidWebDomain(t *testing.T) {
	t.Parallel()
	if !IsValidWebDomain("bojoko.com") {
		t.Fatal("expected valid domain")
	}
	if IsValidWebDomain("13-02-2023-1.jpg") {
		t.Fatal("expected jpg host invalid")
	}
	if IsValidWebDomain("durov.gram") {
		t.Fatal("expected gram tld invalid")
	}
}
