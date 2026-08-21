package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/bidshard/parser/pkg/bpfenv"
	"github.com/cilium/ebpf"
)

func bpfProgramName(suffix string) string {
	// Object section names are parser_bpf_<suffix> (see deploy/dev/bpf/parser_probe.bpf.c).
	return bpfenv.ProgramPrefix() + suffix
}

type Config struct {
	SampleRate    uint32
	SlowSyscallNs uint32
	Enabled       uint32
	Pad           uint32
}

type Hist struct {
	Buckets [32]uint64
	Count   uint64
	SumNs   uint64
	MaxNs   uint64
}

type PIDStats struct {
	Role            uint8
	Pad             [7]byte
	CtxSwitchOut    uint64
	CtxSwitchIn     uint64
	VoluntaryCtx    uint64
	InvoluntaryCtx  uint64
	RunqueueNs      uint64
	RunqueueSamples uint64
	OnCPUNs         uint64
	LastOnCPUNs     uint64
	MajorFaults     uint64
	MinorFaults     uint64
	FdOpen          uint64
	FdClose         uint64
	SocketOpen      uint64
	SocketAccept    uint64
	ThreadFork      uint64
	ThreadExit      uint64
}

type SyscallHistKey struct {
	PID       uint32
	SyscallID uint32
}

type NetKey struct {
	PID   uint32
	Dport uint16
	Pad   uint16
}

type NetStats struct {
	Connects       uint64
	ConnectNsSum   uint64
	ConnectSamples uint64
	Retrans        uint64
	Rst            uint64
	SendtoCalls    uint64
	SendtoBytes    uint64
}

type MarkerHistKey struct {
	PID         uint32
	MarkerID    uint32
	ContextSlot uint32
	Pad         uint32
}

type Programs struct {
	SysEnter         *ebpf.Program
	SysExit          *ebpf.Program
	SchedWakeup      *ebpf.Program
	SchedSwitch      *ebpf.Program
	PageFaultUser    *ebpf.Program
	TCPRetransmit    *ebpf.Program
	SchedProcessFork *ebpf.Program
	SchedProcessExit *ebpf.Program
	TraceEnter       *ebpf.Program
	TraceExit        *ebpf.Program
}

type Maps struct {
	TargetPids    *ebpf.Map
	TargetCgroups *ebpf.Map
	Config        *ebpf.Map
	SyscallEnter  *ebpf.Map
	SyscallHist   *ebpf.Map
	PidStats      *ebpf.Map
	RunqueueHist  *ebpf.Map
	WakeupTS      *ebpf.Map
	NetStats      *ebpf.Map
	SlowEvents    *ebpf.Map
	MarkerEnterTS *ebpf.Map
	MarkerHist    *ebpf.Map
}

type Collection struct {
	Progs Programs
	Maps  Maps
	raw   *ebpf.Collection
}

func Load(objectPath string) (*Collection, error) {
	data, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("read bpf object %s: %w", objectPath, err)
	}
	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("load bpf spec: %w", err)
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("new bpf collection: %w", err)
	}
	out := &Collection{
		raw: coll,
		Progs: Programs{
			SysEnter:         coll.Programs[bpfProgramName("sys_enter")],
			SysExit:          coll.Programs[bpfProgramName("sys_exit")],
			SchedWakeup:      coll.Programs[bpfProgramName("sched_wakeup")],
			SchedSwitch:      coll.Programs[bpfProgramName("sched_switch")],
			PageFaultUser:    coll.Programs[bpfProgramName("page_fault_user")],
			TCPRetransmit:    coll.Programs[bpfProgramName("tcp_retransmit")],
			SchedProcessFork: coll.Programs[bpfProgramName("sched_process_fork")],
			SchedProcessExit: coll.Programs[bpfProgramName("sched_process_exit")],
			TraceEnter:       coll.Programs[bpfProgramName("trace_enter")],
			TraceExit:        coll.Programs[bpfProgramName("trace_exit")],
		},
		Maps: Maps{
			TargetPids:    coll.Maps["target_pids"],
			TargetCgroups: coll.Maps["target_cgroups"],
			Config:        coll.Maps["config"],
			SyscallEnter:  coll.Maps["syscall_enter"],
			SyscallHist:   coll.Maps["syscall_hist"],
			PidStats:      coll.Maps["pid_stats"],
			RunqueueHist:  coll.Maps["runqueue_hist"],
			WakeupTS:      coll.Maps["wakeup_ts"],
			NetStats:      coll.Maps["net_stats"],
			SlowEvents:    coll.Maps["slow_events"],
			MarkerEnterTS: coll.Maps["marker_enter_ts"],
			MarkerHist:    coll.Maps["marker_hist"],
		},
	}
	if out.Progs.SysEnter == nil || out.Progs.SysExit == nil {
		coll.Close()
		return nil, fmt.Errorf("bpf object missing required programs (rebuild deploy/dev/bpf)")
	}
	return out, nil
}

func (c *Collection) Close() {
	if c.raw != nil {
		c.raw.Close()
	}
}

func (c *Collection) SetConfig(cfg Config) error {
	if c.Maps.Config == nil {
		return fmt.Errorf("config map missing")
	}
	key := uint32(0)
	return c.Maps.Config.Put(key, cfg)
}

func (c *Collection) PutTargetPID(pid uint32, role uint8) error {
	if c.Maps.TargetPids == nil {
		return fmt.Errorf("target_pids map missing")
	}
	return c.Maps.TargetPids.Put(pid, role)
}

func (c *Collection) PutTargetCgroup(cgroupID uint64, role uint8) error {
	if c.Maps.TargetCgroups == nil || cgroupID == 0 {
		return nil
	}
	return c.Maps.TargetCgroups.Put(cgroupID, role)
}

type SlowEvent struct {
	TSNs        uint64
	PID         uint32
	SyscallID   uint32
	DurNs       uint64
	Role        uint8
	Kind        uint8
	ContextSlot uint16
	MarkerID    uint32
}

func DecodeSlowEvent(raw []byte) SlowEvent {
	// Wire layout from BPF ringbuf: ts_ns, pid, syscall_id, dur_ns, role, kind, context_slot, marker_id.
	if len(raw) < 24 {
		return SlowEvent{}
	}
	var e SlowEvent
	e.TSNs = binary.LittleEndian.Uint64(raw[0:8])
	e.PID = binary.LittleEndian.Uint32(raw[8:12])
	e.SyscallID = binary.LittleEndian.Uint32(raw[12:16])
	e.DurNs = binary.LittleEndian.Uint64(raw[16:24])
	if len(raw) >= 26 {
		e.Role = raw[24]
		e.Kind = raw[25]
	}
	if len(raw) >= 28 {
		e.ContextSlot = binary.LittleEndian.Uint16(raw[26:28])
	}
	if len(raw) >= 32 {
		e.MarkerID = binary.LittleEndian.Uint32(raw[28:32])
	}
	return e
}
