package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cilium/ebpf/ringbuf"
)

func drainRingbufRecords(ctx context.Context, rd *ringbuf.Reader, sessionDir string, otel *otelLogExporter) {
	outPath := filepath.Join(sessionDir, "events.ndjson")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Warn("events file", "error", err)
		return
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		rec, err := rd.Read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		e := DecodeSlowEvent(rec.RawSample)
		row := map[string]any{
			"ts_ns":        e.TSNs,
			"pid":          e.PID,
			"role":         roleName(e.Role),
			"syscall_id":   e.SyscallID,
			"syscall_name": syscallName(int(e.SyscallID)),
			"duration_us":  e.DurNs / 1000,
			"kind":         e.Kind,
			"context_slot": e.ContextSlot,
			"marker_id":    e.MarkerID,
			"marker_name":  markerName(e.MarkerID),
		}
		_ = enc.Encode(row)
		if otel != nil {
			otel.emit(otelLogRecord(row))
		}
	}
}

func syscallName(nr int) string {
	names := map[int]string{
		0: "read", 1: "write", 3: "close", 7: "poll", 9: "mmap", 16: "ioctl",
		19: "writev", 41: "socket", 42: "connect", 43: "accept", 44: "sendto", 45: "recvfrom",
		74: "fsync", 75: "fdatasync",
		57: "clone", 202: "futex", 228: "clock_gettime", 230: "exit_group",
		232: "epoll_wait", 233: "epoll_ctl", 257: "openat", 291: "epoll_create1",
		318: "getrandom",
	}
	if n, ok := names[nr]; ok {
		return n
	}
	return fmt.Sprintf("sys_%d", nr)
}

func markerName(id uint32) string {
	switch id {
	case 1:
		return "hot_path_enter"
	case 2:
		return "hot_path_exit"
	default:
		return fmt.Sprintf("marker_%d", id)
	}
}
