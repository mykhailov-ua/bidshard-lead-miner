package pipeline

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

func TestProcessorRecordsEntityOnAccept(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	entityStore := entity.NewMemoryStore()
	store := sink.NewStubStore()
	proc := &Processor{
		Registry:        reg,
		Seen:            dedup.NewSeenCache(1000, 0),
		Store:           store,
		MX:              validate.StubMX{OK: true},
		EntityRecorder:  entityStore,
		EntitySightings: true,
	}

	item := model.RawItem{
		Source:  "stub:test",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@igaming-team.com",
	}
	out := proc.Process(context.Background(), Task{RoundID: "r1", Item: item})
	if !out.Accepted {
		t.Fatalf("expected accepted, outcome=%+v", out)
	}
	if out.Lead.EntityID == "" {
		t.Fatal("expected entity_id on accepted lead")
	}
	if out.Lead.EntitySightingCount != 1 {
		t.Fatalf("sighting_count=%d want 1", out.Lead.EntitySightingCount)
	}
}

func TestProcessorRecordsEntityOnDedup(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	contacts := []extract.Contact{{Type: "email", Value: "ops@igaming-team.com"}}
	entityStore := entity.NewMemoryStore()
	store := sink.NewStubStore()
	store.Seed(sink.LeadHashIDFromExtract(contacts))

	proc := &Processor{
		Registry:        reg,
		Seen:            dedup.NewSeenCache(1000, 0),
		Store:           store,
		MX:              validate.StubMX{OK: true},
		EntityRecorder:  entityStore,
		EntitySightings: true,
	}

	item := model.RawItem{
		Source:  "reddit:igaming",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@igaming-team.com",
	}
	out := proc.Process(context.Background(), Task{RoundID: "r1", Item: item})
	if !out.Dedup {
		t.Fatalf("expected dedup, outcome=%+v", out)
	}

	keys := entity.ResolveKeys(entity.ResolveInput{
		Source:   item.Source,
		Contacts: contacts,
	})
	entityID := entity.EntityID(keys)
	doc, ok := entityStore.Get(entityID)
	if !ok {
		t.Fatal("expected entity sighting on dedup")
	}
	if doc.SightingCount != 1 {
		t.Fatalf("sighting_count=%d want 1", doc.SightingCount)
	}
}

func TestProcessorAppliesCrossSourceHotBoost(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	entityStore := entity.NewMemoryStore()
	contacts := []extract.Contact{{Type: "email", Value: "ops@affnet.com"}}
	_, _ = entityStore.RecordSighting(context.Background(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			CompanyName: "AffNet Media",
			Source:      "telegram:@affnet",
			Contacts:    contacts,
		},
		HashID:  "hash-a",
		Matched: []string{"voluum"},
		Text:    "voluum alternative postback failing",
	})
	_, _ = entityStore.RecordSighting(context.Background(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			Source:   "reddit:igaming",
			Contacts: contacts,
		},
		HashID:  "hash-b",
		Matched: []string{"postback"},
		Text:    "postback failing on FTD",
	})

	store := sink.NewStubStore()
	proc := &Processor{
		Registry:         reg,
		Seen:             dedup.NewSeenCache(1000, 0),
		Store:            store,
		MX:               validate.StubMX{OK: true},
		EntityRecorder:   entityStore,
		EntitySightings:  true,
		CrossSourceHot:   true,
		CrossSourceBoost: 20,
	}

	item := model.RawItem{
		Source:  "tgweb:affnet.com",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@affnet.com",
	}
	out := proc.Process(context.Background(), Task{RoundID: "r1", Item: item})
	if !out.Accepted {
		t.Fatalf("expected accepted, outcome=%+v", out)
	}
	if !out.Lead.Hot {
		t.Fatal("expected hot lead")
	}
	found := false
	for _, tag := range out.Lead.Tags {
		if tag == entity.TagCrossSourceHot {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tags=%v", out.Lead.Tags)
	}
}

func TestProcessorTagsPublisherSurfaceFromAdsTxt(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	store := sink.NewStubStore()
	proc := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	item := model.RawItem{
		Source:  "ads_txt:example-pub.com",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@example-pub.com",
	}
	out := proc.Process(context.Background(), Task{RoundID: "r1", Item: item})
	if !out.Accepted {
		t.Fatalf("expected accepted, outcome=%+v", out)
	}
	found := false
	for _, tag := range out.Lead.Tags {
		if tag == scoring.TagPublisherSurface {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tags=%v", out.Lead.Tags)
	}
}

func TestProcessorCrossSourceHotSupplyAndTgweb(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	entityStore := entity.NewMemoryStore()
	contacts := []extract.Contact{{Type: "email", Value: "ops@affnet.com"}}
	_, _ = entityStore.RecordSighting(context.Background(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			Source:   "ads_txt:affnet.com",
			Contacts: contacts,
		},
		HashID:  "hash-supply",
		Matched: []string{"postback"},
		Text:    "postback failing on FTD",
	})

	store := sink.NewStubStore()
	proc := &Processor{
		Registry:         reg,
		Seen:             dedup.NewSeenCache(1000, 0),
		Store:            store,
		MX:               validate.StubMX{OK: true},
		EntityRecorder:   entityStore,
		EntitySightings:  true,
		CrossSourceHot:   true,
		CrossSourceBoost: 20,
	}

	item := model.RawItem{
		Source:  "tgweb:@aff_ops:affnet.com",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@affnet.com",
	}
	out := proc.Process(context.Background(), Task{RoundID: "r1", Item: item})
	if !out.Accepted {
		t.Fatalf("expected accepted, outcome=%+v", out)
	}
	if !out.Lead.Hot {
		t.Fatal("expected hot lead")
	}
	found := false
	for _, tag := range out.Lead.Tags {
		if tag == entity.TagCrossSourceHot {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tags=%v", out.Lead.Tags)
	}
}
