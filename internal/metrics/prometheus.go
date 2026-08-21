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

	acceptedTotal     = make(map[string]int64)
	leadsWrittenTotal int64
	tasksDroppedTotal = make(map[string]int64)
	statsDroppedTotal int64
	junkTotal         = make(map[string]int64)
	geminiWaitSec     float64
)

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
