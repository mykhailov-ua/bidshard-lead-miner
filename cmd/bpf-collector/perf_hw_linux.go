//go:build linux

package main

import (
	"log/slog"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	perfTypeHardware        = 0
	perfCountHWCacheMisses  = 3
	perfCountHWBranchMisses = 5
	perfIOCEnable           = 0x2400
	perfIODisable           = 0x2401
	perfIOCGroups           = 1
)

type hwPerfCounters struct {
	CacheMisses  uint64
	BranchMisses uint64
}

func readHardwarePerf(pid int) (hwPerfCounters, bool) {
	if pid <= 0 {
		return hwPerfCounters{}, false
	}
	cache, okCache := readPerfCounter(pid, perfCountHWCacheMisses)
	branch, okBranch := readPerfCounter(pid, perfCountHWBranchMisses)
	if !okCache && !okBranch {
		return hwPerfCounters{}, false
	}
	return hwPerfCounters{CacheMisses: cache, BranchMisses: branch}, true
}

func readPerfCounter(pid int, config uint64) (uint64, bool) {
	attr := unix.PerfEventAttr{
		Type:   perfTypeHardware,
		Size:   uint32(unsafe.Sizeof(unix.PerfEventAttr{})),
		Config: config,
		Bits:   unix.PerfBitExcludeKernel | unix.PerfBitExcludeHv,
	}
	fd, err := unix.PerfEventOpen(&attr, pid, -1, -1, perfIOCGroups)
	if err != nil {
		slog.Debug("perf_event_open", "pid", pid, "config", config, "error", err)
		return 0, false
	}
	defer func() { _ = unix.Close(fd) }()

	if err := unix.IoctlSetPointerInt(fd, perfIOCEnable, 0); err != nil {
		slog.Debug("perf enable", "error", err)
		return 0, false
	}
	defer func() { _ = unix.IoctlSetPointerInt(fd, perfIODisable, 0) }()

	var count uint64
	if _, err := unix.Read(fd, unsafe.Slice((*byte)(unsafe.Pointer(&count)), 8)); err != nil {
		slog.Debug("perf read", "error", err)
		return 0, false
	}
	return count, true
}

func (r *probeRun) collectHardwarePerf(pidStats []dumpedPIDStats) []map[string]any {
	var out []map[string]any
	for i := range pidStats {
		s := &pidStats[i]
		if s.Role != "parser" && s.Role != "worker" {
			continue
		}
		hw, ok := readHardwarePerf(int(s.PID))
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"pid":           s.PID,
			"name":          s.Name,
			"role":          s.Role,
			"cache_misses":  hw.CacheMisses,
			"branch_misses": hw.BranchMisses,
		})
	}
	return out
}
