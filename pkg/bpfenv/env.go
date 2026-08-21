package bpfenv

import (
	"os"
	"strings"
)

const envPrefix = "PARSER_BPF_"

// RepoRootFromEnv returns repo root for BPF scripts when cwd is not the checkout (make sets PARSER_REPO_ROOT).
func RepoRootFromEnv() string {
	return strings.TrimSpace(os.Getenv("PARSER_REPO_ROOT"))
}

// Env reads PARSER_BPF_<suffix>.
func Env(suffix string) string {
	if v, ok := os.LookupEnv(envPrefix + suffix); ok {
		return v
	}
	return ""
}

// TraceBuildTag is the Go build tag for optional parser uprobes.
func TraceBuildTag() string {
	return "parser_bpf_trace"
}

// MetricPrefix for Prometheus BPF sidecar metrics.
func MetricPrefix() string {
	return "parser_bpf_"
}

// ProgramPrefix matches parser_probe.bpf.c object program names (parser_bpf_*).
func ProgramPrefix() string {
	return "parser_bpf_"
}
