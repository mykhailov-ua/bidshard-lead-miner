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

func TestEntityClassifyServiceDebounce(t *testing.T) {
	store := entity.NewMemoryStore()
	now := time.Now().UTC()
	_, err := store.RecordSighting(t.Context(), entity.SightingInput{
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
	}, capturer, store, store, classifier)

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
	}, capturer, store, store, classifier)

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

func TestEntityClassifyServiceNilSafe(t *testing.T) {
	var svc *EntityClassifyService
	var wg sync.WaitGroup
	svc.Run(context.Background(), &wg)
}
