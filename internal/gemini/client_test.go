package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAnalyzeJunkBatch(t *testing.T) {
	t.Parallel()

	id := primitive.NewObjectID()
	mockID := id.Hex()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]string{
						{"text": `{"items":[{"id":"` + mockID + `","category":"false_negative","why":"mentions voluum billing pain","suggestions":["lower score threshold"]}]}`},
					},
				}},
			},
		})
	}))
	defer srv.Close()

	cl, err := NewClient("test-key", "gemini-2.0-flash", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	results, err := cl.AnalyzeJunkBatch(context.Background(), []sink.JunkDoc{{
		ID:      id,
		Source:  "stub:test",
		Reason:  "low_score",
		Snippet: "voluum alternative billing too high",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d want 1", len(results))
	}
	if results[0].ID != id.Hex() {
		t.Fatalf("id=%q", results[0].ID)
	}
	if results[0].Category != "false_negative" {
		t.Fatalf("category=%q", results[0].Category)
	}
}

func TestBuildJunkReport(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]string{
						{"text": `{"summary":"Most drops are prescan misses.","top_reasons":[{"reason":"keyword_prescan","count":12,"why":"missing synonyms"}],"false_negative_candidates":3,"recommendations":["add voluum synonyms"]}`},
					},
				}},
			},
		})
	}))
	defer srv.Close()

	cl, err := NewClient("test-key", "gemini-2.0-flash", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	result, err := cl.BuildJunkReport(context.Background(), ReportInput{
		PeriodFrom: "2026-01-01T00:00:00Z",
		PeriodTo:   "2026-01-02T00:00:00Z",
		TotalJunk:  20,
		ReasonStats: []sink.ReasonCount{{Reason: "keyword_prescan", Count: 12}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary == "" {
		t.Fatal("expected summary")
	}
	if result.FalseNegativeCandidates != 3 {
		t.Fatalf("false_negative=%d", result.FalseNegativeCandidates)
	}
}
