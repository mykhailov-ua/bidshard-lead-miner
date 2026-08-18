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

	acceptedTotal = make(map[string]int64)
	junkTotal     = make(map[string]int64)
	geminiWaitSec float64
)

// RecordAccepted increments the accepted leads counter by source and priority.
func RecordAccepted(source, priority string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key := fmt.Sprintf("%s|%s", source, priority)
	acceptedTotal[key]++
}

// RecordJunk increments the junk counter by reason.
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

		fmt.Fprintln(w, "# HELP parser_leads_accepted_total Total accepted leads by source and priority")
		fmt.Fprintln(w, "# TYPE parser_leads_accepted_total counter")
		for key, val := range acceptedTotal {
			parts := strings.SplitN(key, "|", 2)
			source, prio := parts[0], parts[1]
			fmt.Fprintf(w, "parser_leads_accepted_total{source=\"%s\",priority=\"%s\"} %d\n", source, prio, val)
		}

		fmt.Fprintln(w, "# HELP parser_junk_total Total junk captures by reason")
		fmt.Fprintln(w, "# TYPE parser_junk_total counter")
		for reason, val := range junkTotal {
			fmt.Fprintf(w, "parser_junk_total{reason=\"%s\"} %d\n", reason, val)
		}

		fmt.Fprintln(w, "# HELP parser_gemini_wait_seconds Total seconds spent waiting on Gemini rate limits")
		fmt.Fprintln(w, "# TYPE parser_gemini_wait_seconds counter")
		fmt.Fprintf(w, "parser_gemini_wait_seconds %.3f\n", geminiWaitSec)
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
