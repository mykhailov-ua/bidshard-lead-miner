package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"

	"github.com/bidshard/parser/pkg/bpfenv"
)

const (
	// Must match PARSER_BPF_ROLE_* in deploy/dev/bpf/parser_probe.h and bpf_resolve_targets.sh.
	roleParser   = 1
	roleTelegram = 2
	roleMongo    = 3
	roleLoadgen  = 4
	roleWorker   = 5
)

func main() {
	sessionDir := flag.String("session-dir", "", "session output directory (contains targets.json)")
	bpfObject := flag.String("bpf-object", "", "path to parser_probe.o (default: deploy/dev/bpf/parser_probe.o)")
	sampleRate := flag.Uint("sample-rate", 1, "syscall sample rate (1=every event)")
	slowUs := flag.Uint("slow-us", 10000, "slow syscall threshold microseconds")
	discoverLoadgen := flag.Bool("discover-loadgen", false, "watch for load generator PIDs by /proc comm")
	loadgenComms := flag.String("loadgen-comms", "", "comma-separated /proc comm names (env PARSER_BPF_LOADGEN_COMM)")
	discoverInterval := flag.Duration("discover-interval", 2*time.Second, "dynamic target scan interval")
	dumpInterval := flag.Duration("dump-interval", 0, "periodic maps/summary.json dump (0=disabled)")
	metricsAddr := flag.String("metrics-addr", "", "Prometheus /metrics listen address (empty=disabled)")
	refreshTargets := flag.Duration("refresh-targets", 0, "re-resolve docker/native targets interval (0=disabled)")
	parserBinary := flag.String("parser-binary", "", "parser executable for optional uprobes")
	flag.Parse()

	if *sessionDir == "" {
		fmt.Fprintln(os.Stderr, "session-dir is required")
		os.Exit(2)
	}

	discover := *discoverLoadgen
	comms := *loadgenComms
	if comms == "" {
		comms = bpfenv.Env("LOADGEN_COMM")
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		slog.Error("rlimit", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sess, err := openSession(*sessionDir)
	if err != nil {
		slog.Error("open session", "error", err)
		os.Exit(1)
	}

	objPath := *bpfObject
	if objPath == "" {
		objPath = filepath.Join(repoRoot(), "deploy", "dev", "bpf", "parser_probe.o")
	}

	run := &probeRun{
		session:         sess,
		sampleRate:      uint32(*sampleRate),
		slowNs:          uint64(*slowUs) * 1000,
		discoverLoadgen: discover,
		loadgenComms:    parseLoadgenComms(comms),
		discoverTick:    *discoverInterval,
		dumpInterval:    *dumpInterval,
		metricsAddr:     *metricsAddr,
		refreshTargets:  *refreshTargets,
		rolesWanted:     sess.Meta.RolesWanted,
		bpfObject:       objPath,
		parserBinary:    *parserBinary,
	}
	if err := run.start(ctx); err != nil {
		slog.Error("probe start", "error", err)
		os.Exit(1)
	}
	defer run.stop()

	slog.Info("bpf-collector running", "session", *sessionDir, "object", objPath)
	<-ctx.Done()
	slog.Info("bpf-collector stopping")
}

func repoRoot() string {
	if v := bpfenv.RepoRootFromEnv(); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

type probeRun struct {
	session         *session
	sampleRate      uint32
	slowNs          uint64
	discoverLoadgen bool
	loadgenComms    []string
	discoverTick    time.Duration
	dumpInterval    time.Duration
	metricsAddr     string
	refreshTargets  time.Duration
	rolesWanted     string
	bpfObject       string
	parserBinary    string

	coll     *Collection
	links    []link.Link
	otel     *otelLogExporter
	ringWG   sync.WaitGroup
	sampleWG sync.WaitGroup
	cancel   context.CancelFunc
	tracked  map[uint32]targetEntry
	mu       sync.Mutex
}

func (r *probeRun) start(ctx context.Context) error {
	coll, err := Load(r.bpfObject)
	if err != nil {
		return err
	}
	r.coll = coll

	cfg := Config{
		SampleRate:    r.sampleRate,
		SlowSyscallNs: uint32(minU64(r.slowNs, 0xffffffff)),
		Enabled:       1,
	}
	if err := coll.SetConfig(cfg); err != nil {
		return err
	}

	r.tracked = make(map[uint32]targetEntry)
	for _, t := range sessTargets(r.session) {
		if err := r.trackTarget(t.PID, t.CgroupID, t.Role, t.Name); err != nil {
			slog.Warn("track target", "pid", t.PID, "name", t.Name, "error", err)
		}
	}

	attach := []struct {
		group string
		name  string
		prog  *ebpf.Program
	}{
		{"raw_syscalls", "sys_enter", coll.Progs.SysEnter},
		{"raw_syscalls", "sys_exit", coll.Progs.SysExit},
		{"sched", "sched_wakeup", coll.Progs.SchedWakeup},
		{"sched", "sched_switch", coll.Progs.SchedSwitch},
		{"exceptions", "page_fault_user", coll.Progs.PageFaultUser},
		{"sched", "sched_process_fork", coll.Progs.SchedProcessFork},
		{"sched", "sched_process_exit", coll.Progs.SchedProcessExit},
	}

	for _, a := range attach {
		if a.prog == nil {
			continue
		}
		l, err := link.Tracepoint(a.group, a.name, a.prog, nil)
		if err != nil {
			slog.Warn("tracepoint attach skipped", "group", a.group, "name", a.name, "error", err)
			continue
		}
		r.links = append(r.links, l)
	}

	if coll.Progs.TCPRetransmit != nil {
		if l, err := link.Kprobe("tcp_retransmit_skb", coll.Progs.TCPRetransmit, nil); err != nil {
			slog.Warn("kprobe attach skipped", "symbol", "tcp_retransmit_skb", "error", err)
		} else {
			r.links = append(r.links, l)
		}
	}

	r.attachUprobes()

	r.otel = newOTelLogExporter(ctx, otelEndpointFromEnv())
	if r.otel != nil {
		slog.Info("otel log export enabled", "endpoint", r.otel.endpoint)
	}

	ringCtx, ringCancel := context.WithCancel(ctx)
	r.cancel = ringCancel
	r.ringWG.Add(1)
	go r.drainRingbuf(ringCtx, coll.Maps.SlowEvents)

	r.sampleWG.Add(1)
	go r.procSampleLoop(ringCtx)

	r.sampleWG.Add(1)
	go r.cgroupSampleLoop(ringCtx)

	if r.discoverLoadgen {
		go r.discoverLoop(ringCtx)
	}
	if r.dumpInterval > 0 {
		go r.dumpLoop(ringCtx)
	}
	if r.refreshTargets > 0 {
		go r.refreshTargetsLoop(ringCtx)
	}
	if r.metricsAddr != "" {
		go r.serveMetrics(ringCtx, r.metricsAddr)
	}

	if err := r.writeTimeline(); err != nil {
		slog.Warn("timeline", "error", err)
	}
	return r.writeMemSnapshot("mem-start.json")
}

func (r *probeRun) stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.ringWG.Wait()
	r.sampleWG.Wait()
	if r.otel != nil {
		r.otel.close()
	}
	for _, l := range r.links {
		_ = l.Close()
	}
	if r.coll != nil {
		if err := r.dumpMaps(); err != nil {
			slog.Error("dump maps", "error", err)
		}
		if err := r.writeMemSnapshot("mem-end.json"); err != nil {
			slog.Error("mem-end snapshot", "error", err)
		}
		if err := r.session.markEnded(); err != nil {
			slog.Error("mark ended", "error", err)
		}
		r.coll.Close()
	}
}

func sessTargets(s *session) []targetEntry {
	return s.Meta.Targets
}

func minU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (r *probeRun) drainRingbuf(ctx context.Context, m *ebpf.Map) {
	defer r.ringWG.Done()
	if m == nil {
		return
	}
	rd, err := ringbuf.NewReader(m)
	if err != nil {
		slog.Warn("ringbuf reader", "error", err)
		return
	}
	defer func() { _ = rd.Close() }()
	drainRingbufRecords(ctx, rd, r.session.Dir, r.otel)
}

func (r *probeRun) writeTimeline() error {
	path := filepath.Join(r.session.Dir, "timeline.json")
	started := r.session.Meta.StartedAt.UTC()
	startedUnix := started.Unix()
	promURL := os.Getenv("PROMETHEUS_URL")
	if promURL == "" {
		promURL = "http://127.0.0.1:9190"
	}
	row := map[string]any{
		"started_at":           started.Format(time.RFC3339),
		"started_at_unix":      startedUnix,
		"prometheus_url":       promURL,
		"roles_wanted":         r.rolesWanted,
		"sample_rate":          r.sampleRate,
		"dump_interval_s":      r.dumpInterval.Seconds(),
		"prometheus_query_tpl": "sum(rate(parser_pipeline_accepted_total[5m]))",
		"grafana_url":          os.Getenv("GRAFANA_URL"),
	}
	if row["grafana_url"] == "" {
		row["grafana_url"] = "http://127.0.0.1:3100"
	}
	data, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
