package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/httpclient"
)

type otelLogExporter struct {
	endpoint string
	client   *http.Client
	ch       chan map[string]any
	wg       sync.WaitGroup
	ctx      context.Context
}

func newOTelLogExporter(ctx context.Context, endpoint string) *otelLogExporter {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		return nil
	}
	ep = strings.TrimRight(ep, "/")
	if !strings.HasSuffix(ep, "/v1/logs") {
		ep += "/v1/logs"
	}
	e := &otelLogExporter{
		endpoint: ep,
		client:   &http.Client{Timeout: 5 * time.Second},
		ch:       make(chan map[string]any, 256),
		ctx:      ctx,
	}
	e.wg.Add(1)
	go e.loop()
	return e
}

func (e *otelLogExporter) loop() {
	defer e.wg.Done()
	batch := make([]map[string]any, 0, 32)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	flush := func() {
		if len(batch) == 0 {
			return
		}
		payload := map[string]any{
			"resourceLogs": []map[string]any{{
				"scopeLogs": []map[string]any{{
					"logRecords": batch,
				}},
			}},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			slog.Debug("otel marshal", "error", err)
			batch = batch[:0]
			return
		}
		req, err := http.NewRequestWithContext(e.ctx, http.MethodPost, e.endpoint, bytes.NewReader(body))
		if err != nil {
			slog.Debug("otel request", "error", err)
			batch = batch[:0]
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := e.client.Do(req)
		if err != nil {
			slog.Debug("otel export", "error", err)
		} else {
			_, _ = httpclient.ReadResponseBody(resp, 4096)
		}
		batch = batch[:0]
	}
	for {
		select {
		case row, ok := <-e.ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, row)
			if len(batch) >= 32 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (e *otelLogExporter) emit(row map[string]any) {
	select {
	case e.ch <- row:
	default:
		slog.Debug("otel export queue full; dropping slow event")
	}
}

func (e *otelLogExporter) close() {
	if e == nil {
		return
	}
	close(e.ch)
	e.wg.Wait()
}

func otelEndpointFromEnv() string {
	for _, key := range []string{"PARSER_BPF_OTEL_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func otelLogRecord(row map[string]any) map[string]any {
	now := time.Now().UTC().UnixNano()
	body := fmt.Sprintf("bpf slow event role=%v syscall=%v marker=%v duration_us=%v",
		row["role"], row["syscall_name"], row["marker_name"], row["duration_us"])
	attrs := []map[string]any{
		{"key": "bpf.role", "value": map[string]any{"stringValue": fmt.Sprint(row["role"])}},
		{"key": "bpf.syscall", "value": map[string]any{"stringValue": fmt.Sprint(row["syscall_name"])}},
		{"key": "bpf.marker", "value": map[string]any{"stringValue": fmt.Sprint(row["marker_name"])}},
		{"key": "bpf.duration_us", "value": map[string]any{"intValue": fmt.Sprint(row["duration_us"])}},
		{"key": "bpf.pid", "value": map[string]any{"intValue": fmt.Sprint(row["pid"])}},
	}
	return map[string]any{
		"timeUnixNano": fmt.Sprintf("%d", now),
		"severityText": "INFO",
		"body":         map[string]any{"stringValue": body},
		"attributes":   attrs,
	}
}
