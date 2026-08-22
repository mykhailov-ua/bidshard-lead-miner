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

type stubAnalyzer struct {
	calls int
	failN int
}

func (a *stubAnalyzer) AnalyzeLeadBatch(_ context.Context, items []gemini.LeadBatchInput, _ bool) ([]gemini.LeadBatchResult, error) {
	a.calls++
	if a.calls <= a.failN {
		return nil, ErrWarmAnalysisExhausted
	}
	out := make([]gemini.LeadBatchResult, 0, len(items))
	for _, item := range items {
		out = append(out, gemini.LeadBatchResult{
			HashID: item.ID,
			ICP:    gemini.ICPResult{ICP: "starter"},
			Geo:    gemini.GeoResult{PersonCountry: "US", Confidence: "medium"},
		})
	}
	return out, nil
}

type recordingDLQ struct {
	batches  [][]Event
	attempts int
	err      error
}

func (d *recordingDLQ) InsertWarmAnalysisFailures(_ context.Context, batch []Event, attempts int, err error) error {
	d.batches = append(d.batches, append([]Event(nil), batch...))
	d.attempts = attempts
	d.err = err
	return nil
}

func TestProcessBatchRetriesBeforeSuccess(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	analyzer := &stubAnalyzer{failN: 2}
	s := &Service{
		cfg: Config{
			RetryMaxAttempts:   3,
			RetryBaseDelay:     time.Millisecond,
			GeoClassifyEnabled: false,
		},
		patcher:  patcher,
		analyzer: analyzer,
	}

	s.processBatch(context.Background(), []Event{{HashID: "h-retry", Source: "forum", Score: 40, Priority: "Medium"}})

	if analyzer.calls != 3 {
		t.Fatalf("analyzer calls=%d want 3", analyzer.calls)
	}
	patch := patcher.last()
	if patch.AnalysisStatus != "done" {
		t.Fatalf("analysis_status=%q want done", patch.AnalysisStatus)
	}
}

func TestProcessBatchDLQAfterRetries(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	analyzer := &stubAnalyzer{failN: 5}
	dlq := &recordingDLQ{}
	s := &Service{
		cfg: Config{
			RetryMaxAttempts: 3,
			RetryBaseDelay:   time.Millisecond,
		},
		patcher:  patcher,
		analyzer: analyzer,
		dlq:      dlq,
	}
	batch := []Event{{HashID: "h-fail", Source: "forum"}}

	s.processBatch(context.Background(), batch)

	if len(dlq.batches) != 1 || len(dlq.batches[0]) != 1 {
		t.Fatalf("dlq batches=%v", dlq.batches)
	}
	if dlq.attempts != 3 {
		t.Fatalf("dlq attempts=%d want 3", dlq.attempts)
	}
	if len(patcher.patches) != 0 {
		t.Fatalf("expected no patches, got %d", len(patcher.patches))
	}
}

func TestProcessBatchRequeuesOnCancel(t *testing.T) {
	t.Parallel()

	analyzer := &stubAnalyzer{failN: 5}
	s := &Service{
		cfg: Config{
			RetryMaxAttempts: 3,
			RetryBaseDelay:   time.Second,
		},
		analyzer: analyzer,
		buffer:   make([]Event, 0, 4),
	}
	batch := []Event{{HashID: "h-cancel", Source: "forum"}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.processBatch(ctx, batch)

	if len(s.buffer) != 1 || s.buffer[0].HashID != "h-cancel" {
		t.Fatalf("buffer=%+v want re-queued event", s.buffer)
	}
}

func TestEnqueueDedupe(t *testing.T) {
	t.Parallel()

	s := &Service{buffer: []Event{{HashID: "a"}}}
	added := s.enqueueDedupe([]Event{
		{HashID: "a"},
		{HashID: "b"},
		{HashID: "b"},
	})
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
	if len(s.buffer) != 2 {
		t.Fatalf("buffer len=%d want 2", len(s.buffer))
	}
}

type stubWarmCluster struct {
	dup    bool
	dupOf  string
	record int
}

func (s *stubWarmCluster) CheckDuplicate(_ context.Context, _, _ string) (bool, string, error) {
	return s.dup, s.dupOf, nil
}

func (s *stubWarmCluster) Record(_ context.Context, _, _ string) error {
	s.record++
	return nil
}

type stubMediumEngager struct {
	angle string
}

func (s stubMediumEngager) ClassifyEngagement(context.Context, gemini.EngagementInput) (gemini.EngagementResult, error) {
	return gemini.EngagementResult{OutreachAngle: s.angle}, nil
}

func TestApplyResultMarksSemanticDuplicate(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	cluster := &stubWarmCluster{dup: true, dupOf: "canonical-hash"}
	s := &Service{
		cfg:     Config{GeoClassifyEnabled: false},
		patcher: patcher,
		cluster: cluster,
	}
	ev := Event{HashID: "dup-hash", Source: "forum", Score: 80, Priority: "High", Snippet: "voluum pain"}
	res := gemini.LeadBatchResult{
		ICP: gemini.ICPResult{ICP: "pro", Hot: true},
	}
	s.applyResult(context.Background(), ev, res, 70)

	patch := patcher.last()
	if patch.DuplicateOf != "canonical-hash" {
		t.Fatalf("duplicate_of=%q", patch.DuplicateOf)
	}
	if patch.AnalysisStatus != "duplicate" {
		t.Fatalf("status=%q", patch.AnalysisStatus)
	}
}

func TestApplyResultMediumEngageAngle(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	s := &Service{
		cfg: Config{
			GeoClassifyEnabled:  false,
			EngageMediumEnabled: true,
		},
		patcher:      patcher,
		engageMedium: stubMediumEngager{angle: "Tracker migration hook"},
	}
	ev := Event{
		HashID:   "med-1",
		Source:   "forum",
		Score:    55,
		Priority: "Medium",
		HeatTier: "warm",
		Snippet:  "voluum alternative",
	}
	res := gemini.LeadBatchResult{ICP: gemini.ICPResult{ICP: "starter"}}
	s.applyResult(context.Background(), ev, res, 70)

	patch := patcher.last()
	if patch.OutreachAngle != "Tracker migration hook" {
		t.Fatalf("outreach_angle=%q", patch.OutreachAngle)
	}
	if patch.PilotQualified {
		t.Fatal("medium engage must not set pilot_qualified")
	}
}
