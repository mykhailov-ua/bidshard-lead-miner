package config

import "testing"

func TestValidateProxyURLsAcceptsValid(t *testing.T) {
	if err := ValidateProxyURLs([]string{
		"http://user:pass@gw.dataimpulse.com:823",
		"http://127.0.0.1:3128",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateProxyURLsRejectsDialZero(t *testing.T) {
	cases := []string{
		"http://:0",
		"http://user:pass@:823",
		"http://user:pass@",
		"http://:@",
		"socks5://host:1080",
		"",
	}
	for _, raw := range cases {
		if err := ValidateProxyURLs([]string{raw}); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}
