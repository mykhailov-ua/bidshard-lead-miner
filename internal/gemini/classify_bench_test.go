package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func benchTestClient(b *testing.B) *Client {
	b.Helper()
	srv := httptest.NewServer(newTestClientHandler(b, `{"icp":"starter","hot":true,"spend_tier":"15k-150k","why":"voluum pain"}`))
	b.Cleanup(srv.Close)
	cl, err := NewClient("test-key", "gemini-2.5-flash",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithLimitConfig(LimitConfig{ModelLimits: ModelLimits{RPM: 1_000_000, TPM: 1_000_000, RPD: 1_000_000, MaxRetries: 0}}),
	)
	if err != nil {
		b.Fatal(err)
	}
	return cl
}

func newTestClientHandler(t testing.TB, responseText string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeTestGenerateResponse(w, responseText)
	}
}

func BenchmarkClassifyICP(b *testing.B) {
	if testing.Short() {
		b.Skip("skip gemini bench in -short")
	}
	cl := benchTestClient(b)
	text := strings.Repeat("voluum alternative billing too high for our media team ", 20)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := cl.ClassifyICP(context.Background(), text)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildKeywordDiff(b *testing.B) {
	if testing.Short() {
		b.Skip("skip gemini bench in -short")
	}
	srv := httptest.NewServer(newTestClientHandler(b, `{"add_keywords":[],"add_hard_reject":[],"summary":"ok"}`))
	b.Cleanup(srv.Close)
	cl, err := NewClient("test-key", "gemini-2.5-flash",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithLimitConfig(LimitConfig{ModelLimits: ModelLimits{RPM: 1_000_000, TPM: 1_000_000, RPD: 1_000_000, MaxRetries: 0}}),
	)
	if err != nil {
		b.Fatal(err)
	}
	samples := []string{"voluum pain", "binom migration", "keitaro billing"}
	suggestions := []string{"tracker pain", "postback loss"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := cl.BuildKeywordDiff(context.Background(), suggestions, samples)
		if err != nil {
			b.Fatal(err)
		}
	}
}
