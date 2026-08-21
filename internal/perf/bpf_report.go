package perf

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bidshard/parser/pkg/bpfenv"
)

var ErrNoBPFSummary = errors.New("perf: bpf/maps/summary.json not found")

type bpfSummary struct {
	DurationSec   float64        `json:"duration_sec"`
	PIDStats      []pidStat      `json:"pid_stats"`
	ProcSamples   []procSample   `json:"proc_samples"`
	CgroupSamples []cgroupSample `json:"cgroup_samples"`
	HotSyscalls   []syscallStat  `json:"hot_syscalls"`
	Markers       []markerStat   `json:"markers"`
	Syscalls      []syscallStat  `json:"syscalls"`
	Network       []networkStat  `json:"network"`
	HardwarePerf  []hwPerfStat   `json:"hardware_perf,omitempty"`
}

type hwPerfStat struct {
	PID          int    `json:"pid"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	CacheMisses  uint64 `json:"cache_misses"`
	BranchMisses uint64 `json:"branch_misses"`
}

type pidStat struct {
	PID             int     `json:"pid"`
	Name            string  `json:"name"`
	Role            string  `json:"role"`
	CtxSwitchPerSec float64 `json:"ctx_switch_per_sec"`
	VoluntaryCtx    int64   `json:"voluntary_ctx"`
	InvoluntaryCtx  int64   `json:"involuntary_ctx"`
	OnCPUPct        float64 `json:"oncpu_pct"`
	OnCPUNs         int64   `json:"oncpu_ns"`
	RunqueueAvgUs   float64 `json:"runqueue_avg_us"`
	RunqueueP99Us   float64 `json:"runqueue_p99_us"`
	MinorFaults     int64   `json:"minor_faults"`
	MajorFaults     int64   `json:"major_faults"`
	FDOpenPerSec    float64 `json:"fd_open_per_sec"`
	FDClosePerSec   float64 `json:"fd_close_per_sec"`
	SocketOpen      int64   `json:"socket_open"`
	SocketAccept    int64   `json:"socket_accept"`
	NetFDEstimate   int64   `json:"net_fd_estimate"`
	ThreadFork      int64   `json:"thread_fork"`
	ThreadExit      int64   `json:"thread_exit"`
}

type procSample struct {
	PID           int     `json:"pid"`
	Name          string  `json:"name"`
	Role          string  `json:"role"`
	PeakOpenFDs   int64   `json:"peak_open_fds"`
	PeakSocketFDs int64   `json:"peak_socket_fds"`
	FDDelta       int64   `json:"fd_delta"`
	FDOpenPerSec  float64 `json:"fd_open_per_sec"`
	FDClosePerSec float64 `json:"fd_close_per_sec"`
	SocketOpen    int64   `json:"socket_open"`
	SocketAccept  int64   `json:"socket_accept"`
	PeakThreads   int64   `json:"peak_threads"`
	ThreadDelta   int64   `json:"thread_delta"`
	ThreadFork    int64   `json:"thread_fork"`
	ThreadExit    int64   `json:"thread_exit"`
	StartRSSKB    int64   `json:"start_rss_kb"`
	EndRSSKB      int64   `json:"end_rss_kb"`
	PeakRSSKB     int64   `json:"peak_rss_kb"`
	RSSDelta      int64   `json:"rss_delta"`
	MinFlt        int64   `json:"min_flt"`
	MajFlt        int64   `json:"maj_flt"`
}

type cgroupSample struct {
	PID                   int     `json:"pid"`
	Name                  string  `json:"name"`
	Role                  string  `json:"role"`
	PeakMemoryCurrent     int64   `json:"peak_memory_current"`
	PeakMemoryAnon        int64   `json:"peak_memory_anon"`
	TotalCPUThrottledUsec int64   `json:"total_cpu_throttled_usec"`
	ThrottlePct           float64 `json:"throttle_pct"`
	MemoryMaxEvents       int64   `json:"memory_max_events"`
	IOReadBytes           int64   `json:"io_read_bytes"`
	IOWriteBytes          int64   `json:"io_write_bytes"`
}

type syscallStat struct {
	PID     int     `json:"pid"`
	Role    string  `json:"role"`
	Syscall string  `json:"syscall"`
	Count   int64   `json:"count"`
	AvgUs   float64 `json:"avg_us"`
	P99Us   float64 `json:"p99_us"`
	MaxUs   float64 `json:"max_us"`
	WallPct float64 `json:"wall_pct"`
}

type markerStat struct {
	Role        string  `json:"role"`
	Marker      string  `json:"marker"`
	ContextSlot int     `json:"context_slot"`
	Count       int64   `json:"count"`
	AvgUs       float64 `json:"avg_us"`
	P99Us       float64 `json:"p99_us"`
	MaxUs       float64 `json:"max_us"`
}

type networkStat struct {
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	Dport        uint16  `json:"dport"`
	ConnectAvgUs float64 `json:"connect_avg_us"`
	Connects     int64   `json:"connects"`
	Retrans      int64   `json:"retrans"`
	SendtoCalls  int64   `json:"sendto_calls"`
	SendtoBytes  int64   `json:"sendto_bytes"`
}

type bpfTimeline struct {
	PrometheusURL string `json:"prometheus_url"`
	StartedAt     string `json:"started_at"`
}

type slowEvent struct {
	Kind        int    `json:"kind"`
	Role        string `json:"role"`
	SyscallName string `json:"syscall_name"`
	MarkerName  string `json:"marker_name"`
	MarkerID    int    `json:"marker_id"`
	ContextSlot int    `json:"context_slot"`
	DurationUs  int64  `json:"duration_us"`
	PID         int    `json:"pid"`
}

type memSnapshot struct {
	Processes []struct {
		PID     int `json:"pid"`
		VMRSSKb int `json:"vm_rss_kb"`
	} `json:"processes"`
}

func WriteBPFReport(outDir string) (string, error) {
	// Read bpf/maps/summary.json produced by bpf-collector; write human markdown beside session dir.
	bpfDir := filepath.Join(outDir, "bpf")
	summaryPath := filepath.Join(bpfDir, "maps", "summary.json")
	if _, err := os.Stat(summaryPath); err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoBPFSummary
		}
		return "", err
	}

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return "", err
	}
	var summary bpfSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return "", err
	}

	var timeline bpfTimeline
	timelinePath := filepath.Join(bpfDir, "timeline.json")
	if timelineData, err := os.ReadFile(timelinePath); err == nil {
		_ = json.Unmarshal(timelineData, &timeline)
	}

	reportPath := filepath.Join(outDir, "bpf-report.md")
	var b strings.Builder
	writeBPFReport(&b, bpfDir, &summary, &timeline)
	if err := os.WriteFile(reportPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeBPFReport(b *strings.Builder, bpfDir string, data *bpfSummary, timeline *bpfTimeline) {
	fmt.Fprintf(b, "# BPF Load-Test Report\n\n")
	fmt.Fprintf(b, "Generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(b, "Session dir: `%s`\n", bpfDir)
	fmt.Fprintf(b, "Duration: %.1fs\n", data.DurationSec)
	if timeline.PrometheusURL != "" || timeline.StartedAt != "" {
		prom := timeline.PrometheusURL
		if prom == "" {
			prom = "n/a"
		}
		fmt.Fprintf(b, "Prometheus (session): `%s`\n", prom)
		if timeline.StartedAt != "" {
			fmt.Fprintf(b, "Session started: `%s`\n", timeline.StartedAt)
		}
	}
	b.WriteString("\n")

	loadgenStats := loadgenGroup(data.PIDStats)
	totalOnCPU := int64(0)
	for i := range data.PIDStats {
		totalOnCPU += data.PIDStats[i].OnCPUNs
	}
	loadgenOnCPU := int64(0)
	for i := range loadgenStats {
		loadgenOnCPU += loadgenStats[i].OnCPUNs
	}
	loadgenPct := 0.0
	if totalOnCPU > 0 {
		loadgenPct = float64(loadgenOnCPU) / float64(totalOnCPU) * 100
	}

	b.WriteString("## Load generator overhead\n\n")
	b.WriteString("Load generator context switches and on-CPU time are tracked separately from parser pipeline work.\n\n")
	if len(loadgenStats) > 0 {
		b.WriteString("| process | ctx/s | voluntary | involuntary | on-CPU % | RSS delta |\n")
		b.WriteString("|---------|-------|-----------|-------------|----------|-----------|\n")
		rssStart, rssEnd := readRSSMaps(bpfDir)
		sort.Slice(loadgenStats, func(i, j int) bool {
			return loadgenStats[i].OnCPUNs > loadgenStats[j].OnCPUNs
		})
		for i := range loadgenStats {
			s := &loadgenStats[i]
			delta := rssEnd[s.PID] - rssStart[s.PID]
			name := s.Name
			if name == "" {
				name = fmt.Sprintf("%d", s.PID)
			}
			fmt.Fprintf(b, "| %s | %.0f | %d | %d | %.1f%% | %+d KB |\n",
				name, s.CtxSwitchPerSec, s.VoluntaryCtx, s.InvoluntaryCtx, s.OnCPUPct, delta)
		}
		b.WriteString("\n")
		fmt.Fprintf(b, "- **loadgen share of tracked on-CPU time:** %.1f%%\n", loadgenPct)
	} else {
		b.WriteString("_loadgen not observed (set PARSER_BPF_TRACK_LOADGEN=1 during load)._\n")
	}
	b.WriteString("\n")

	b.WriteString("## Scheduler / context switches (services)\n\n")
	b.WriteString("| Process | Role | ctx/s | runqueue avg (us) | runqueue p99 (us) | on-CPU % | minor flt | major flt |\n")
	b.WriteString("|---------|------|-------|-------------------|-------------------|----------|-----------|-----------|\n")
	serviceStats := filterServicePIDStats(data.PIDStats)
	for i := range serviceStats {
		s := &serviceStats[i]
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("%d", s.PID)
		}
		fmt.Fprintf(b, "| %s | %s | %.0f | %.1f | %.1f | %.1f | %d | %d |\n",
			name, s.Role, s.CtxSwitchPerSec, s.RunqueueAvgUs, s.RunqueueP99Us, s.OnCPUPct, s.MinorFaults, s.MajorFaults)
	}
	b.WriteString("\n")

	if len(data.CgroupSamples) > 0 {
		writeCgroupSection(b, data.CgroupSamples)
	}
	if len(data.HotSyscalls) > 0 {
		writeHotSyscallsSection(b, data.HotSyscalls)
	}
	writeDiskDurabilitySection(b, data.HotSyscalls, data.Syscalls)
	writeFDSection(b, data)
	writeThreadsSection(b, data)
	if len(data.Markers) > 0 {
		writeMarkersSection(b, data.Markers)
	}
	writeSyscallsWallSection(b, data.Syscalls)
	writeSlowEventsSection(b, bpfDir)
	if len(data.Network) > 0 {
		writeNetworkSection(b, data.Network)
	}
	writeBPFInterpretation(b)
}

func loadgenGroup(stats []pidStat) []pidStat {
	var out []pidStat
	for i := range stats {
		if stats[i].Role == "loadgen" {
			out = append(out, stats[i])
		}
	}
	return out
}

func isLoadgenRole(role string) bool {
	return role == "loadgen"
}

func filterServicePIDStats(stats []pidStat) []pidStat {
	var out []pidStat
	for i := range stats {
		// Exclude loadgen from service table; it has its own overhead section.
		if !isLoadgenRole(stats[i].Role) {
			out = append(out, stats[i])
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ri := isLoadgenRole(out[i].Role)
		rj := isLoadgenRole(out[j].Role)
		if ri != rj {
			return ri
		}
		return out[i].CtxSwitchPerSec > out[j].CtxSwitchPerSec
	})
	return out
}

func readRSSMaps(bpfDir string) (rssStart, rssEnd map[int]int) {
	start := make(map[int]int)
	end := make(map[int]int)
	for path, store := range map[string]map[int]int{
		filepath.Join(bpfDir, "mem-start.json"): start,
		filepath.Join(bpfDir, "mem-end.json"):   end,
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var snap memSnapshot
		if json.Unmarshal(data, &snap) != nil {
			continue
		}
		for _, p := range snap.Processes {
			store[p.PID] = p.VMRSSKb
		}
	}
	return start, end
}

func writeCgroupSection(b *strings.Builder, samples []cgroupSample) {
	b.WriteString("## Cgroup limits (CPU throttle & memory)\n\n")
	b.WriteString("From cgroup v2 `cpu.stat`, `memory.current`, `memory.events` sampled every 2s.\n\n")
	b.WriteString("| Container | Role | peak RAM (MiB) | peak anon (MiB) | throttle % | throttled (ms) | mem max events | IO read (MiB) | IO write (MiB) |\n")
	b.WriteString("|-----------|------|----------------|-----------------|------------|----------------|----------------|---------------|----------------|\n")
	sorted := append([]cgroupSample(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PeakMemoryCurrent > sorted[j].PeakMemoryCurrent
	})
	const mib = 1024 * 1024
	for _, s := range sorted {
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("%d", s.PID)
		}
		fmt.Fprintf(b, "| %s | %s | %.0f | %.0f | %.1f | %.0f | %d | %.1f | %.1f |\n",
			name, s.Role,
			float64(s.PeakMemoryCurrent)/mib,
			float64(s.PeakMemoryAnon)/mib,
			s.ThrottlePct,
			float64(s.TotalCPUThrottledUsec)/1000,
			s.MemoryMaxEvents,
			float64(s.IOReadBytes)/mib,
			float64(s.IOWriteBytes)/mib,
		)
	}
	b.WriteString("\n")
}

func writeHotSyscallsSection(b *strings.Builder, hot []syscallStat) {
	b.WriteString("## Hot syscalls (HTTP crawl / Mongo / Telethon)\n\n")
	b.WriteString("Always traced: `epoll_wait`, `read`, `write`, `writev`, `fsync`, `fdatasync`, `connect`, `sendto`, `recvfrom`, `futex`.\n\n")
	b.WriteString("| Role | syscall | count | avg (us) | p99 (us) | max (us) |\n")
	b.WriteString("|------|---------|-------|----------|----------|----------|\n")
	sorted := append([]syscallStat(nil), hot...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Role == "parser" && sorted[j].Role != "parser" {
			return true
		}
		if sorted[i].Role != "parser" && sorted[j].Role == "parser" {
			return false
		}
		return sorted[i].P99Us > sorted[j].P99Us
	})
	for _, s := range sorted {
		fmt.Fprintf(b, "| %s | %s | %d | %.1f | %.1f | %.1f |\n",
			s.Role, s.Syscall, s.Count, s.AvgUs, s.P99Us, s.MaxUs)
	}
	b.WriteString("\n")
}

func writeDiskDurabilitySection(b *strings.Builder, hot, syscalls []syscallStat) {
	diskRows := collectDiskRows(hot, syscalls)
	if len(diskRows) == 0 {
		return
	}
	var writeOps, writevOps, syncOps int64
	for _, s := range diskRows {
		if s.Syscall == "write" || s.Syscall == "writev" {
			writeOps += s.Count
		}
		if s.Syscall == "writev" {
			writevOps += s.Count
		}
		if s.Syscall == "fsync" || s.Syscall == "fdatasync" {
			syncOps += s.Count
		}
	}
	syncReductionPct := 0.0
	if writeOps > 0 && syncOps > 0 {
		syncReductionPct = (1.0 - float64(syncOps)/float64(writeOps)) * 100.0
	}

	b.WriteString("## Durability syscalls (fsync / fdatasync)\n\n")
	b.WriteString("High sync syscall rates on parser or mongo often correlate with JSONL export flush or WiredTiger journal pressure during tgweb crawl.\n\n")
	b.WriteString("| Role | syscall | count | avg (us) | p99 (us) | max (us) |\n")
	b.WriteString("|------|---------|-------|----------|----------|----------|\n")
	sort.Slice(diskRows, func(i, j int) bool {
		if diskRows[i].Count != diskRows[j].Count {
			return diskRows[i].Count > diskRows[j].Count
		}
		return diskRows[i].Syscall < diskRows[j].Syscall
	})
	for _, s := range diskRows {
		fmt.Fprintf(b, "| %s | %s | %d | %.1f | %.1f | %.1f |\n",
			s.Role, s.Syscall, s.Count, s.AvgUs, s.P99Us, s.MaxUs)
	}
	b.WriteString("\n")
	fmt.Fprintf(b, "- **vectored writes (writev):** %d\n", writevOps)
	fmt.Fprintf(b, "- **combined write+writev:** %d\n", writeOps)
	fmt.Fprintf(b, "- **durability sync (fsync+fdatasync):** %d\n", syncOps)
	if writeOps > 0 && syncOps > 0 {
		fmt.Fprintf(b, "- **sync reduction vs write ops: %.1f%%**\n", syncReductionPct)
	}
	b.WriteString("\n")
}

func collectDiskRows(hot, syscalls []syscallStat) []syscallStat {
	seen := make(map[string]struct{})
	var rows []syscallStat
	for _, src := range [][]syscallStat{hot, syscalls} {
		for _, s := range src {
			switch s.Syscall {
			case "write", "writev", "fsync", "fdatasync":
			default:
				continue
			}
			key := fmt.Sprintf("%s:%s:%d", s.Role, s.Syscall, s.PID)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			rows = append(rows, s)
		}
	}
	return rows
}

func writeFDSection(b *strings.Builder, data *bpfSummary) {
	b.WriteString("## File descriptors & sockets\n\n")
	b.WriteString("Open FD counts come from `/proc/pid/fd` sampling every 2s; syscall counters (`openat`, `socket`, `accept`, `close`) are from BPF.\n\n")
	if len(data.ProcSamples) > 0 {
		b.WriteString("| Process | Role | peak FDs | peak sockets | FD delta | open/s | close/s | socket() | accept() |\n")
		b.WriteString("|---------|------|----------|--------------|------|--------|---------|----------|----------|\n")
		sorted := append([]procSample(nil), data.ProcSamples...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].PeakOpenFDs > sorted[j].PeakOpenFDs
		})
		for i := range sorted {
			s := &sorted[i]
			name := s.Name
			if name == "" {
				name = fmt.Sprintf("%d", s.PID)
			}
			fmt.Fprintf(b, "| %s | %s | %d | %d | %+d | %.1f | %.1f | %d | %d |\n",
				name, s.Role, s.PeakOpenFDs, s.PeakSocketFDs, s.FDDelta,
				s.FDOpenPerSec, s.FDClosePerSec, s.SocketOpen, s.SocketAccept)
		}
		b.WriteString("\n")
		return
	}
	if len(data.PIDStats) == 0 {
		return
	}
	b.WriteString("| Process | Role | fd open/s | fd close/s | socket() | accept() | net FD est |\n")
	b.WriteString("|---------|------|-----------|------------|----------|----------|------------|\n")
	sorted := append([]pidStat(nil), data.PIDStats...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FDOpenPerSec > sorted[j].FDOpenPerSec
	})
	for i := range sorted {
		s := &sorted[i]
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("%d", s.PID)
		}
		fmt.Fprintf(b, "| %s | %s | %.1f | %.1f | %d | %d | %+d |\n",
			name, s.Role, s.FDOpenPerSec, s.FDClosePerSec, s.SocketOpen, s.SocketAccept, s.NetFDEstimate)
	}
	b.WriteString("\n")
}

func writeThreadsSection(b *strings.Builder, data *bpfSummary) {
	b.WriteString("## OS threads\n\n")
	b.WriteString("Thread count from `/proc/pid/status`; fork/exit events from `sched_process_{fork,exit}` tracepoints (per process TGID).\n\n")
	if len(data.ProcSamples) > 0 {
		b.WriteString("| Process | Role | peak threads | thread delta | fork events | exit events |\n")
		b.WriteString("|---------|------|--------------|----------|-------------|-------------|\n")
		sorted := append([]procSample(nil), data.ProcSamples...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].PeakThreads > sorted[j].PeakThreads
		})
		for i := range sorted {
			s := &sorted[i]
			name := s.Name
			if name == "" {
				name = fmt.Sprintf("%d", s.PID)
			}
			fmt.Fprintf(b, "| %s | %s | %d | %+d | %d | %d |\n",
				name, s.Role, s.PeakThreads, s.ThreadDelta, s.ThreadFork, s.ThreadExit)
		}
		b.WriteString("\n")
		return
	}
	if len(data.PIDStats) == 0 {
		return
	}
	b.WriteString("| Process | Role | fork events | exit events |\n")
	b.WriteString("|---------|------|-------------|-------------|\n")
	sorted := append([]pidStat(nil), data.PIDStats...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ThreadFork > sorted[j].ThreadFork
	})
	for i := range sorted {
		s := &sorted[i]
		if s.ThreadFork == 0 && s.ThreadExit == 0 {
			continue
		}
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("%d", s.PID)
		}
		fmt.Fprintf(b, "| %s | %s | %d | %d |\n", name, s.Role, s.ThreadFork, s.ThreadExit)
	}
	b.WriteString("\n")
}

func writeMarkersSection(b *strings.Builder, markers []markerStat) {
	b.WriteString("## Hot path uprobes (Go)\n\n")
	fmt.Fprintf(b, "Requires parser built with `-tags %s` and bpf-collector uprobes attached.\n\n", bpfenv.TraceBuildTag())
	b.WriteString("| role | marker | context | count | avg (us) | p99 (us) | max (us) |\n")
	b.WriteString("|------|--------|---------|-------|----------|----------|----------|\n")
	sorted := append([]markerStat(nil), markers...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Role == "parser" && sorted[j].Role != "parser" {
			return true
		}
		if sorted[i].Role != "parser" && sorted[j].Role == "parser" {
			return false
		}
		return sorted[i].P99Us > sorted[j].P99Us
	})
	for _, m := range sorted {
		fmt.Fprintf(b, "| %s | %s | %d | %d | %.1f | %.1f | %.1f |\n",
			m.Role, m.Marker, m.ContextSlot, m.Count, m.AvgUs, m.P99Us, m.MaxUs)
	}
	b.WriteString("\n")
}

func writeSyscallsWallSection(b *strings.Builder, syscalls []syscallStat) {
	b.WriteString("## Syscalls (wall time)\n\n")
	b.WriteString("| Role | syscall | count | avg (us) | p99 (us) | max (us) | wall % |\n")
	b.WriteString("|------|---------|-------|----------|----------|----------|--------|\n")
	sorted := append([]syscallStat(nil), syscalls...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].WallPct > sorted[j].WallPct
	})
	limit := 40
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for _, s := range sorted[:limit] {
		fmt.Fprintf(b, "| %s | %s | %d | %.1f | %.1f | %.1f | %.1f |\n",
			s.Role, s.Syscall, s.Count, s.AvgUs, s.P99Us, s.MaxUs, s.WallPct)
	}
	b.WriteString("\n")
}

func writeSlowEventsSection(b *strings.Builder, bpfDir string) {
	eventsPath := filepath.Join(bpfDir, "events.ndjson")
	f, err := os.Open(eventsPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	var slow []slowEvent
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e slowEvent
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		slow = append(slow, e)
	}
	if err := sc.Err(); err != nil || len(slow) == 0 {
		return
	}
	sort.Slice(slow, func(i, j int) bool {
		return slow[i].DurationUs > slow[j].DurationUs
	})

	var slowSyscall, slowUprobe []slowEvent
	for _, e := range slow {
		if e.Kind == 2 {
			slowUprobe = append(slowUprobe, e)
		} else {
			slowSyscall = append(slowSyscall, e)
		}
	}

	b.WriteString("## Slow events\n\n")
	if len(slowSyscall) > 0 {
		b.WriteString("### Syscalls\n\n")
		b.WriteString("| role | syscall | duration (us) | pid |\n")
		b.WriteString("|------|---------|---------------|-----|\n")
		limit := 25
		if len(slowSyscall) < limit {
			limit = len(slowSyscall)
		}
		for _, e := range slowSyscall[:limit] {
			fmt.Fprintf(b, "| %s | %s | %d | %d |\n", e.Role, e.SyscallName, e.DurationUs, e.PID)
		}
		b.WriteString("\n")
	}
	if len(slowUprobe) > 0 {
		b.WriteString("### Hot path uprobes\n\n")
		b.WriteString("| role | marker | context | duration (us) | pid |\n")
		b.WriteString("|------|--------|------|---------------|-----|\n")
		limit := 25
		if len(slowUprobe) < limit {
			limit = len(slowUprobe)
		}
		for _, e := range slowUprobe[:limit] {
			marker := e.MarkerName
			if marker == "" {
				marker = fmt.Sprintf("%d", e.MarkerID)
			}
			fmt.Fprintf(b, "| %s | %s | %d | %d | %d |\n",
				e.Role, marker, e.ContextSlot, e.DurationUs, e.PID)
		}
		b.WriteString("\n")
	}
	if len(slowSyscall) == 0 && len(slowUprobe) == 0 {
		b.WriteString("_No slow events above threshold._\n\n")
	}
}

func writeNetworkSection(b *strings.Builder, network []networkStat) {
	b.WriteString("## Network (connect latency & TCP retrans)\n\n")
	b.WriteString("| process | role | dport | connect avg (us) | connects | sendto calls | sendto bytes | retrans |\n")
	b.WriteString("|---------|------|-------|------------------|----------|--------------|--------------|---------|\n")
	sorted := append([]networkStat(nil), network...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Retrans == sorted[j].Retrans {
			return sorted[i].SendtoBytes > sorted[j].SendtoBytes
		}
		return sorted[i].Retrans > sorted[j].Retrans
	})
	for _, n := range sorted {
		if n.Retrans == 0 && n.Connects == 0 && n.SendtoCalls == 0 {
			continue
		}
		fmt.Fprintf(b, "| %s | %s | %d | %.1f | %d | %d | %d | %d |\n",
			n.Name, n.Role, n.Dport, n.ConnectAvgUs, n.Connects, n.SendtoCalls, n.SendtoBytes, n.Retrans)
	}
	b.WriteString("\n")
}

func writeBPFInterpretation(b *strings.Builder) {
	b.WriteString("## Interpretation\n\n")
	b.WriteString("1. **loadgen on-CPU > 15%** - synthetic load competes with parser; lower crawl concurrency or move loadgen off-host.\n")
	b.WriteString("2. **parser connect/sendto wall %** - outbound HTTP proxy or Mongo wire traffic; check `PARSER_PROXY_LIST` and `MONGO_URI`.\n")
	b.WriteString("3. **involuntary ctx >> voluntary** - CPU oversubscription (GOMAXPROCS, compose CPU limits).\n")
	b.WriteString("4. **cpu throttle % > 5%** - cgroup CPU limit is biting; raise compose cpus or lower crawl parallelism.\n")
	b.WriteString("5. **memory max events > 0** - container hit memory.max; risk of OOM kill during tgweb batch.\n")
	b.WriteString("6. **parser epoll_wait p99 high** - poll wait dominates; check open connections and proxy latency.\n")
	b.WriteString("7. **peak FDs growing / high open rate** - HTTP client leak or missing body close; compare peak sockets with proxy pool size.\n")
	b.WriteString("8. **thread_fork >> thread_exit or peak threads climbing** - goroutine growth; check worker pool and Telethon sidecar churn.\n")
	b.WriteString("9. **retrans > 0** - kernel TCP retry; check proxy egress or Mongo network path.\n")
	b.WriteString("10. **hot_path_enter p99 (uprobe)** - optional parser hot span when built with `-tags parser_bpf_trace`.\n")
	b.WriteString("11. **mongo fdatasync p99** - disk pressure on WiredTiger journal during bulk lead writes.\n")
	b.WriteString("\n")
}
