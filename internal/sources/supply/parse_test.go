package supply

import (
	"testing"
)

func TestParseAdsTxtLine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		line     string
		wantOK   bool
		domain   string
		relation string
	}{
		{"voluum.com, 12345, DIRECT", true, "voluum.com", "DIRECT"},
		{"google.com, pub-1, RESELLER, f08c47fec0942fa0", true, "google.com", "RESELLER"},
		{"# comment only", false, "", ""},
		{"invalid", false, "", ""},
	}

	for _, tc := range cases {
		entry, ok := ParseAdsTxtLine(tc.line)
		if ok != tc.wantOK {
			t.Fatalf("line %q ok=%v want %v", tc.line, ok, tc.wantOK)
		}
		if !ok {
			continue
		}
		if entry.Domain != tc.domain || entry.Relation != tc.relation {
			t.Fatalf("line %q parsed %+v", tc.line, entry)
		}
	}
}

func TestParseSellersJSON(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"contact_email": "ops@igaming-team.com",
		"sellers": [
			{"name":"Acme","domain":"acme.com","contact_email":"ads@acme.com"}
		]
	}`)
	contacts := ParseSellersJSON(body)
	if len(contacts) != 2 {
		t.Fatalf("contacts=%d want 2", len(contacts))
	}
}

func TestLoadSeedDomains(t *testing.T) {
	t.Parallel()

	domains, err := LoadSeedDomains("../../../data/seeds/domains.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) < 20 {
		t.Fatalf("domains=%d want >=20", len(domains))
	}
}
