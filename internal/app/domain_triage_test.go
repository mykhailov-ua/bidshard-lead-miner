package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sourceregistry"
)

type stubDomainTriageClient struct{}

func (s *stubDomainTriageClient) TriageDomains(_ context.Context, items []gemini.DomainTriageInput) ([]gemini.DomainTriageResult, error) {
	out := make([]gemini.DomainTriageResult, 0, len(items))
	for _, item := range items {
		out = append(out, gemini.DomainTriageResult{
			ID:     item.ID,
			Action: "drop",
			Why:    "stub noise",
		})
	}
	return out, nil
}

func TestRunDomainTriageStubGeminiDrops(t *testing.T) {
	dir := t.TempDir()
	regPath := filepath.Join(dir, "source_registry.json")
	cachePath := filepath.Join(dir, "domain_triage_cache.json")

	if _, err := sourceregistry.Upsert(regPath, sourceregistry.Entry{
		Domain:       "noise-blog.example",
		Types:        []string{sourceregistry.TypeSupply},
		DiscoveredBy: "telegram",
		Channel:      "aff_net",
	}); err != nil {
		t.Fatal(err)
	}

	if err := RunDomainTriage(context.Background(), DomainTriageConfig{
		RegistryPath: regPath,
		CachePath:    cachePath,
		BatchSize:    5,
	}, &stubDomainTriageClient{}); err != nil {
		t.Fatal(err)
	}

	entry, ok := sourceregistry.LookupEntry(regPath, "noise-blog.example")
	if !ok {
		t.Fatal("expected registry entry")
	}
	if entry.TriageStatus != "drop" {
		t.Fatalf("triage_status=%q want drop", entry.TriageStatus)
	}
}
