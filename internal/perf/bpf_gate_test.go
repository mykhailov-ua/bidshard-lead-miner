package perf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateBPFGatePass(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "bpf", "maps", "summary.json")
	if err := EvaluateBPFGate(path, DefaultBPFGateConfig()); err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestEvaluateBPFLeakGateFailFDDelta(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "summary.json")
	payload := []byte(`{
  "proc_samples": [{
    "role": "parser",
    "name": "parser",
    "peak_open_fds": 300,
    "fd_delta": 40
  }],
  "pid_stats": []
}`)
	if err := os.WriteFile(summaryPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EvaluateBPFGate(summaryPath, LeakGateConfig()); err == nil {
		t.Fatal("expected leak gate failure for fd_delta")
	}
}

func TestEvaluateBPFGateFailMongoSync(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "summary.json")
	payload := []byte(`{
  "duration_sec": 60,
  "hot_syscalls": [{
    "role": "mongo",
    "syscall": "fdatasync",
    "count": 40,
    "p99_us": 52000
  }]
}`)
	if err := os.WriteFile(summaryPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultBPFGateConfig()
	cfg.MaxMongoFDatasyncP99Us = 1000
	if err := EvaluateBPFGate(summaryPath, cfg); err == nil {
		t.Fatal("expected gate failure for high mongo fdatasync p99")
	}
}
