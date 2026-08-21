package metrics

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsExposition(t *testing.T) {
	RecordAccepted("stub", "High")
	RecordLeadWritten()
	RecordTaskDropped("forum")
	RecordStatsDropped()
	RecordJunk("geo_reject")
	RecordGeminiWait(500 * time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	Handler().ServeHTTP(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	content := string(body)
	if !strings.Contains(content, "parser_leads_accepted_total{source=\"stub\",priority=\"High\"}") {
		t.Errorf("expected parser_leads_accepted_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_junk_total{reason=\"geo_reject\"}") {
		t.Errorf("expected parser_junk_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_gemini_wait_seconds") {
		t.Errorf("expected parser_gemini_wait_seconds metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_leads_written_total") {
		t.Errorf("expected parser_leads_written_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_tasks_dropped_total{source=\"forum\"}") {
		t.Errorf("expected parser_tasks_dropped_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_stats_dropped_total") {
		t.Errorf("expected parser_stats_dropped_total metric in output:\n%s", content)
	}
}
