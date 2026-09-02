package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	metricsMu sync.Mutex

	acceptedTotal           = make(map[string]int64)
	leadsWrittenTotal       int64
	tasksDroppedTotal       = make(map[string]int64)
	statsDroppedTotal       int64
	junkTotal               = make(map[string]int64)
	geminiWaitSec           float64
	warmAnalysisFailedTotal int64
	warmAnalysisPending     int64
	leadsAnalysisPending    int64
	sourcesDiscoveredTotal  = make(map[string]int64)
	sourcesTriagedDropped   int64
	processorRejectTotal = make(map[string]int64)
	proxyEgressBytesTotal   = make(map[string]int64)
	proxyBudgetSkippedTotal = make(map[string]int64)
	proxyBudgetExceeded     int64
	proxyBudgetDailyBytes   int64
	headlessQueuedTotal     int64
	headlessDrainedTotal    int64
	icpDriftTotal           = make(map[string]int64)
	queueDroppedTotal       = make(map[string]int64)
	proxyCfBlockTotal       = make(map[string]int64)
	proxyCooldownWaitTotal  = make(map[string]int64)
	proxyTransportFailTotal = make(map[string]int64)
	crawlHTTPFailTotal      = make(map[string]int64)
	serpHarvestFailedTotal  int64
	geminiJunkBatchFailed   int64
	telethonSidecarFailed   int64
)

// EgressCounters is a point-in-time snapshot of egress health counters (auto report).
type EgressCounters struct {
	ProxyCfBlock          int64
	ProxyCooldownWait     int64
	ProxyTransportFail    int64
	CrawlHTTPFail         int64
	SerpHarvestFailed     int64
	GeminiJunkBatchFailed int64
	TelethonSidecarFailed int64
}

// RecordAccepted increments the accepted leads counter by source and priority.
func RecordAccepted(source, priority string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key := fmt.Sprintf("%s|%s", source, priority)
	acceptedTotal[key]++
}

// RecordLeadWritten increments persisted lead counter (Mongo/export upsert success).
func RecordLeadWritten() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	leadsWrittenTotal++
}

// LeadsWrittenTotal returns current leads_written counter (tests).
func LeadsWrittenTotal() int64 {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	return leadsWrittenTotal
}

// RecordTaskDropped increments non-blocking taskCh drops by source.
func RecordTaskDropped(source string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	tasksDroppedTotal[source]++
}

// RecordStatsDropped increments round stats channel drops (reporter backlog).
func RecordStatsDropped() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	statsDroppedTotal++
}

// StatsDroppedTotal returns stats channel drop counter (tests).
func StatsDroppedTotal() int64 {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	return statsDroppedTotal
}

func RecordJunk(reason string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	junkTotal[reason]++
}

// RecordGeminiWait adds wait duration for Gemini calls.
func RecordGeminiWait(d time.Duration) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	geminiWaitSec += d.Seconds()
}

// RecordWarmAnalysisFailed increments leads dropped after warm-path retries.
func RecordWarmAnalysisFailed(n int) {
	if n <= 0 {
		return
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	warmAnalysisFailedTotal += int64(n)
}

// SetWarmAnalysisPending sets Mongo analysis_status=pending gauge (warm-path rescan tick).
func SetWarmAnalysisPending(n int64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	warmAnalysisPending = n
	leadsAnalysisPending = n
}

// SetLeadsAnalysisPending sets defer backlog gauge (runner poll).
func SetLeadsAnalysisPending(n int64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	leadsAnalysisPending = n
	warmAnalysisPending = n
}

// RecordSourcesDiscovered increments registry growth for a source family (serp, telegram, forum, ...).
func RecordSourcesDiscovered(family string, n int) {
	if n <= 0 {
		return
	}
	if family == "" {
		family = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	sourcesDiscoveredTotal[family] += int64(n)
}

// RecordProcessorReject increments scan-round processor reject counters by reason class.
func RecordProcessorReject(reason string) {
	if reason == "" {
		reason = "dropped"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	processorRejectTotal[reason]++
}

// RecordSourcesTriagedDropped increments sources removed by AI/heuristic triage.
func RecordSourcesTriagedDropped(n int) {
	if n <= 0 {
		return
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	sourcesTriagedDropped += int64(n)
}

// RecordProxyEgressBytes adds HTTP response bytes for proxy-routed crawls by source id.
func RecordProxyEgressBytes(source string, n int64) {
	if n <= 0 {
		return
	}
	if source == "" {
		source = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	proxyEgressBytesTotal[source] += n
}

// RecordProxyBudgetSkipped increments when proxy crawl is skipped due to daily cap.
func RecordProxyBudgetSkipped(source string) {
	if source == "" {
		source = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	proxyBudgetSkippedTotal[source]++
}

// SetProxyBudgetExceeded sets gauge when daily proxy cap is reached (1 or 0).
func SetProxyBudgetExceeded(exceeded bool) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	if exceeded {
		proxyBudgetExceeded = 1
		return
	}
	proxyBudgetExceeded = 0
}

// SetProxyBudgetDailyBytes sets today's recorded proxy egress bytes gauge.
func SetProxyBudgetDailyBytes(n int64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	proxyBudgetDailyBytes = n
}

// RecordHeadlessQueued increments deferred headless queue enqueue count.
func RecordHeadlessQueued(n int) {
	if n <= 0 {
		return
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	headlessQueuedTotal += int64(n)
}

// RecordHeadlessDrained increments successful nightly headless drain count.
func RecordHeadlessDrained(n int) {
	if n <= 0 {
		return
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	headlessDrainedTotal += int64(n)
}

func RecordICPDrift(source string) {
	if source == "" {
		source = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	icpDriftTotal[source]++
}

// RecordQueueDropped increments when a non-blocking capturer queue is full.
func RecordQueueDropped(queue string) {
	if queue == "" {
		queue = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	queueDroppedTotal[queue]++
}

// RecordProxyCFBlock increments when a proxy endpoint enters cooldown after a block response.
func RecordProxyCFBlock(source, reason string) {
	if source == "" {
		source = "unknown"
	}
	if reason == "" {
		reason = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key := fmt.Sprintf("%s|%s", source, reason)
	proxyCfBlockTotal[key]++
}

// RecordProxyCooldownWait increments when the proxy pool sleeps waiting for cooldown expiry.
func RecordProxyCooldownWait(source string) {
	if source == "" {
		source = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	proxyCooldownWaitTotal[source]++
}

// RecordProxyTransportFail increments proxy RoundTrip transport errors by crawler source id.
func RecordProxyTransportFail(source string) {
	if source == "" {
		source = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	proxyTransportFailTotal[source]++
}

// RecordCrawlHTTPFail increments final non-OK HTTP responses by source and status code.
func RecordCrawlHTTPFail(source string, status int) {
	if source == "" {
		source = "unknown"
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key := fmt.Sprintf("%s|%d", source, status)
	crawlHTTPFailTotal[key]++
}

// RecordSERPHarvestFailed increments when a SERP dork search fails after retries.
func RecordSERPHarvestFailed() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	serpHarvestFailedTotal++
}

// RecordGeminiJunkBatchFailed increments cold-path junk Gemini batch failures.
func RecordGeminiJunkBatchFailed() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	geminiJunkBatchFailed++
}

// RecordTelethonSidecarFailed increments Telethon scrape/discover sidecar failures.
func RecordTelethonSidecarFailed() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	telethonSidecarFailed++
}

// SnapshotEgress returns summed egress counters for automation snapshots.
func SnapshotEgress() EgressCounters {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	var out EgressCounters
	for _, v := range proxyCfBlockTotal {
		out.ProxyCfBlock += v
	}
	for _, v := range proxyCooldownWaitTotal {
		out.ProxyCooldownWait += v
	}
	for _, v := range proxyTransportFailTotal {
		out.ProxyTransportFail += v
	}
	for _, v := range crawlHTTPFailTotal {
		out.CrawlHTTPFail += v
	}
	out.SerpHarvestFailed = serpHarvestFailedTotal
	out.GeminiJunkBatchFailed = geminiJunkBatchFailed
	out.TelethonSidecarFailed = telethonSidecarFailed
	return out
}

// Handler returns Prometheus exposition text format.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricsMu.Lock()
		defer metricsMu.Unlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		_, _ = fmt.Fprintln(w, "# HELP parser_leads_accepted_total Total accepted leads by source and priority")
		_, _ = fmt.Fprintln(w, "# TYPE parser_leads_accepted_total counter")
		for key, val := range acceptedTotal {
			parts := strings.SplitN(key, "|", 2)
			source, prio := parts[0], parts[1]
			_, _ = fmt.Fprintf(w, "parser_leads_accepted_total{source=\"%s\",priority=\"%s\"} %d\n", source, prio, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_junk_total Total junk captures by reason")
		_, _ = fmt.Fprintln(w, "# TYPE parser_junk_total counter")
		for reason, val := range junkTotal {
			_, _ = fmt.Fprintf(w, "parser_junk_total{reason=\"%s\"} %d\n", reason, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_leads_written_total Total leads persisted to store")
		_, _ = fmt.Fprintln(w, "# TYPE parser_leads_written_total counter")
		_, _ = fmt.Fprintf(w, "parser_leads_written_total %d\n", leadsWrittenTotal)

		_, _ = fmt.Fprintln(w, "# HELP parser_tasks_dropped_total Tasks dropped when taskCh buffer was full")
		_, _ = fmt.Fprintln(w, "# TYPE parser_tasks_dropped_total counter")
		for source, val := range tasksDroppedTotal {
			_, _ = fmt.Fprintf(w, "parser_tasks_dropped_total{source=\"%s\"} %d\n", source, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_stats_dropped_total Scan round stats dropped when statsCh buffer was full")
		_, _ = fmt.Fprintln(w, "# TYPE parser_stats_dropped_total counter")
		_, _ = fmt.Fprintf(w, "parser_stats_dropped_total %d\n", statsDroppedTotal)

		_, _ = fmt.Fprintln(w, "# HELP parser_gemini_wait_seconds Total seconds spent waiting on Gemini rate limits")
		_, _ = fmt.Fprintln(w, "# TYPE parser_gemini_wait_seconds counter")
		_, _ = fmt.Fprintf(w, "parser_gemini_wait_seconds %.3f\n", geminiWaitSec)

		_, _ = fmt.Fprintln(w, "# HELP parser_warm_analysis_failed_total Warm-path leads dropped after retry exhaustion")
		_, _ = fmt.Fprintln(w, "# TYPE parser_warm_analysis_failed_total counter")
		_, _ = fmt.Fprintf(w, "parser_warm_analysis_failed_total %d\n", warmAnalysisFailedTotal)

		_, _ = fmt.Fprintln(w, "# HELP parser_warm_analysis_pending Leads with analysis_status=pending in Mongo")
		_, _ = fmt.Fprintln(w, "# TYPE parser_warm_analysis_pending gauge")
		_, _ = fmt.Fprintf(w, "parser_warm_analysis_pending %d\n", warmAnalysisPending)

		_, _ = fmt.Fprintln(w, "# HELP parser_leads_analysis_pending Deferred Gemini backlog (analysis_status=pending)")
		_, _ = fmt.Fprintln(w, "# TYPE parser_leads_analysis_pending gauge")
		_, _ = fmt.Fprintf(w, "parser_leads_analysis_pending %d\n", leadsAnalysisPending)

		_, _ = fmt.Fprintln(w, "# HELP parser_sources_discovered_total New sources appended to runtime registries")
		_, _ = fmt.Fprintln(w, "# TYPE parser_sources_discovered_total counter")
		for family, val := range sourcesDiscoveredTotal {
			_, _ = fmt.Fprintf(w, "parser_sources_discovered_total{family=\"%s\"} %d\n", family, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_processor_reject_total Processor rejects by reason class per scan round")
		_, _ = fmt.Fprintln(w, "# TYPE parser_processor_reject_total counter")
		for reason, val := range processorRejectTotal {
			_, _ = fmt.Fprintf(w, "parser_processor_reject_total{reason=\"%s\"} %d\n", reason, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_sources_triaged_dropped_total Sources dropped by triage jobs")
		_, _ = fmt.Fprintln(w, "# TYPE parser_sources_triaged_dropped_total counter")
		_, _ = fmt.Fprintf(w, "parser_sources_triaged_dropped_total %d\n", sourcesTriagedDropped)

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_egress_bytes_total HTTP response bytes via proxy by source")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_egress_bytes_total counter")
		for source, val := range proxyEgressBytesTotal {
			_, _ = fmt.Fprintf(w, "parser_proxy_egress_bytes_total{source=\"%s\"} %d\n", source, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_budget_skipped_total Proxy crawl skips when daily cap exceeded")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_budget_skipped_total counter")
		for source, val := range proxyBudgetSkippedTotal {
			_, _ = fmt.Fprintf(w, "parser_proxy_budget_skipped_total{source=\"%s\"} %d\n", source, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_budget_exceeded Daily proxy egress cap reached (1=yes)")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_budget_exceeded gauge")
		_, _ = fmt.Fprintf(w, "parser_proxy_budget_exceeded %d\n", proxyBudgetExceeded)

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_budget_daily_bytes Proxy egress bytes recorded today (UTC)")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_budget_daily_bytes gauge")
		_, _ = fmt.Fprintf(w, "parser_proxy_budget_daily_bytes %d\n", proxyBudgetDailyBytes)

		_, _ = fmt.Fprintln(w, "# HELP parser_headless_queued_total URLs deferred to nightly headless drain queue")
		_, _ = fmt.Fprintln(w, "# TYPE parser_headless_queued_total counter")
		_, _ = fmt.Fprintf(w, "parser_headless_queued_total %d\n", headlessQueuedTotal)

		_, _ = fmt.Fprintln(w, "# HELP parser_headless_drained_total URLs successfully rendered by headless drain")
		_, _ = fmt.Fprintln(w, "# TYPE parser_headless_drained_total counter")
		_, _ = fmt.Fprintf(w, "parser_headless_drained_total %d\n", headlessDrainedTotal)

		_, _ = fmt.Fprintln(w, "# HELP parser_icp_drift_total Inline vs warm-path ICP mismatches by source")
		_, _ = fmt.Fprintln(w, "# TYPE parser_icp_drift_total counter")
		for source, val := range icpDriftTotal {
			_, _ = fmt.Fprintf(w, "parser_icp_drift_total{source=\"%s\"} %d\n", source, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_queue_dropped_total Events dropped when capturer queue was full")
		_, _ = fmt.Fprintln(w, "# TYPE parser_queue_dropped_total counter")
		for queue, val := range queueDroppedTotal {
			_, _ = fmt.Fprintf(w, "parser_queue_dropped_total{queue=\"%s\"} %d\n", queue, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_cf_block_total Proxy endpoints cooled down after block responses")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_cf_block_total counter")
		for key, val := range proxyCfBlockTotal {
			parts := strings.SplitN(key, "|", 2)
			source, reason := parts[0], parts[1]
			_, _ = fmt.Fprintf(w, "parser_proxy_cf_block_total{source=\"%s\",reason=\"%s\"} %d\n", source, reason, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_cooldown_wait_total Proxy pool sleeps while all endpoints cool down")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_cooldown_wait_total counter")
		for source, val := range proxyCooldownWaitTotal {
			_, _ = fmt.Fprintf(w, "parser_proxy_cooldown_wait_total{source=\"%s\"} %d\n", source, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_proxy_transport_fail_total Proxy RoundTrip transport errors")
		_, _ = fmt.Fprintln(w, "# TYPE parser_proxy_transport_fail_total counter")
		for source, val := range proxyTransportFailTotal {
			_, _ = fmt.Fprintf(w, "parser_proxy_transport_fail_total{source=\"%s\"} %d\n", source, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_crawl_http_fail_total Final non-OK HTTP responses by source and status")
		_, _ = fmt.Fprintln(w, "# TYPE parser_crawl_http_fail_total counter")
		for key, val := range crawlHTTPFailTotal {
			parts := strings.SplitN(key, "|", 2)
			source, status := parts[0], parts[1]
			_, _ = fmt.Fprintf(w, "parser_crawl_http_fail_total{source=\"%s\",status=\"%s\"} %d\n", source, status, val)
		}

		_, _ = fmt.Fprintln(w, "# HELP parser_serp_harvest_failed_total SERP dork searches that failed after retries")
		_, _ = fmt.Fprintln(w, "# TYPE parser_serp_harvest_failed_total counter")
		_, _ = fmt.Fprintf(w, "parser_serp_harvest_failed_total %d\n", serpHarvestFailedTotal)

		_, _ = fmt.Fprintln(w, "# HELP parser_gemini_junk_batch_failed_total Cold-path junk Gemini batch failures")
		_, _ = fmt.Fprintln(w, "# TYPE parser_gemini_junk_batch_failed_total counter")
		_, _ = fmt.Fprintf(w, "parser_gemini_junk_batch_failed_total %d\n", geminiJunkBatchFailed)

		_, _ = fmt.Fprintln(w, "# HELP parser_telethon_sidecar_failed_total Telethon scrape/discover sidecar failures")
		_, _ = fmt.Fprintln(w, "# TYPE parser_telethon_sidecar_failed_total counter")
		_, _ = fmt.Fprintf(w, "parser_telethon_sidecar_failed_total %d\n", telethonSidecarFailed)
	})
}

// StartMetricsServer launches HTTP metrics endpoint if addr is non-empty.
func StartMetricsServer(ctx context.Context, addr string) *http.Server {
	if addr == "" {
		return nil
	}
	server := &http.Server{
		Addr:    addr,
		Handler: Handler(),
	}
	go func() {
		_ = server.ListenAndServe()
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	return server
}
