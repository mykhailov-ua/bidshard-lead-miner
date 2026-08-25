package seedfeedback

import (
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sourceregistry"
)

func TestAcceptedHotLeadEnqueuesDomain(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "source_registry.json")
	rec := NewRecorder(Config{
		Enabled:      true,
		RegistryPath: regPath,
		MinHeatTier:  entity.HeatTierHot,
	})
	added := rec.RecordAccepted(AcceptInput{
		Lead: model.Lead{
			Source:   "reddit:igaming",
			HeatTier: entity.HeatTierHot,
			Priority: "High",
		},
		Contacts: []extract.Contact{{Type: "email", Value: "ops@affnet.com"}},
	})
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
	f, err := sourceregistry.Load(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Sources) != 1 || f.Sources[0].Domain != "affnet.com" {
		t.Fatalf("sources=%+v", f.Sources)
	}
	if f.Sources[0].DiscoveredBy != "accepted_lead" {
		t.Fatalf("discovered_by=%q", f.Sources[0].DiscoveredBy)
	}
}

func TestForumCrossMentionEnqueuesSnippetDomain(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "source_registry.json")
	rec := NewRecorder(Config{
		Enabled:      true,
		RegistryPath: regPath,
		MinHeatTier:  entity.HeatTierHot,
	})
	added := rec.RecordAccepted(AcceptInput{
		Lead: model.Lead{
			Source:   "forum:affiliatefix.com/voluum-thread",
			HeatTier: entity.HeatTierCold,
			Priority: "Low",
		},
		Snippet: "considering migration to https://partner-aff.io tracker stack",
	})
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
	f, err := sourceregistry.Load(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Sources) != 1 || f.Sources[0].Domain != "partner-aff.io" {
		t.Fatalf("sources=%+v", f.Sources)
	}
	if f.Sources[0].DiscoveredBy != "forum_crossmention" {
		t.Fatalf("discovered_by=%q", f.Sources[0].DiscoveredBy)
	}
}

func TestColdLowLeadSkippedWithoutForumMention(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "source_registry.json")
	rec := NewRecorder(Config{
		Enabled:      true,
		RegistryPath: regPath,
		MinHeatTier:  entity.HeatTierHot,
	})
	added := rec.RecordAccepted(AcceptInput{
		Lead: model.Lead{
			Source:   "reddit:igaming",
			HeatTier: entity.HeatTierCold,
			Priority: "Low",
		},
		Contacts: []extract.Contact{{Type: "email", Value: "ops@affnet.com"}},
	})
	if added != 0 {
		t.Fatalf("added=%d want 0", added)
	}
}

func TestNewRecorderDisabledReturnsNil(t *testing.T) {
	if rec := NewRecorder(Config{Enabled: false, RegistryPath: "x"}); rec != nil {
		t.Fatal("expected nil recorder when disabled")
	}
}
