package enrich

import (
	"testing"
)

func TestParseRDAPCountry(t *testing.T) {
	body := []byte(`{"country":"BY","events":[{"eventAction":"registration","eventDate":"2020-01-01T00:00:00Z"}]}`)
	info := parseRDAP(body)
	if info.Country != "BY" {
		t.Fatalf("country=%q", info.Country)
	}
	if info.CreatedAt.IsZero() {
		t.Fatal("expected created_at")
	}
}

func TestTargetDomain(t *testing.T) {
	d := TargetDomain("ads_txt:example.com", nil)
	if d != "example.com" {
		t.Fatalf("got %q", d)
	}
	d = TargetDomain("tgweb:@affnet:topxpartners.com", nil)
	if d != "topxpartners.com" {
		t.Fatalf("tgweb domain=%q", d)
	}
}
