package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateJSONRejectsMaxTokensFinishReason(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"finishReason": "MAX_TOKENS",
					"content": map[string]any{
						"parts": []map[string]string{
							{"text": `{"icp":"none","hot":false,"spend_tier":"unknown"}`},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	cl, err := NewClient("test-key", "gemini-3.6-flash", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = cl.generateJSON(context.Background(), PriorityHigh, "sys", "user", icpSchema)
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncation error, got %v", err)
	}
}

func TestAnalyzeLeadBatchSplitsOnTruncation(t *testing.T) {
	t.Parallel()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body generateRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		user := body.Contents[0].Parts[0].Text
		itemCount := strings.Count(user, `"id":`)
		finish := "STOP"
		if itemCount > 1 {
			finish = "MAX_TOKENS"
		}
		resp := `{"items":[{"id":"a1","blocked":false,"geo_confidence":"low","icp":"pro","hot":true,"spend_tier":"unknown"}]}`
		if strings.Contains(user, `"id":"b1"`) {
			resp = `{"items":[{"id":"b1","blocked":false,"geo_confidence":"low","icp":"starter","hot":false,"spend_tier":"unknown"}]}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"finishReason": finish,
					"content": map[string]any{
						"parts": []map[string]string{{"text": resp}},
					},
				},
			},
		})
	}))
	defer srv.Close()

	cl, err := NewClient("test-key", "gemini-3.6-flash", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	items := []LeadBatchInput{
		{ID: "a1", Snippet: "voluum alternative"},
		{ID: "b1", Snippet: "keitaro migration"},
	}
	results, err := cl.AnalyzeLeadBatch(context.Background(), items, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results=%d want 2", len(results))
	}
	if calls < 3 {
		t.Fatalf("expected split retry calls >= 3, got %d", calls)
	}
}
