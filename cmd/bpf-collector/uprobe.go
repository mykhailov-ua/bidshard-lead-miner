package main

import (
	"log/slog"

	"github.com/bidshard/parser/pkg/bpfenv"
)

func (r *probeRun) attachUprobes() {
	bin := r.parserBinary
	if bin == "" {
		bin = r.findParserBinary()
	}
	if bin == "" {
		slog.Info("uprobes skipped: parser binary not found (optional: go build -tags " + bpfenv.TraceBuildTag() + " ./cmd/parser)")
		return
	}
	if r.coll == nil {
		return
	}
	// Parser uprobes require traceprobe symbols (parser_bpf_trace build tag). Not wired yet.
	slog.Info("uprobes skipped: parser traceprobe symbols not configured", "binary", bin)
}

func (r *probeRun) findParserBinary() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tracked {
		if t.Role != roleParser || t.PID == 0 {
			continue
		}
		return procExe(t.PID)
	}
	return ""
}

func procExe(pid uint32) string {
	// kept minimal - uprobe path optional for parser analysis sessions
	return ""
}
