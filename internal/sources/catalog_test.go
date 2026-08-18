package sources

import "testing"

func TestCatalog(t *testing.T) {
	t.Parallel()

	catalog := Catalog()
	if len(catalog) < 5 {
		t.Fatalf("expected several sources, got %d", len(catalog))
	}
	names := make(map[string]bool, len(catalog))
	for _, info := range catalog {
		if info.Name == "" {
			t.Fatal("empty source name")
		}
		names[info.Name] = true
	}
	for _, required := range []string{"stub", "forum", "discord"} {
		if !names[required] {
			t.Fatalf("catalog missing %q", required)
		}
	}
}
