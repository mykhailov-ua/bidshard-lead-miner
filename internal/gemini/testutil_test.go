package gemini

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func writeTestGenerateResponse(w http.ResponseWriter, responseText string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"candidates": []map[string]any{
			{"content": map[string]any{
				"parts": []map[string]string{
					{"text": responseText},
				},
			}},
		},
	})
}

func newTestClient(t testing.TB, responseText string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeTestGenerateResponse(w, responseText)
	}))
	t.Cleanup(srv.Close)
	cl, err := NewClient("test-key", "gemini-2.5-flash", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	return cl
}
