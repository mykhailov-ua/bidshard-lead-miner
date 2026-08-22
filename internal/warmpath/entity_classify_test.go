package warmpath

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/gemini"
)

type stubEntityClassifier struct {
	result gemini.EntityClassifyResult
	err    error
	calls  int
}

func (s *stubEntityClassifier) ClassifyEntity(context.Context, gemini.EntityClassifyInput) (gemini.EntityClassifyResult, error) {
	s.calls++
	return s.result, s.err
}

type stubLeadHeatSync struct {
	calls int
	last  entity.LinkedLeadHeatSync
}

func (s *stubLeadHeatSync) PatchLinkedLeadsHeat(_ context.Context, entityID string, hashIDs []string, sync entity.LinkedLeadHeatSync) error {
	s.calls++
	s.last = sync
	return nil
}

func TestEntityClassifyServiceDebounce(t *testing.T) {
	store := entity.NewMemoryStore()
	now := time.Now().UTC()
	_, _ = store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			CompanyName: "AffNet",
			Source:      "telegram:@affnet",
			Contacts:    []extract.Contact{{Type: "email", Value: "ops@affnet.com"}},
		},
		HashID:   "hash-a",
		Matched:  []string{"voluum"},
		Text:     "voluum alternative",
		PostedAt: now,
		Score:    80,
	})
	result, err := store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			Source:   "reddit:igaming",
			Contacts: []extract.Contact{{Type: "email", Value: "ops@affnet.com"}},
		},
		HashID:   "hash-b",
		Matched:  []string{"postback"},
		Text:     "postback failing",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}

	classifier := &stubEntityClassifier{
		result: gemini.EntityClassifyResult{
			SameActor:       true,
			ActorConfidence: 0.9,
			UnifiedPain:     "tracker pain",
			BuyerIntent:     "hot",
		},
	}
	capturer := NewEntityClassifyCapturer(8)
	svc := NewEntityClassifyService(EntityClassifyConfig{
		AnalyzeInterval: 15 * time.Millisecond,
		Debounce:        time.Hour,
	}, capturer, store, store, classifier, EntityClassifyExtras{})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	svc.Run(ctx, &wg)

	capturer.TryCapture(EntityClassifyEvent{EntityID: result.EntityID})
	capturer.TryCapture(EntityClassifyEvent{EntityID: result.EntityID})

	time.Sleep(60 * time.Millisecond)
	cancel()
	wg.Wait()

	if classifier.calls != 1 {
		t.Fatalf("classify calls=%d want 1", classifier.calls)
	}
	doc, ok := store.Get(result.EntityID)
	if !ok {
		t.Fatal("entity missing")
	}
	if doc.UnifiedPain != "tracker pain" {
		t.Fatalf("unified_pain=%q", doc.UnifiedPain)
	}
}

func TestEntityClassifyServiceSyncsLeadHeatOnDowngrade(t *testing.T) {
	store := entity.NewMemoryStore()
	now := time.Now().UTC()
	first, err := store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			CompanyName: "Acme",
			Source:      "forum:affiliatefix.com/a",
			Contacts:    []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "h1",
		Matched:  []string{"voluum"},
		Text:     "voluum pain",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			Source:   "reddit:igaming",
			Contacts: []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "h2",
		Matched:  []string{"postback"},
		Text:     "postback issue",
		PostedAt: now,
		Score:    80,
	})
	if err != nil {
		t.Fatal(err)
	}

	leadHeat := &stubLeadHeatSync{}
	classifier := &stubEntityClassifier{
		result: gemini.EntityClassifyResult{
			SameActor:        false,
			ActorConfidence:  0.9,
			BuyerIntent:      "cold",
			SplitRecommended: true,
		},
	}
	capturer := NewEntityClassifyCapturer(4)
	svc := NewEntityClassifyService(EntityClassifyConfig{
		AnalyzeInterval: 10 * time.Millisecond,
		Debounce:        time.Hour,
	}, capturer, store, store, classifier, EntityClassifyExtras{LeadHeat: leadHeat})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	svc.Run(ctx, &wg)
	capturer.TryCapture(EntityClassifyEvent{EntityID: first.EntityID, ForceFresh: true})
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()

	if leadHeat.calls != 1 {
		t.Fatalf("lead heat sync calls=%d want 1", leadHeat.calls)
	}
	if leadHeat.last.HeatTier != entity.HeatTierWarm {
		t.Fatalf("heat_tier=%q want warm", leadHeat.last.HeatTier)
	}
}

func TestEntityClassifyServiceForceFreshBypassesDebounce(t *testing.T) {
	store := entity.NewMemoryStore()
	now := time.Now().UTC()
	first, err := store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			CompanyName: "Acme",
			Source:      "forum:affiliatefix.com/a",
			Contacts:    []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "h1",
		Matched:  []string{"voluum"},
		Text:     "voluum pain",
		PostedAt: now,
		Score:    70,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			Source:   "reddit:igaming",
			Contacts: []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "h2",
		Matched:  []string{"postback"},
		Text:     "postback issue",
		PostedAt: now,
		Score:    70,
	})

	classifier := &stubEntityClassifier{
		result: gemini.EntityClassifyResult{SameActor: true, ActorConfidence: 0.8, BuyerIntent: "warm"},
	}
	capturer := NewEntityClassifyCapturer(8)
	svc := NewEntityClassifyService(EntityClassifyConfig{
		AnalyzeInterval: 15 * time.Millisecond,
		Debounce:        time.Hour,
	}, capturer, store, store, classifier, EntityClassifyExtras{})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	svc.Run(ctx, &wg)

	capturer.TryCapture(EntityClassifyEvent{EntityID: first.EntityID, ForceFresh: true})
	time.Sleep(40 * time.Millisecond)
	capturer.TryCapture(EntityClassifyEvent{EntityID: first.EntityID, ForceFresh: true})
	time.Sleep(40 * time.Millisecond)
	cancel()
	wg.Wait()

	if classifier.calls != 2 {
		t.Fatalf("classify calls=%d want 2", classifier.calls)
	}
}

func TestMergeEntityClassifyEventPreservesForceFresh(t *testing.T) {
	prev := EntityClassifyEvent{EntityID: "e1", ForceFresh: true}
	ev := EntityClassifyEvent{EntityID: "e1", ForceFresh: false}
	got := mergeEntityClassifyEvent(prev, ev)
	if !got.ForceFresh {
		t.Fatal("expected ForceFresh preserved when coalescing")
	}
}

func TestEntityClassifyServiceNilSafe(t *testing.T) {
	var svc *EntityClassifyService
	var wg sync.WaitGroup
	svc.Run(context.Background(), &wg)
}

type stubLeadOutreachSync struct {
	calls int
	last  entity.LinkedLeadOutreachSync
}

func (s *stubLeadOutreachSync) PatchLinkedLeadsOutreach(_ context.Context, _ string, _ []string, sync entity.LinkedLeadOutreachSync) error {
	s.calls++
	s.last = sync
	return nil
}

type stubOutreachNarrator struct {
	result gemini.EntityOutreachResult
}

func (s *stubOutreachNarrator) BuildEntityOutreach(context.Context, gemini.EntityOutreachInput) (gemini.EntityOutreachResult, error) {
	return s.result, nil
}

func TestEntityClassifyServiceSyncsOutreachNarrative(t *testing.T) {
	store := entity.NewMemoryStore()
	now := time.Now().UTC()
	first, err := store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			CompanyName: "Acme",
			Source:      "forum:affiliatefix.com/a",
			Contacts:    []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "h1",
		Matched:  []string{"voluum"},
		Text:     "voluum pain",
		PostedAt: now,
		Score:    70,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.RecordSighting(t.Context(), entity.SightingInput{
		ResolveInput: entity.ResolveInput{
			Source:   "reddit:igaming",
			Contacts: []extract.Contact{{Type: "email", Value: "ops@acme.com"}},
		},
		HashID:   "h2",
		Matched:  []string{"postback"},
		Text:     "postback issue",
		PostedAt: now,
		Score:    70,
	})

	classifier := &stubEntityClassifier{
		result: gemini.EntityClassifyResult{
			SameActor:       true,
			ActorConfidence: 0.9,
			UnifiedPain:     "Voluum migration pain",
			BuyerIntent:     "hot",
		},
	}
	outreach := &stubLeadOutreachSync{}
	narrator := &stubOutreachNarrator{
		result: gemini.EntityOutreachResult{
			OutreachAngle: "Tracker migration across forum and reddit",
			EntityProof:   "Three threads with consistent voluum pain",
		},
	}
	capturer := NewEntityClassifyCapturer(8)
	svc := NewEntityClassifyService(EntityClassifyConfig{
		AnalyzeInterval:          15 * time.Millisecond,
		Debounce:               time.Millisecond,
		OutreachNarrativeEnabled: true,
	}, capturer, store, store, classifier, EntityClassifyExtras{
		Outreach: outreach,
		Narrator: narrator,
	})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	svc.Run(ctx, &wg)
	capturer.TryCapture(EntityClassifyEvent{EntityID: first.EntityID, ForceFresh: true})
	time.Sleep(80 * time.Millisecond)
	cancel()
	wg.Wait()

	if outreach.calls != 1 {
		t.Fatalf("outreach calls=%d want 1", outreach.calls)
	}
	if outreach.last.OutreachAngle == "" {
		t.Fatal("expected outreach_angle")
	}
	if outreach.last.EntityProof == "" {
		t.Fatal("expected entity_proof")
	}
}
