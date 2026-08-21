package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type memSnapshot struct {
	CapturedAt string            `json:"captured_at"`
	Processes  []procMemSnapshot `json:"processes"`
}

type procMemSnapshot struct {
	PID             uint32 `json:"pid"`
	Name            string `json:"name"`
	Role            string `json:"role"`
	VMRSSKB         uint64 `json:"vm_rss_kb"`
	VMHWMKB         uint64 `json:"vm_hwm_kb"`
	VMDataKB        uint64 `json:"vm_data_kb"`
	RssAnonKB       uint64 `json:"rss_anon_kb"`
	Threads         uint64 `json:"threads"`
	OpenFDs         uint64 `json:"open_fds"`
	SocketFDs       uint64 `json:"socket_fds"`
	MinFlt          uint64 `json:"minflt"`
	MajFlt          uint64 `json:"majflt"`
	VoluntaryCtx    uint64 `json:"voluntary_ctxt_switches"`
	NonvoluntaryCtx uint64 `json:"nonvoluntary_ctxt_switches"`
}

func (r *probeRun) writeMemSnapshot(filename string) error {
	snap := memSnapshot{
		CapturedAt: utcNow(),
	}
	r.mu.Lock()
	targets := make([]targetEntry, 0, len(r.tracked))
	for _, t := range r.tracked {
		targets = append(targets, t)
	}
	r.mu.Unlock()

	for _, t := range targets {
		row, err := readProcMem(t)
		if err != nil {
			continue
		}
		if fds, socks, _, err := readProcFDAndThreads(t.PID); err == nil {
			row.OpenFDs = fds
			row.SocketFDs = socks
		}
		snap.Processes = append(snap.Processes, row)
	}

	path := filepath.Join(r.session.Dir, filename)
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readProcMem(t targetEntry) (procMemSnapshot, error) {
	row := procMemSnapshot{
		PID:  t.PID,
		Name: t.Name,
		Role: roleName(t.Role),
	}
	statusPath := fmt.Sprintf("/proc/%d/status", t.PID)
	f, err := os.Open(statusPath)
	if err != nil {
		return row, err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "VmRSS:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.VMRSSKB = v
			}
		case strings.HasPrefix(line, "VmHWM:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.VMHWMKB = v
			}
		case strings.HasPrefix(line, "VmData:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.VMDataKB = v
			}
		case strings.HasPrefix(line, "RssAnon:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.RssAnonKB = v
			}
		case strings.HasPrefix(line, "Threads:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.Threads = v
			}
		case strings.HasPrefix(line, "voluntary_ctxt_switches:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.VoluntaryCtx = v
			}
		case strings.HasPrefix(line, "nonvoluntary_ctxt_switches:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				row.NonvoluntaryCtx = v
			}
		}
	}
	if err := sc.Err(); err != nil {
		return row, err
	}
	if minflt, majflt, err := readProcStatFlt(t.PID); err == nil {
		row.MinFlt = minflt
		row.MajFlt = majflt
	}
	return row, nil
}

func readProcStatFlt(pid uint32) (minflt, majflt uint64, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0, err
	}
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return 0, 0, fmt.Errorf("bad stat")
	}
	fields := strings.Fields(string(data[end+2:]))
	if len(fields) < 11 {
		return 0, 0, fmt.Errorf("short stat")
	}
	minflt, _ = strconv.ParseUint(fields[7], 10, 64)
	majflt, _ = strconv.ParseUint(fields[9], 10, 64)
	return minflt, majflt, nil
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
