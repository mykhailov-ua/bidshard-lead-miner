package gemini

import (
	"context"
	"testing"
)

func TestSynthesizeEnrichment(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"company_type":"media_buyer","geo_confidence":"medium","summary":"Likely EU media buyer with tracker pain."}`)

	result, err := cl.SynthesizeEnrichment(context.Background(), EnrichSynthInput{
		Snippet:     "voluum alternative postback failing",
		Domain:      "buyer-team.com",
		RDAPCountry: "CY",
		Stack:       []string{"voluum"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CompanyType != "media_buyer" {
		t.Fatalf("company_type=%q", result.CompanyType)
	}
	if result.Summary == "" {
		t.Fatal("expected summary")
	}
}
