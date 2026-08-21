package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cilium/ebpf"
)

type dumpedPIDStats struct {
	PID                uint32  `json:"pid"`
	Name               string  `json:"name"`
	Role               string  `json:"role"`
	CtxSwitchOut       uint64  `json:"ctx_switch_out"`
	CtxSwitchIn        uint64  `json:"ctx_switch_in"`
	VoluntaryCtx       uint64  `json:"voluntary_ctx"`
	InvoluntaryCtx     uint64  `json:"involuntary_ctx"`
	RunqueueNs         uint64  `json:"runqueue_ns"`
	RunqueueSamples    uint64  `json:"runqueue_samples"`
	RunqueueAvgUs      float64 `json:"runqueue_avg_us"`
	RunqueueP99Us      float64 `json:"runqueue_p99_us"`
	OnCPUNs            uint64  `json:"oncpu_ns"`
	OnCPUPct           float64 `json:"oncpu_pct"`
	MajorFaults        uint64  `json:"major_faults"`
	MinorFaults        uint64  `json:"minor_faults"`
	FdOpen             uint64  `json:"fd_open"`
	FdClose            uint64  `json:"fd_close"`
	SocketOpen         uint64  `json:"socket_open"`
	SocketAccept       uint64  `json:"socket_accept"`
	ThreadFork         uint64  `json:"thread_fork"`
	ThreadExit         uint64  `json:"thread_exit"`
	FdOpenPerSec       float64 `json:"fd_open_per_sec"`
	FdClosePerSec      float64 `json:"fd_close_per_sec"`
	NetFdEstimate      int64   `json:"net_fd_estimate"`
	CtxSwitchPerSec    float64 `json:"ctx_switch_per_sec"`
	LoadgenOverheadPct float64 `json:"loadgen_overhead_pct,omitempty"`
}

type dumpedSyscall struct {
	PID       uint32  `json:"pid"`
	Role      string  `json:"role"`
	SyscallID uint32  `json:"syscall_id"`
	Syscall   string  `json:"syscall"`
	Count     uint64  `json:"count"`
	SumNs     uint64  `json:"sum_ns"`
	AvgUs     float64 `json:"avg_us"`
	P99Us     float64 `json:"p99_us"`
	MaxUs     float64 `json:"max_us"`
	WallPct   float64 `json:"wall_pct"`
}

type dumpedMarker struct {
	PID         uint32  `json:"pid"`
	Role        string  `json:"role"`
	MarkerID    uint32  `json:"marker_id"`
	Marker      string  `json:"marker"`
	ContextSlot uint32  `json:"context_slot"`
	Count       uint64  `json:"count"`
	AvgUs       float64 `json:"avg_us"`
	P99Us       float64 `json:"p99_us"`
	MaxUs       float64 `json:"max_us"`
}

type dumpBundle struct {
	DurationSec   float64            `json:"duration_sec"`
	PIDStats      []dumpedPIDStats   `json:"pid_stats"`
	ProcSamples   []procSamplePeak   `json:"proc_samples"`
	CgroupSamples []cgroupSamplePeak `json:"cgroup_samples"`
	HotSyscalls   []dumpedSyscall    `json:"hot_syscalls"`
	Syscalls      []dumpedSyscall    `json:"syscalls"`
	Network       []map[string]any   `json:"network"`
	Markers       []dumpedMarker     `json:"markers"`
	HardwarePerf  []map[string]any   `json:"hardware_perf,omitempty"`
}

func (r *probeRun) dumpMaps() error {
	mapsDir := filepath.Join(r.session.Dir, "maps")
	if err := os.MkdirAll(mapsDir, 0o755); err != nil {
		return err
	}

	duration := sessionDuration(r.session)
	pidStats, err := r.aggregatePIDStats(duration)
	if err != nil {
		return err
	}
	procSamples, err := aggregateProcSamples(r.session.Dir, duration, pidStats)
	if err != nil {
		slog.Warn("proc samples", "error", err)
	}
	syscalls, err := r.aggregateSyscalls()
	if err != nil {
		return err
	}
	netStats, err := r.aggregateNet()
	if err != nil {
		return err
	}
	cgroupSamples, err := aggregateCgroupSamples(r.session.Dir)
	if err != nil {
		slog.Warn("cgroup samples", "error", err)
	}
	enrichMajorFaultsFromMem(r.session.Dir, pidStats)
	hotSyscalls := filterHotSyscalls(syscalls)
	markers, err := r.aggregateMarkers()
	if err != nil {
		slog.Warn("marker hist", "error", err)
	}

	bundle := dumpBundle{
		DurationSec:   duration,
		PIDStats:      pidStats,
		ProcSamples:   procSamples,
		CgroupSamples: cgroupSamples,
		HotSyscalls:   hotSyscalls,
		Syscalls:      syscalls,
		Network:       netStats,
		Markers:       markers,
		HardwarePerf:  r.collectHardwarePerf(pidStats),
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(mapsDir, "summary.json"), data, 0o644)
}

func sessionDuration(s *session) float64 {
	start := s.Meta.StartedAt
	end := time.Now().UTC()
	if s.Meta.EndedAt != nil {
		end = *s.Meta.EndedAt
	}
	sec := end.Sub(start).Seconds()
	if sec <= 0 {
		return 1
	}
	return sec
}

func (r *probeRun) aggregatePIDStats(durationSec float64) ([]dumpedPIDStats, error) {
	m := r.coll.Maps.PidStats
	if m == nil {
		return nil, nil
	}

	var out []dumpedPIDStats
	var loadgenOnCPU, trackedOnCPU uint64

	var key uint32
	var perCPU []PIDStats
	iter := m.Iterate()
	for iter.Next(&key, &perCPU) {
		var agg PIDStats
		for i := range perCPU {
			agg.CtxSwitchOut += perCPU[i].CtxSwitchOut
			agg.CtxSwitchIn += perCPU[i].CtxSwitchIn
			agg.VoluntaryCtx += perCPU[i].VoluntaryCtx
			agg.InvoluntaryCtx += perCPU[i].InvoluntaryCtx
			agg.RunqueueNs += perCPU[i].RunqueueNs
			agg.RunqueueSamples += perCPU[i].RunqueueSamples
			agg.OnCPUNs += perCPU[i].OnCPUNs
			agg.MinorFaults += perCPU[i].MinorFaults
			agg.FdOpen += perCPU[i].FdOpen
			agg.FdClose += perCPU[i].FdClose
			agg.SocketOpen += perCPU[i].SocketOpen
			agg.SocketAccept += perCPU[i].SocketAccept
			agg.ThreadFork += perCPU[i].ThreadFork
			agg.ThreadExit += perCPU[i].ThreadExit
		}
		name, role := r.lookupTarget(key, agg.Role)
		row := dumpedPIDStats{
			PID:             key,
			Name:            name,
			Role:            roleName(role),
			CtxSwitchOut:    agg.CtxSwitchOut,
			CtxSwitchIn:     agg.CtxSwitchIn,
			VoluntaryCtx:    agg.VoluntaryCtx,
			InvoluntaryCtx:  agg.InvoluntaryCtx,
			RunqueueNs:      agg.RunqueueNs,
			RunqueueSamples: agg.RunqueueSamples,
			OnCPUNs:         agg.OnCPUNs,
			MinorFaults:     agg.MinorFaults,
			FdOpen:          agg.FdOpen,
			FdClose:         agg.FdClose,
			SocketOpen:      agg.SocketOpen,
			SocketAccept:    agg.SocketAccept,
			ThreadFork:      agg.ThreadFork,
			ThreadExit:      agg.ThreadExit,
			CtxSwitchPerSec: float64(agg.CtxSwitchOut+agg.CtxSwitchIn) / 2 / durationSec,
		}
		if durationSec > 0 {
			row.FdOpenPerSec = float64(agg.FdOpen) / durationSec
			row.FdClosePerSec = float64(agg.FdClose) / durationSec
		}
		row.NetFdEstimate = int64(agg.FdOpen) - int64(agg.FdClose)
		if agg.RunqueueSamples > 0 {
			row.RunqueueAvgUs = float64(agg.RunqueueNs/agg.RunqueueSamples) / 1000
		}
		wallNs := uint64(durationSec * 1e9)
		if wallNs > 0 {
			row.OnCPUPct = float64(agg.OnCPUNs) / float64(wallNs) * 100
		}
		out = append(out, row)
		trackedOnCPU += agg.OnCPUNs
		if role == roleLoadgen {
			loadgenOnCPU += agg.OnCPUNs
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	r.enrichRunqueueP99(out)
	if trackedOnCPU > 0 && loadgenOnCPU > 0 {
		// Share of tracked on-CPU time consumed by loadgen role (attached to loadgen rows only).
		pct := float64(loadgenOnCPU) / float64(trackedOnCPU) * 100
		for i := range out {
			if out[i].Role == "loadgen" {
				out[i].LoadgenOverheadPct = pct
			}
		}
	}
	return out, nil
}

func (r *probeRun) lookupTarget(pid uint32, role uint8) (name string, outRole uint8) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.tracked[pid]; ok {
		return t.Name, t.Role
	}
	return fmt.Sprintf("pid:%d", pid), role
}

func (r *probeRun) aggregateSyscalls() ([]dumpedSyscall, error) {
	m := r.coll.Maps.SyscallHist
	if m == nil {
		return nil, nil
	}

	type acc struct {
		hist Hist
		role uint8
	}
	merged := map[string]*acc{}
	var totalWall uint64

	var key SyscallHistKey
	var perCPU []Hist
	iter := m.Iterate()
	for iter.Next(&key, &perCPU) {
		k := fmt.Sprintf("%d:%d", key.PID, key.SyscallID)
		if merged[k] == nil {
			merged[k] = &acc{}
		}
		for i := range perCPU {
			mergeHist(&merged[k].hist, &perCPU[i])
		}
		_, merged[k].role = r.lookupTarget(key.PID, 0)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	for _, a := range merged {
		totalWall += a.hist.SumNs
	}

	var rows []dumpedSyscall
	for k, a := range merged {
		var pid, sysID uint32
		if _, err := fmt.Sscanf(k, "%d:%d", &pid, &sysID); err != nil {
			continue
		}
		avgUs := float64(0)
		if a.hist.Count > 0 {
			avgUs = float64(a.hist.SumNs/a.hist.Count) / 1000
		}
		row := dumpedSyscall{
			PID:       pid,
			Role:      roleName(a.role),
			SyscallID: sysID,
			Syscall:   syscallName(int(sysID)),
			Count:     a.hist.Count,
			SumNs:     a.hist.SumNs,
			AvgUs:     avgUs,
			P99Us:     histP99Us(&a.hist),
			MaxUs:     float64(a.hist.MaxNs) / 1000,
		}
		if totalWall > 0 {
			row.WallPct = float64(a.hist.SumNs) / float64(totalWall) * 100
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func mergeHist(dst, src *Hist) {
	dst.Count += src.Count
	dst.SumNs += src.SumNs
	if src.MaxNs > dst.MaxNs {
		dst.MaxNs = src.MaxNs
	}
	for i := range dst.Buckets {
		dst.Buckets[i] += src.Buckets[i]
	}
}

func histP99Us(h *Hist) float64 {
	if h.Count == 0 {
		return 0
	}
	target := h.Count - h.Count/100
	if target == 0 {
		target = 1
	}
	var seen uint64
	for i := len(h.Buckets) - 1; i >= 0; i-- {
		seen += h.Buckets[i]
		if seen >= target {
			upper := uint64(1) << i
			return float64(upper) / 1000
		}
	}
	return float64(h.MaxNs) / 1000
}

func (r *probeRun) aggregateNet() ([]map[string]any, error) {
	m := r.coll.Maps.NetStats
	if m == nil {
		return nil, nil
	}
	var out []map[string]any
	var key NetKey
	var perCPU []NetStats
	iter := m.Iterate()
	for iter.Next(&key, &perCPU) {
		var agg NetStats
		for _, v := range perCPU {
			agg.Retrans += v.Retrans
			agg.Connects += v.Connects
			agg.ConnectNsSum += v.ConnectNsSum
			agg.ConnectSamples += v.ConnectSamples
			agg.SendtoCalls += v.SendtoCalls
			agg.SendtoBytes += v.SendtoBytes
		}
		name, role := r.lookupTarget(key.PID, 0)
		connectAvgUs := float64(0)
		if agg.ConnectSamples > 0 {
			connectAvgUs = float64(agg.ConnectNsSum/agg.ConnectSamples) / 1000
		}
		out = append(out, map[string]any{
			"pid":            key.PID,
			"name":           name,
			"role":           roleName(role),
			"dport":          key.Dport,
			"retrans":        agg.Retrans,
			"connects":       agg.Connects,
			"connect_avg_us": connectAvgUs,
			"sendto_calls":   agg.SendtoCalls,
			"sendto_bytes":   agg.SendtoBytes,
		})
	}
	return out, iter.Err()
}

func (r *probeRun) enrichRunqueueP99(stats []dumpedPIDStats) {
	m := r.coll.Maps.RunqueueHist
	if m == nil {
		return
	}
	for i := range stats {
		key := stats[i].PID
		var perCPU []Hist
		if err := m.Lookup(key, &perCPU); err != nil {
			continue
		}
		var agg Hist
		for j := range perCPU {
			mergeHist(&agg, &perCPU[j])
		}
		stats[i].RunqueueP99Us = histP99Us(&agg)
	}
}

func enrichMajorFaultsFromMem(sessionDir string, stats []dumpedPIDStats) {
	start := readMemSnap(filepath.Join(sessionDir, "mem-start.json"))
	end := readMemSnap(filepath.Join(sessionDir, "mem-end.json"))
	for i := range stats {
		s, ok := start[stats[i].PID]
		e, ok2 := end[stats[i].PID]
		if ok && ok2 && e.MajFlt >= s.MajFlt {
			stats[i].MajorFaults = e.MajFlt - s.MajFlt
		}
	}
}

var hotSyscallNames = map[string]bool{
	"read": true, "write": true, "writev": true, "connect": true, "sendto": true,
	"recvfrom": true, "futex": true, "epoll_wait": true, "fsync": true, "fdatasync": true,
}

func filterHotSyscalls(rows []dumpedSyscall) []dumpedSyscall {
	var out []dumpedSyscall
	for _, row := range rows {
		if hotSyscallNames[row.Syscall] {
			out = append(out, row)
		}
	}
	return out
}

func (r *probeRun) aggregateMarkers() ([]dumpedMarker, error) {
	m := r.coll.Maps.MarkerHist
	if m == nil {
		return nil, nil
	}
	var out []dumpedMarker
	var key MarkerHistKey
	var perCPU []Hist
	iter := m.Iterate()
	for iter.Next(&key, &perCPU) {
		var agg Hist
		for i := range perCPU {
			mergeHist(&agg, &perCPU[i])
		}
		if agg.Count == 0 {
			continue
		}
		_, role := r.lookupTarget(key.PID, 0)
		avgUs := float64(agg.SumNs/agg.Count) / 1000
		out = append(out, dumpedMarker{
			PID:         key.PID,
			Role:        roleName(role),
			MarkerID:    key.MarkerID,
			Marker:      markerLabel(key.MarkerID),
			ContextSlot: key.ContextSlot,
			Count:       agg.Count,
			AvgUs:       avgUs,
			P99Us:       histP99Us(&agg),
			MaxUs:       float64(agg.MaxNs) / 1000,
		})
	}
	return out, iter.Err()
}

func markerLabel(id uint32) string {
	switch id {
	case 1:
		return "hot_path_enter"
	case 2:
		return "hot_path_exit"
	default:
		return fmt.Sprintf("marker_%d", id)
	}
}

var _ = ebpf.Map{}
