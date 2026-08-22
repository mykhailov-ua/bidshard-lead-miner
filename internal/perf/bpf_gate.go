package perf

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// BPFGateConfig holds release gate thresholds (override via env in scripts/lib/bpf_gate.sh).
type BPFGateConfig struct {
	MaxMongoFDatasyncP99Us float64
	MaxParserEpollP99Us    float64
	MaxCPUThrottlePct      float64
	MaxRetrans             int64
	MaxThreadForkDelta     int64
	// Leak probe (proc sample + syscall FD counters in summary.json).
	MaxParserFDDelta       int64
	MaxParserPeakOpenFDs   int64
	MaxParserNetFDEstimate int64
	// Syscall FD counters are noisy on short windows (cold-start open burst). Skip net_fd_estimate
	// and fd_open-close/s checks when duration_sec is below this (0 = always evaluate).
	MinSyscallLeakEvalSec float64
}

func DefaultBPFGateConfig() BPFGateConfig {
	return BPFGateConfig{
		MaxMongoFDatasyncP99Us: 50_000,
		MaxParserEpollP99Us:    100_000,
		MaxCPUThrottlePct:      5,
		MaxRetrans:             0,
		MaxThreadForkDelta:     500,
		MaxParserFDDelta:       32,
		MaxParserPeakOpenFDs:   512,
		MaxParserNetFDEstimate: 64,
	}
}

// LeakGateConfig returns thresholds tuned for soak/leak hunts (stricter FD drift).
func LeakGateConfig() BPFGateConfig {
	cfg := DefaultBPFGateConfig()
	cfg.MaxParserFDDelta = 16
	cfg.MaxParserPeakOpenFDs = 256
	cfg.MaxParserNetFDEstimate = 32
	cfg.MaxThreadForkDelta = 200
	cfg.MinSyscallLeakEvalSec = 30
	return cfg
}

// EvaluateBPFGate returns an error when summary.json exceeds release thresholds.
func EvaluateBPFGate(summaryPath string, cfg BPFGateConfig) error {
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return err
	}
	var summary bpfSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return err
	}
	var fails []string

	for _, s := range summary.HotSyscalls {
		if s.Syscall == "fdatasync" || s.Syscall == "fsync" {
			if strings.EqualFold(s.Role, "mongo") && s.P99Us > cfg.MaxMongoFDatasyncP99Us {
				fails = append(fails, fmt.Sprintf("mongo %s p99=%.0fus (max %.0f)", s.Syscall, s.P99Us, cfg.MaxMongoFDatasyncP99Us))
			}
		}
		if s.Syscall == "epoll_wait" && strings.EqualFold(s.Role, "parser") && s.P99Us > cfg.MaxParserEpollP99Us {
			fails = append(fails, fmt.Sprintf("parser epoll_wait p99=%.0fus (max %.0f)", s.P99Us, cfg.MaxParserEpollP99Us))
		}
	}

	for _, p := range summary.PIDStats {
		if strings.EqualFold(p.Role, "parser") {
			delta := p.ThreadFork - p.ThreadExit
			if delta > cfg.MaxThreadForkDelta {
				fails = append(fails, fmt.Sprintf("parser thread_fork-exit=%d (max %d)", delta, cfg.MaxThreadForkDelta))
			}
		}
	}

	for _, c := range summary.CgroupSamples {
		if c.ThrottlePct > cfg.MaxCPUThrottlePct {
			fails = append(fails, fmt.Sprintf("%s cpu throttle=%.1f%% (max %.1f%%)", c.Role, c.ThrottlePct, cfg.MaxCPUThrottlePct))
		}
		if c.MemoryMaxEvents > 0 {
			fails = append(fails, fmt.Sprintf("%s memory.max events=%d", c.Role, c.MemoryMaxEvents))
		}
	}

	for _, n := range summary.Network {
		if n.Retrans > cfg.MaxRetrans {
			fails = append(fails, fmt.Sprintf("%s tcp retrans=%d (max %d)", n.Role, n.Retrans, cfg.MaxRetrans))
		}
	}

	fails = append(fails, evaluateLeakSignals(summary, cfg)...)

	if len(fails) == 0 {
		return nil
	}
	return fmt.Errorf("bpf release gate: %s", strings.Join(fails, "; "))
}

// evaluateLeakSignals checks proc FD/thread drift (BPF sampler + open/close syscalls).
// Wired by scripts/lib/bpf_leak_gate.sh and PARSER_BPF_LEAK_GATE on tgweb BPF baseline.
func evaluateLeakSignals(summary bpfSummary, cfg BPFGateConfig) []string {
	var fails []string
	for _, p := range summary.ProcSamples {
		if !strings.EqualFold(p.Role, "parser") {
			continue
		}
		if cfg.MaxParserPeakOpenFDs > 0 && p.PeakOpenFDs > cfg.MaxParserPeakOpenFDs {
			fails = append(fails, fmt.Sprintf("parser peak_open_fds=%d (max %d)", p.PeakOpenFDs, cfg.MaxParserPeakOpenFDs))
		}
		if cfg.MaxParserFDDelta > 0 && p.FDDelta > cfg.MaxParserFDDelta {
			fails = append(fails, fmt.Sprintf("parser fd_delta=%d (max %d)", p.FDDelta, cfg.MaxParserFDDelta))
		}
		if cfg.MaxParserFDDelta > 0 && p.FDDelta < -cfg.MaxParserFDDelta {
			fails = append(fails, fmt.Sprintf("parser fd_delta=%d (unexpected close drift)", p.FDDelta))
		}
	}
	for _, p := range summary.PIDStats {
		if !strings.EqualFold(p.Role, "parser") {
			continue
		}
		if cfg.MinSyscallLeakEvalSec > 0 && summary.DurationSec > 0 && summary.DurationSec < cfg.MinSyscallLeakEvalSec {
			continue
		}
		if cfg.MaxParserNetFDEstimate > 0 && p.NetFDEstimate > cfg.MaxParserNetFDEstimate {
			fails = append(fails, fmt.Sprintf("parser net_fd_estimate=%d (max %d)", p.NetFDEstimate, cfg.MaxParserNetFDEstimate))
		}
		openGap := p.FDOpenPerSec - p.FDClosePerSec
		if cfg.MaxParserFDDelta > 0 && openGap > float64(cfg.MaxParserFDDelta) {
			fails = append(fails, fmt.Sprintf("parser fd_open-close/s=%.1f (sustained open > close)", openGap))
		}
	}
	return fails
}
