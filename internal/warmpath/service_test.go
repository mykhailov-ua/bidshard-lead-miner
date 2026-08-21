package warmpath

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

type recordingPatcher struct {
	mu      sync.Mutex
	patches []sink.LeadAnalysisPatch
}

func (p *recordingPatcher) PatchLeadAnalysis(_ context.Context, patch sink.LeadAnalysisPatch) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.patches = append(p.patches, patch)
	return nil
}

func (p *recordingPatcher) last() sink.LeadAnalysisPatch {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.patches) == 0 {
		return sink.LeadAnalysisPatch{}
	}
	return p.patches[len(p.patches)-1]
}

func TestApplyResultSkipsGeoRejectWhenGeoDisabled(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	s := &Service{
		cfg: Config{
			GeoClassifyEnabled: false,
			GeoBlockCountries:  []string{"RU", "BY"},
		},
		patcher: patcher,
	}

	ev := Event{HashID: "h1", Source: "forum", Score: 50, Priority: "Medium"}
	res := gemini.LeadBatchResult{
		Geo: gemini.GeoResult{
			Blocked:       true,
			Confidence:    "high",
			PersonCountry: "RU",
		},
		ICP: gemini.ICPResult{ICP: "starter"},
	}

	s.applyResult(context.Background(), ev, res, 70)

	patch := patcher.last()
	if patch.AnalysisStatus != "done" {
		t.Fatalf("analysis_status=%q want done", patch.AnalysisStatus)
	}
}

func TestApplyResultGeoRejectWhenGeoEnabled(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	s := &Service{
		cfg: Config{
			GeoClassifyEnabled: true,
			GeoBlockCountries:  []string{"RU", "BY"},
		},
		patcher: patcher,
	}

	ev := Event{HashID: "h2", Source: "forum"}
	res := gemini.LeadBatchResult{
		Geo: gemini.GeoResult{
			Blocked:       true,
			Confidence:    "high",
			PersonCountry: "RU",
		},
	}

	s.applyResult(context.Background(), ev, res, 70)

	patch := patcher.last()
	if patch.AnalysisStatus != "geo_rejected" {
		t.Fatalf("analysis_status=%q want geo_rejected", patch.AnalysisStatus)
	}
}

func TestApplyResultDeferredWebhookOnDone(t *testing.T) {
	done := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	patcher := &recordingPatcher{}
	client := sink.NewWebhookClient(srv.URL, "", time.Second)
	s := &Service{
		cfg: Config{
			GeoClassifyEnabled:      true,
			GeoBlockCountries:       []string{"RU", "BY"},
			CRMWebhook:              client,
			CRMWebhookAfterAnalysis: true,
		},
		patcher: patcher,
	}

	ev := Event{HashID: "h3", Source: "forum", Score: 80, Priority: "High"}
	res := gemini.LeadBatchResult{
		Geo: gemini.GeoResult{PersonCountry: "US", Confidence: "medium"},
		ICP: gemini.ICPResult{ICP: "pro", Hot: true},
	}

	s.applyResult(context.Background(), ev, res, 70)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deferred crm webhook not called")
	}
}

func TestLeadForCRMUsesDoneAnalysis(t *testing.T) {
	t.Parallel()

	lead := leadForCRM(Event{
		HashID: "abc",
		Source: "tgweb",
		Title:  "Voluum alt",
	}, sink.LeadAnalysisPatch{
		AnalysisStatus: "done",
		Priority:       "High",
		Score:          85,
		ICP:            "pro",
	})

	if lead.AnalysisStatus != "done" {
		t.Fatalf("analysis_status=%q", lead.AnalysisStatus)
	}
	if lead.HashID != "abc" || lead.Source != "tgweb" {
		t.Fatalf("lead=%+v", lead)
	}
}
