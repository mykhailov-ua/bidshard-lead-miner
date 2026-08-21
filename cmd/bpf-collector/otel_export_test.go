package main

import (
	"testing"
)

func TestOtelLogRecord(t *testing.T) {
	row := map[string]any{
		"role":         "parser",
		"syscall_name": "epoll_wait",
		"marker_name":  "hot_path_enter",
		"duration_us":  12000,
		"pid":          42,
	}
	rec := otelLogRecord(row)
	if rec["severityText"] != "INFO" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestOtelEndpointFromEnv(t *testing.T) {
	t.Setenv("PARSER_BPF_OTEL_ENDPOINT", "http://127.0.0.1:4318")
	if got := otelEndpointFromEnv(); got != "http://127.0.0.1:4318" {
		t.Fatalf("got %q", got)
	}
}
