package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type cgroupSampleRow struct {
	TSNs             int64  `json:"ts_ns"`
	PID              uint32 `json:"pid"`
	Name             string `json:"name"`
	Role             string `json:"role"`
	CgroupPath       string `json:"cgroup_path"`
	MemoryCurrent    uint64 `json:"memory_current"`
	MemoryAnon       uint64 `json:"memory_anon"`
	MemoryFile       uint64 `json:"memory_file"`
	MemoryMaxEvents  uint64 `json:"memory_max_events"`
	CPUUsageUsec     uint64 `json:"cpu_usage_usec"`
	CPUThrottledUsec uint64 `json:"cpu_throttled_usec"`
	CPUNrThrottled   uint64 `json:"cpu_nr_throttled"`
	IOReadBytes      uint64 `json:"io_read_bytes"`
	IOWriteBytes     uint64 `json:"io_write_bytes"`
}

type cgroupSamplePeak struct {
	PID                uint32  `json:"pid"`
	Name               string  `json:"name"`
	Role               string  `json:"role"`
	PeakMemoryCurrent  uint64  `json:"peak_memory_current"`
	PeakMemoryAnon     uint64  `json:"peak_memory_anon"`
	PeakThrottledUsec  uint64  `json:"peak_cpu_throttled_usec"`
	TotalThrottledUsec uint64  `json:"total_cpu_throttled_usec"`
	ThrottlePct        float64 `json:"cpu_throttle_pct"`
	MemoryMaxEvents    uint64  `json:"memory_max_events"`
	IOReadBytes        uint64  `json:"io_read_bytes"`
	IOWriteBytes       uint64  `json:"io_write_bytes"`
	SampleCount        uint64  `json:"sample_count"`
}

func (r *probeRun) cgroupSampleLoop(ctx context.Context) {
	defer r.sampleWG.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	outPath := filepath.Join(r.session.Dir, "cgroup-samples.ndjson")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			targets := make([]targetEntry, 0, len(r.tracked))
			for _, t := range r.tracked {
				targets = append(targets, t)
			}
			r.mu.Unlock()

			ts := time.Now().UnixNano()
			for _, t := range targets {
				row, err := readCgroupSample(t, ts)
				if err != nil {
					continue
				}
				_ = enc.Encode(row)
			}
		}
	}
}

func readCgroupSample(t targetEntry, ts int64) (cgroupSampleRow, error) {
	row := cgroupSampleRow{
		TSNs: ts,
		PID:  t.PID,
		Name: t.Name,
		Role: roleName(t.Role),
	}
	root, err := cgroupPathForPID(t.PID)
	if err != nil {
		return row, err
	}
	row.CgroupPath = root

	if v, err := readUintFromFile(filepath.Join(root, "memory.current")); err == nil {
		row.MemoryCurrent = v
	}
	row.MemoryAnon = readMemoryStatKey(root, "anon")
	row.MemoryFile = readMemoryStatKey(root, "file")
	row.MemoryMaxEvents = readMemoryEventsKey(root, "max")
	row.CPUUsageUsec, row.CPUThrottledUsec, row.CPUNrThrottled = readCPUStat(root)
	row.IOReadBytes, row.IOWriteBytes = readIOStat(root)
	return row, nil
}

func cgroupPathForPID(pid uint32) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "0::") {
			rel := strings.TrimPrefix(line, "0::")
			if rel == "" {
				return filepath.Join(string(filepath.Separator), "sys", "fs", "cgroup"), nil
			}
			return filepath.Join(string(filepath.Separator), "sys", "fs", "cgroup", rel), nil
		}
	}
	return "", fmt.Errorf("cgroup v2 path not found for pid %d", pid)
}

func readUintFromFile(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func readMemoryStatKey(root, key string) uint64 {
	data, err := os.ReadFile(filepath.Join(root, "memory.stat"))
	if err != nil {
		return 0
	}
	prefix := key + " "
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				return v
			}
		}
	}
	return 0
}

func readMemoryEventsKey(root, key string) uint64 {
	data, err := os.ReadFile(filepath.Join(root, "memory.events"))
	if err != nil {
		return 0
	}
	prefix := key + " "
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				return v
			}
		}
	}
	return 0
}

func readCPUStat(root string) (usageUsec, throttledUsec, nrThrottled uint64) {
	data, err := os.ReadFile(filepath.Join(root, "cpu.stat"))
	if err != nil {
		return 0, 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "usage_usec":
			usageUsec = v
		case "throttled_usec":
			throttledUsec = v
		case "nr_throttled":
			nrThrottled = v
		}
	}
	return usageUsec, throttledUsec, nrThrottled
}

func readIOStat(root string) (readBytes, writeBytes uint64) {
	data, err := os.ReadFile(filepath.Join(root, "io.stat"))
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "rbytes=") {
				v, _ := strconv.ParseUint(strings.TrimPrefix(field, "rbytes="), 10, 64)
				readBytes += v
			}
			if strings.HasPrefix(field, "wbytes=") {
				v, _ := strconv.ParseUint(strings.TrimPrefix(field, "wbytes="), 10, 64)
				writeBytes += v
			}
		}
	}
	return readBytes, writeBytes
}

func aggregateCgroupSamples(sessionDir string) ([]cgroupSamplePeak, error) {
	path := filepath.Join(sessionDir, "cgroup-samples.ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	type acc struct {
		name, role     string
		peakMem        uint64
		peakAnon       uint64
		peakThrottled  uint64
		firstUsage     uint64
		lastUsage      uint64
		firstThrottled uint64
		lastThrottled  uint64
		memMaxEvents   uint64
		firstIORead    uint64
		lastIORead     uint64
		firstIOWrite   uint64
		lastIOWrite    uint64
		count          uint64
		seenFirst      bool
	}
	byPID := map[uint32]*acc{}

	for line := range splitLines(data) {
		var row cgroupSampleRow
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		a := byPID[row.PID]
		if a == nil {
			a = &acc{name: row.Name, role: row.Role}
			byPID[row.PID] = a
		}
		if !a.seenFirst {
			a.firstUsage = row.CPUUsageUsec
			a.firstThrottled = row.CPUThrottledUsec
			a.firstIORead = row.IOReadBytes
			a.firstIOWrite = row.IOWriteBytes
			a.seenFirst = true
		}
		a.lastUsage = row.CPUUsageUsec
		a.lastThrottled = row.CPUThrottledUsec
		a.lastIORead = row.IOReadBytes
		a.lastIOWrite = row.IOWriteBytes
		a.count++
		if row.MemoryCurrent > a.peakMem {
			a.peakMem = row.MemoryCurrent
		}
		if row.MemoryAnon > a.peakAnon {
			a.peakAnon = row.MemoryAnon
		}
		if row.CPUThrottledUsec > a.peakThrottled {
			a.peakThrottled = row.CPUThrottledUsec
		}
		if row.MemoryMaxEvents > a.memMaxEvents {
			a.memMaxEvents = row.MemoryMaxEvents
		}
	}

	var out []cgroupSamplePeak
	for pid, a := range byPID {
		usageDelta := a.lastUsage - a.firstUsage
		throttleDelta := a.lastThrottled - a.firstThrottled
		pct := float64(0)
		if usageDelta > 0 {
			pct = float64(throttleDelta) / float64(usageDelta+throttleDelta) * 100
		}
		out = append(out, cgroupSamplePeak{
			PID:                pid,
			Name:               a.name,
			Role:               a.role,
			PeakMemoryCurrent:  a.peakMem,
			PeakMemoryAnon:     a.peakAnon,
			PeakThrottledUsec:  a.peakThrottled,
			TotalThrottledUsec: throttleDelta,
			ThrottlePct:        pct,
			MemoryMaxEvents:    a.memMaxEvents,
			IOReadBytes:        a.lastIORead - a.firstIORead,
			IOWriteBytes:       a.lastIOWrite - a.firstIOWrite,
			SampleCount:        a.count,
		})
	}
	return out, nil
}
