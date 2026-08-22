package entity

import "testing"

func TestFindLinkSuggestPairs(t *testing.T) {
	t.Parallel()

	docs := []EntityDoc{
		{
			EntityID:    "e1",
			AliasKeys:   []string{"domain:acme.com"},
			UnifiedPain: "Voluum postback pain",
		},
		{
			EntityID:    "e2",
			AliasKeys:   []string{"domain:acme.com"},
			UnifiedPain: "Recruiting publishers for CPA network",
		},
		{
			EntityID:    "e3",
			AliasKeys:   []string{"domain:other.com"},
			UnifiedPain: "Voluum postback pain",
		},
	}
	pairs := FindLinkSuggestPairs(docs)
	if len(pairs) != 1 {
		t.Fatalf("pairs=%d want 1", len(pairs))
	}
	if pairs[0].SharedDomain != "acme.com" {
		t.Fatalf("domain=%q", pairs[0].SharedDomain)
	}
}
