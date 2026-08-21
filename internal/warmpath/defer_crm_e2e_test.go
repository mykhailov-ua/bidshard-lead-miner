package warmpath

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

type stubLeadBatchAnalyzer struct {
	respond func(items []gemini.LeadBatchInput) ([]gemini.LeadBatchResult, error)
}

func (s *stubLeadBatchAnalyzer) AnalyzeLeadBatch(_ context.Context, items []gemini.LeadBatchInput, _ bool) ([]gemini.LeadBatchResult, error) {
	if s.respond == nil {
		return nil, nil
	}
	return s.respond(items)
}

func (p *recordingPatcher) patchFor(hashID string) (sink.LeadAnalysisPatch, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, patch := range p.patches {
		if patch.HashID == hashID {
			return patch, true
		}
	}
	return sink.LeadAnalysisPatch{}, false
}

func TestDeferredCRMWebhookE2E_MockGeminiBatch(t *testing.T) {
	var webhookMu sync.Mutex
	webhookHashes := make([]string, 0, 2)
	webhookDone := make(chan struct{}, 4)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var doc sink.LeadDoc
		_ = json.Unmarshal(body, &doc)
		webhookMu.Lock()
		webhookHashes = append(webhookHashes, doc.HashID)
		webhookMu.Unlock()
		webhookDone <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	patcher := &recordingPatcher{}
	analyzer := &stubLeadBatchAnalyzer{
		respond: func(items []gemini.LeadBatchInput) ([]gemini.LeadBatchResult, error) {
			out := make([]gemini.LeadBatchResult, 0, len(items))
			for _, in := range items {
				switch in.ID {
				case "ru-lead":
					out = append(out, gemini.LeadBatchResult{
						HashID: in.ID,
						Geo: gemini.GeoResult{
							Blocked:       true,
							Confidence:    "high",
							PersonCountry: "RU",
							Why:           "mock RU operator",
						},
					})
				case "en-lead":
					out = append(out, gemini.LeadBatchResult{
						HashID: in.ID,
						Geo: gemini.GeoResult{
							PersonCountry: "US",
							Confidence:    "high",
						},
						ICP: gemini.ICPResult{ICP: "pro", Hot: true},
					})
				default:
					return nil, fmt.Errorf("unexpected batch id %q", in.ID)
				}
			}
			return out, nil
		},
	}

	svc := &Service{
		cfg: Config{
			BatchSize:               10,
			GeoClassifyEnabled:      true,
			GeoBlockCountries:       []string{"RU", "BY"},
			CRMWebhook:              sink.NewWebhookClient(srv.URL, "", 2*time.Second),
			CRMWebhookAfterAnalysis: true,
		},
		patcher:  patcher,
		analyzer: analyzer,
		buffer: []Event{
			{HashID: "ru-lead", Source: "forum", Score: 60, Priority: "Medium", Snippet: "Moscow team"},
			{HashID: "en-lead", Source: "forum", Score: 85, Priority: "High", Snippet: "voluum alternative LATAM"},
		},
	}

	svc.flush(context.Background())

	ruPatch, ok := patcher.patchFor("ru-lead")
	if !ok {
		t.Fatal("missing patch for ru-lead")
	}
	if ruPatch.AnalysisStatus != "geo_rejected" {
		t.Fatalf("ru-lead analysis_status=%q want geo_rejected", ruPatch.AnalysisStatus)
	}

	enPatch, ok := patcher.patchFor("en-lead")
	if !ok {
		t.Fatal("missing patch for en-lead")
	}
	if enPatch.AnalysisStatus != "done" {
		t.Fatalf("en-lead analysis_status=%q want done", enPatch.AnalysisStatus)
	}

	select {
	case <-webhookDone:
	case <-time.After(2 * time.Second):
		t.Fatal("expected deferred CRM webhook")
	}

	webhookMu.Lock()
	defer webhookMu.Unlock()
	if len(webhookHashes) != 1 {
		t.Fatalf("webhook count=%d want 1 hashes=%v", len(webhookHashes), webhookHashes)
	}
	if webhookHashes[0] != "en-lead" {
		t.Fatalf("webhook hash_id=%q want en-lead", webhookHashes[0])
	}

	select {
	case <-webhookDone:
		t.Fatal("unexpected second CRM webhook")
	case <-time.After(200 * time.Millisecond):
	}
}
