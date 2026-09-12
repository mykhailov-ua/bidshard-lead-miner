package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestAnalyzeLeadBatchOptsTwoPhaseRPM(t *testing.T) {
	t.Parallel()

	var calls int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		var body generateRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		user := body.Contents[0].Parts[0].Text
		resp := `{"items":[{"id":"low1","blocked":false,"geo_confidence":"low","icp":"none","hot":false,"spend_tier":"unknown"}]}`
		if strings.Contains(user, `"id":"high1"`) && !strings.Contains(user, "Draft outreach") {
			resp = `{"items":[{"id":"low1","blocked":false,"geo_confidence":"low","icp":"none","hot":false,"spend_tier":"unknown"},{"id":"high1","blocked":false,"geo_confidence":"low","icp":"pro","hot":true,"spend_tier":"unknown","pilot_signals":["tracker_pain"],"pilot_qualified":false}]}`
		}
		if strings.Contains(user, "Draft outreach") {
			resp = `{"items":[{"id":"high1","outreach_channel":"telegram","outreach_subject":"","outreach_angle":"tracker pain","outreach_draft":"Hi, saw your postback thread."}]}`
		}
		writeTestGenerateResponse(w, resp)
	}))
	defer srv.Close()

	cl, err := NewClient("test-key", "gemini-3.6-flash", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	items := []LeadBatchInput{
		{ID: "low1", Priority: "Medium", Snippet: "generic affiliate"},
		{ID: "high1", Priority: "High", Snippet: "voluum alternative postback failing", ContactTypes: []string{"telegram"}},
	}
	results, err := cl.AnalyzeLeadBatchOpts(context.Background(), items, LeadBatchOptions{
		GeoClassify: true,
		Engage:      true,
		Enrich:      false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results=%d", len(results))
	}
	if calls != 2 {
		t.Fatalf("expected 2 API calls (core+engage), got %d", calls)
	}
	var high LeadBatchResult
	for _, r := range results {
		if r.HashID == "high1" {
			high = r
		}
	}
	if high.Engagement.OutreachDraft == "" {
		t.Fatal("expected engage draft on high lead")
	}
}
