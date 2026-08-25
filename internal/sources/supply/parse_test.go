package supply

import (
	"os"
	"testing"

	"github.com/bidshard/parser/internal/sourceregistry"
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

func TestLoadSeedDomainsCombinedUsesRegistry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	regPath := dir + "/source_registry.json"
	if _, err := sourceregistry.Upsert(regPath, sourceregistry.Entry{
		Domain:       "registry-only.example",
		Types:        []string{sourceregistry.TypeSupply},
		DiscoveredBy: "telegram",
	}); err != nil {
		t.Fatal(err)
	}

	domains, err := LoadSeedDomainsCombined("", regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "registry-only.example" {
		t.Fatalf("domains=%v", domains)
	}
}

func TestLoadSeedDomainsSkipsRUGeoTag(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/seed.csv"
	content := "domain,geo,notes\nallowed.example,global,ok\nblocked.example,ru,skip\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	domains, err := LoadSeedDomains(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "allowed.example" {
		t.Fatalf("domains=%v want [allowed.example]", domains)
	}
}
