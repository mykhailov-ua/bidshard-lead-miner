//go:build integration

package gemini

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/discover"
)

func TestLiveGeminiSmoke(t *testing.T) {
	config.LoadDotEnv()

	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = "gemini-2.5-flash"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cl, err := NewClient(apiKey, model)
	if err != nil {
		t.Fatal(err)
	}

	icp, err := cl.ClassifyICP(ctx, "Switching from Voluum, postback failing on high volume igaming traffic.")
	if quotaSkip(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("ClassifyICP: %v", err)
	}
	t.Logf("ICP=%s hot=%v spend=%s", icp.ICP, icp.Hot, icp.SpendTier)

	vec, err := cl.EmbedText(ctx, "voluum alternative tracker migration")
	if quotaSkip(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}
	if len(vec) != 768 {
		t.Fatalf("embed dims=%d want 768", len(vec))
	}
	t.Logf("embed dims=%d", len(vec))

	engage, err := cl.ClassifyEngagement(ctx, EngagementInput{
		Text:         "Migrating from Keitaro, $8k/mo spend, postback issues",
		Stack:        []string{"keitaro"},
		ICP:          icp.ICP,
		ContactTypes: []string{"telegram"},
		Source:       "telegram:test",
	})
	if quotaSkip(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("ClassifyEngagement: %v", err)
	}
	t.Logf("engage channel=%s pilot=%v", engage.OutreachChannel, engage.PilotQualified)

	synth, err := cl.SynthesizeEnrichment(ctx, EnrichSynthInput{
		Snippet:       "voluum alternative postback failing",
		Domain:        "buyer-team.com",
		RDAPCountry:   "CY",
		DomainAgeDays: 1200,
		Stack:         []string{"voluum"},
		ICP:           icp.ICP,
	})
	if quotaSkip(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("SynthesizeEnrichment: %v", err)
	}
	t.Logf("enrich company_type=%s", synth.CompanyType)

	diff, err := cl.BuildDiscoverICPDiff(ctx, discover.ICPConfig{
		TelegramSearch: []string{"voluum alternative"},
		SerpDorks:      []string{"site:t.me voluum"},
	}, []string{"binom migration"}, nil)
	if quotaSkip(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("BuildDiscoverICPDiff: %v", err)
	}
	t.Logf("discover diff telegram=%d serp=%d", len(diff.AddTelegramSearch), len(diff.AddSerpDorks))
}

func quotaSkip(t *testing.T, err error) bool {
	t.Helper()
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "resource_exhausted") || strings.Contains(msg, "quota") || strings.Contains(msg, "rate limit") {
		t.Skipf("gemini quota/rate limit: %v", err)
		return true
	}
	return false
}
