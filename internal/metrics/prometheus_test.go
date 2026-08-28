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
	RecordWarmAnalysisFailed(2)
	SetLeadsAnalysisPending(7)
	RecordSourcesDiscovered("telegram", 3)
	RecordSourcesTriagedDropped(2)
	RecordProxyEgressBytes("forum", 4096)
	RecordProxyCFBlock("forum", "http_403")
	RecordProxyCooldownWait("forum")
	RecordProxyTransportFail("forum")
	RecordCrawlHTTPFail("forum", 403)
	RecordSERPHarvestFailed()
	RecordGeminiJunkBatchFailed()
	RecordTelethonSidecarFailed()
	RecordICPDrift("tgweb")
	RecordQueueDropped("junk")

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
	if !strings.Contains(content, "parser_warm_analysis_failed_total 2") {
		t.Errorf("expected parser_warm_analysis_failed_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_leads_analysis_pending 7") {
		t.Errorf("expected parser_leads_analysis_pending metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_sources_discovered_total{family=\"telegram\"} 3") {
		t.Errorf("expected parser_sources_discovered_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_sources_triaged_dropped_total 2") {
		t.Errorf("expected parser_sources_triaged_dropped_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_proxy_egress_bytes_total{source=\"forum\"} 4096") {
		t.Errorf("expected parser_proxy_egress_bytes_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_proxy_cf_block_total{source=\"forum\",reason=\"http_403\"}") {
		t.Errorf("expected parser_proxy_cf_block_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_serp_harvest_failed_total 1") {
		t.Errorf("expected parser_serp_harvest_failed_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_telethon_sidecar_failed_total 1") {
		t.Errorf("expected parser_telethon_sidecar_failed_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_icp_drift_total{source=\"tgweb\"}") {
		t.Errorf("expected parser_icp_drift_total metric in output:\n%s", content)
	}
	if !strings.Contains(content, "parser_queue_dropped_total{queue=\"junk\"}") {
		t.Errorf("expected parser_queue_dropped_total metric in output:\n%s", content)
	}
}
