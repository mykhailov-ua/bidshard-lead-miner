package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type procSampleRow struct {
	TSNs      int64  `json:"ts_ns"`
	PID       uint32 `json:"pid"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	OpenFDs   uint64 `json:"open_fds"`
	SocketFDs uint64 `json:"socket_fds"`
	Threads   uint64 `json:"threads"`
	VMRSSKB   uint64 `json:"vm_rss_kb"`
	VMHWMKB   uint64 `json:"vm_hwm_kb"`
	RssAnonKB uint64 `json:"rss_anon_kb"`
	MinFlt    uint64 `json:"minflt"`
	MajFlt    uint64 `json:"majflt"`
}

type procSamplePeak struct {
	PID          uint32  `json:"pid"`
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	PeakOpenFDs  uint64  `json:"peak_open_fds"`
	PeakSocketFD uint64  `json:"peak_socket_fds"`
	PeakThreads  uint64  `json:"peak_threads"`
	StartOpenFDs uint64  `json:"start_open_fds"`
	EndOpenFDs   uint64  `json:"end_open_fds"`
	FdDelta      int64   `json:"fd_delta"`
	ThreadDelta  int64   `json:"thread_delta"`
	SampleCount  uint64  `json:"sample_count"`
	FdOpenRate   float64 `json:"fd_open_per_sec,omitempty"`
	FdCloseRate  float64 `json:"fd_close_per_sec,omitempty"`
	SocketOpen   uint64  `json:"socket_open"`
	SocketAccept uint64  `json:"socket_accept"`
	ThreadFork   uint64  `json:"thread_fork"`
	ThreadExit   uint64  `json:"thread_exit"`
	StartRSSKB   uint64  `json:"start_rss_kb"`
	EndRSSKB     uint64  `json:"end_rss_kb"`
	PeakRSSKB    uint64  `json:"peak_rss_kb"`
	RSSDelta     int64   `json:"rss_delta"`
	MinFlt       uint64  `json:"min_flt"`
	MajFlt       uint64  `json:"maj_flt"`
}

func (r *probeRun) procSampleLoop(ctx context.Context) {
	defer r.sampleWG.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	outPath := filepath.Join(r.session.Dir, "proc-samples.ndjson")
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
				mem, err := readProcMem(t)
				if err != nil {
					continue
				}
				fds, socks, _, _ := readProcFDAndThreads(t.PID)
				_ = enc.Encode(procSampleRow{
					TSNs:      ts,
					PID:       t.PID,
					Name:      t.Name,
					Role:      roleName(t.Role),
					OpenFDs:   fds,
					SocketFDs: socks,
					Threads:   mem.Threads,
					VMRSSKB:   mem.VMRSSKB,
					VMHWMKB:   mem.VMHWMKB,
					RssAnonKB: mem.RssAnonKB,
					MinFlt:    mem.MinFlt,
					MajFlt:    mem.MajFlt,
				})
			}
		}
	}
}

func aggregateProcSamples(sessionDir string, durationSec float64, bpfStats []dumpedPIDStats) ([]procSamplePeak, error) {
	path := filepath.Join(sessionDir, "proc-samples.ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return peaksFromMemSnapshots(sessionDir, bpfStats, durationSec)
		}
		return nil, err
	}

	type acc struct {
		name         string
		role         string
		peakFD       uint64
		peakSock     uint64
		peakThreads  uint64
		firstFD      uint64
		lastFD       uint64
		firstThreads uint64
		lastThreads  uint64
		firstRSS     uint64
		lastRSS      uint64
		peakRSS      uint64
		minFlt       uint64
		majFlt       uint64
		count        uint64
		seenFirst    bool
	}
	byPID := map[uint32]*acc{}

	for line := range splitLines(data) {
		var row procSampleRow
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		a := byPID[row.PID]
		if a == nil {
			a = &acc{name: row.Name, role: row.Role}
			byPID[row.PID] = a
		}
		if !a.seenFirst {
			a.firstFD = row.OpenFDs
			a.firstThreads = row.Threads
			a.firstRSS = row.VMRSSKB
			a.seenFirst = true
		}
		a.lastFD = row.OpenFDs
		a.lastThreads = row.Threads
		a.lastRSS = row.VMRSSKB
		a.count++
		if row.OpenFDs > a.peakFD {
			a.peakFD = row.OpenFDs
		}
		if row.SocketFDs > a.peakSock {
			a.peakSock = row.SocketFDs
		}
		if row.Threads > a.peakThreads {
			a.peakThreads = row.Threads
		}
		if row.VMRSSKB > a.peakRSS {
			a.peakRSS = row.VMRSSKB
		}
		if row.MinFlt > a.minFlt {
			a.minFlt = row.MinFlt
		}
		if row.MajFlt > a.majFlt {
			a.majFlt = row.MajFlt
		}
	}

	bpfByPID := map[uint32]dumpedPIDStats{}
	for i := range bpfStats {
		bpfByPID[bpfStats[i].PID] = bpfStats[i]
	}

	var out []procSamplePeak
	for pid, a := range byPID {
		peak := procSamplePeak{
			PID:          pid,
			Name:         a.name,
			Role:         a.role,
			PeakOpenFDs:  a.peakFD,
			PeakSocketFD: a.peakSock,
			PeakThreads:  a.peakThreads,
			StartOpenFDs: a.firstFD,
			EndOpenFDs:   a.lastFD,
			FdDelta:      int64(a.lastFD) - int64(a.firstFD),
			ThreadDelta:  int64(a.lastThreads) - int64(a.firstThreads),
			SampleCount:  a.count,
			StartRSSKB:   a.firstRSS,
			EndRSSKB:     a.lastRSS,
			PeakRSSKB:    a.peakRSS,
			RSSDelta:     int64(a.lastRSS) - int64(a.firstRSS),
			MinFlt:       a.minFlt,
			MajFlt:       a.majFlt,
		}
		if st, ok := bpfByPID[pid]; ok && durationSec > 0 {
			peak.FdOpenRate = float64(st.FdOpen) / durationSec
			peak.FdCloseRate = float64(st.FdClose) / durationSec
			peak.SocketOpen = st.SocketOpen
			peak.SocketAccept = st.SocketAccept
			peak.ThreadFork = st.ThreadFork
			peak.ThreadExit = st.ThreadExit
		}
		out = append(out, peak)
	}
	return out, nil
}

func peaksFromMemSnapshots(sessionDir string, bpfStats []dumpedPIDStats, durationSec float64) ([]procSamplePeak, error) {
	memStart := readMemSnap(filepath.Join(sessionDir, "mem-start.json"))
	memEnd := readMemSnap(filepath.Join(sessionDir, "mem-end.json"))

	var out []procSamplePeak
	for i := range bpfStats {
		st := &bpfStats[i]
		peak := procSamplePeak{
			PID:          st.PID,
			Name:         st.Name,
			Role:         st.Role,
			SocketOpen:   st.SocketOpen,
			SocketAccept: st.SocketAccept,
			ThreadFork:   st.ThreadFork,
			ThreadExit:   st.ThreadExit,
		}
		if s, ok := memStart[st.PID]; ok {
			peak.StartOpenFDs = s.OpenFDs
			peak.PeakOpenFDs = s.OpenFDs
			peak.PeakSocketFD = s.SocketFDs
			peak.PeakThreads = s.Threads
			peak.StartRSSKB = s.VMRSSKB
			peak.PeakRSSKB = s.VMRSSKB
			peak.MinFlt = s.MinFlt
			peak.MajFlt = s.MajFlt
		}
		if e, ok := memEnd[st.PID]; ok {
			peak.EndOpenFDs = e.OpenFDs
			peak.EndRSSKB = e.VMRSSKB
			if e.OpenFDs > peak.PeakOpenFDs {
				peak.PeakOpenFDs = e.OpenFDs
			}
			if e.SocketFDs > peak.PeakSocketFD {
				peak.PeakSocketFD = e.SocketFDs
			}
			if e.Threads > peak.PeakThreads {
				peak.PeakThreads = e.Threads
			}
			if e.VMRSSKB > peak.PeakRSSKB {
				peak.PeakRSSKB = e.VMRSSKB
			}
			if e.MinFlt > peak.MinFlt {
				peak.MinFlt = e.MinFlt
			}
			if e.MajFlt > peak.MajFlt {
				peak.MajFlt = e.MajFlt
			}
		}
		peak.FdDelta = int64(peak.EndOpenFDs) - int64(peak.StartOpenFDs)
		peak.RSSDelta = int64(peak.EndRSSKB) - int64(peak.StartRSSKB)
		if durationSec > 0 {
			peak.FdOpenRate = st.FdOpenPerSec
			peak.FdCloseRate = st.FdClosePerSec
		}
		out = append(out, peak)
	}
	return out, nil
}

func readMemSnap(path string) map[uint32]procMemSnapshot {
	out := map[uint32]procMemSnapshot{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var snap memSnapshot
	if json.Unmarshal(data, &snap) != nil {
		return out
	}
	for i := range snap.Processes {
		out[snap.Processes[i].PID] = snap.Processes[i]
	}
	return out
}

func splitLines(data []byte) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		start := 0
		for i := range data {
			if data[i] != '\n' {
				continue
			}
			if i > start {
				ch <- string(data[start:i])
			}
			start = i + 1
		}
		if start < len(data) {
			ch <- string(data[start:])
		}
	}()
	return ch
}

func readProcFDAndThreads(pid uint32) (openFDs, socketFDs, threads uint64, err error) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return 0, 0, 0, err
	}
	openFDs = uint64(len(entries))
	for _, e := range entries {
		link, err := os.Readlink(filepath.Join(fdDir, e.Name()))
		if err != nil {
			continue
		}
		if len(link) >= 6 && link[:6] == "socket" {
			socketFDs++
		}
	}

	row, err := readProcMem(targetEntry{PID: pid})
	if err == nil {
		threads = row.Threads
	}
	return openFDs, socketFDs, threads, nil
}
