// Dev-only BPF probe: syscall/sched/net stats for parser, mongo, telegram targets.
#include <linux/bpf.h>
#include <linux/sched.h>
#include <linux/errno.h>

#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#include "parser_pt_regs.h"
#include "parser_probe.h"
#include "parser_trace.h"

char LICENSE[] SEC("license") = "GPL";

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, __u8);
} target_pids SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 128);
	__type(key, __u64);
	__type(value, __u8);
} target_cgroups SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct parser_bpf_config);
} config SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 8192);
	__type(key, __u64);
	__type(value, __u64);
} syscall_enter SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 8192);
	__type(key, __u64);
	__type(value, struct parser_bpf_syscall_peer);
} syscall_peer SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 4096);
	__type(key, struct parser_bpf_syscall_hist_key);
	__type(value, struct parser_bpf_hist);
} syscall_hist SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, struct parser_bpf_pid_stats);
} pid_stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, __u64);
} wakeup_ts SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 2048);
	__type(key, struct parser_bpf_net_key);
	__type(value, struct parser_bpf_net_stats);
} net_stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 20);
} slow_events SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, struct parser_bpf_hist);
} runqueue_hist SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 8192);
	__type(key, struct parser_bpf_marker_ts_key);
	__type(value, __u64);
} marker_enter_ts SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 2048);
	__type(key, struct parser_bpf_marker_hist_key);
	__type(value, struct parser_bpf_hist);
} marker_hist SEC(".maps");

static __always_inline struct parser_bpf_config *probe_config(void)
{
	__u32 key = 0;

	return bpf_map_lookup_elem(&config, &key);
}

static __always_inline struct parser_bpf_net_stats *net_stats_mut(__u32 pid, __u16 dport)
{
	struct parser_bpf_net_key nkey = {};
	struct parser_bpf_net_stats *nst;
	struct parser_bpf_net_stats nfresh = {};

	nkey.pid = pid;
	nkey.dport = dport;
	nst = bpf_map_lookup_elem(&net_stats, &nkey);
	if (!nst) {
		bpf_map_update_elem(&net_stats, &nkey, &nfresh, BPF_NOEXIST);
		nst = bpf_map_lookup_elem(&net_stats, &nkey);
	}
	return nst;
}

static __always_inline struct parser_bpf_pid_stats *pid_stats_mut(__u32 pid, __u8 role)
{
	struct parser_bpf_pid_stats *st;
	struct parser_bpf_pid_stats init = {};

	st = bpf_map_lookup_elem(&pid_stats, &pid);
	if (st)
		return st;
	init.role = role;
	bpf_map_update_elem(&pid_stats, &pid, &init, BPF_NOEXIST);
	return bpf_map_lookup_elem(&pid_stats, &pid);
}

#if defined(__TARGET_ARCH_x86)
static __always_inline __u32 parser_bpf_uprobe_slot(struct pt_regs *ctx)
{
	__u32 slot = (__u32)ctx->di;
	if (!slot)
		slot = (__u32)ctx->ax;
	return slot;
}
#else
static __always_inline __u32 parser_bpf_uprobe_slot(struct pt_regs *ctx)
{
	(void)ctx;
	return 0;
}
#endif

SEC("tracepoint/raw_syscalls/sys_enter")
int parser_bpf_sys_enter(struct trace_event_raw_sys_enter *ctx)
{
	struct parser_bpf_config *cfg;
	__u64 pid_tgid;
	__u32 pid;
	__u64 ts;
	long syscall_id;
	unsigned long arg1;
	unsigned long arg2;
	unsigned long arg4;

	// Read tracepoint fields before BPF helpers; kernel 6.6+ verifier rejects ctx deref after calls.
	syscall_id = ctx->id;
	arg1 = ctx->args[1];
	arg2 = ctx->args[2];
	arg4 = ctx->args[4];

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = pid_tgid >> 32;
	if (!parser_bpf_resolve_role(&target_pids, &target_cgroups, pid))
		return 0;

	if (!parser_bpf_should_sample(cfg) && !parser_bpf_is_hot_syscall(syscall_id))
		return 0;

	ts = bpf_ktime_get_ns();
	bpf_map_update_elem(&syscall_enter, &pid_tgid, &ts, BPF_ANY);

	if (syscall_id == PARSER_BPF_NR_connect || syscall_id == PARSER_BPF_NR_sendto) {
		struct parser_bpf_syscall_peer peer = {};
		void *addr;

		addr = (void *)arg1;
		if (syscall_id == PARSER_BPF_NR_sendto)
			addr = (void *)arg4;
		peer.dport = parser_bpf_read_sockaddr_port(addr);
		if (syscall_id == PARSER_BPF_NR_sendto)
			peer.sendto_len = (__u32)arg2;
		bpf_map_update_elem(&syscall_peer, &pid_tgid, &peer, BPF_ANY);
	}
	return 0;
}

SEC("tracepoint/raw_syscalls/sys_exit")
int parser_bpf_sys_exit(struct trace_event_raw_sys_exit *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_syscall_hist_key hkey;
	struct parser_bpf_hist *hist;
	struct parser_bpf_slow_event ev;
	__u64 *enter_ts;
	__u64 pid_tgid;
	__u32 pid;
	__u64 now;
	__u64 delta;
	__u8 role;
	struct parser_bpf_pid_stats *st;
	long syscall_id;
	long syscall_ret;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	syscall_id = ctx->id;
	syscall_ret = ctx->ret;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = pid_tgid >> 32;
	role = parser_bpf_resolve_role(&target_pids, &target_cgroups, pid);
	if (!role)
		return 0;

	st = pid_stats_mut(pid, role);
	parser_bpf_account_fd_exit(st, syscall_id, syscall_ret);

	enter_ts = bpf_map_lookup_elem(&syscall_enter, &pid_tgid);
	if (!enter_ts)
		return 0;
	now = bpf_ktime_get_ns();
	delta = now - *enter_ts;
	bpf_map_delete_elem(&syscall_enter, &pid_tgid);

	hkey.pid = pid;
	hkey.syscall_id = syscall_id;
	hist = bpf_map_lookup_elem(&syscall_hist, &hkey);
	if (!hist) {
		struct parser_bpf_hist fresh = {};

		bpf_map_update_elem(&syscall_hist, &hkey, &fresh, BPF_NOEXIST);
		hist = bpf_map_lookup_elem(&syscall_hist, &hkey);
	}
	if (hist)
		parser_bpf_hist_record(hist, delta);

	if (syscall_id == PARSER_BPF_NR_connect || syscall_id == PARSER_BPF_NR_sendto) {
		struct parser_bpf_syscall_peer *peer;
		struct parser_bpf_net_stats *nst;
		__u16 dport = 0;

		peer = bpf_map_lookup_elem(&syscall_peer, &pid_tgid);
		if (peer)
			dport = peer->dport;
		bpf_map_delete_elem(&syscall_peer, &pid_tgid);

		if (syscall_id == PARSER_BPF_NR_connect && syscall_ret == 0) {
			nst = net_stats_mut(pid, dport);
			if (nst) {
				nst->connects++;
				nst->connect_ns_sum += delta;
				nst->connect_samples++;
			}
		}
		if (syscall_id == PARSER_BPF_NR_sendto && syscall_ret > 0 && dport == PARSER_BPF_MONGO_PORT) {
			// Count UDP/TCP sendto bytes when peer port is Mongo default (27017).
			nst = net_stats_mut(pid, PARSER_BPF_MONGO_PORT);
			if (nst) {
				nst->sendto_calls++;
				nst->sendto_bytes += (__u64)syscall_ret;
			}
		}
	}

	if (cfg->slow_syscall_ns && delta >= cfg->slow_syscall_ns) {
		__builtin_memset(&ev, 0, sizeof(ev));
		ev.ts_ns = now;
		ev.pid = pid;
		ev.syscall_id = syscall_id;
		ev.duration_ns = delta;
		ev.role = role;
		ev.kind = PARSER_BPF_SLOW_KIND_SYSCALL;
		ev.context_slot = 0;
		ev.marker_id = 0;
		bpf_ringbuf_output(&slow_events, &ev, sizeof(ev), 0);
	}
	return 0;
}

SEC("tracepoint/sched/sched_wakeup")
int parser_bpf_sched_wakeup(struct trace_event_raw_sched_wakeup *ctx)
{
	struct parser_bpf_config *cfg;
	__u32 pid;
	__u64 ts;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid = ctx->pid;
	role = parser_bpf_resolve_role(&target_pids, &target_cgroups, pid);
	if (!role)
		return 0;

	ts = bpf_ktime_get_ns();
	bpf_map_update_elem(&wakeup_ts, &pid, &ts, BPF_ANY);
	return 0;
}

SEC("tracepoint/sched/sched_switch")
int parser_bpf_sched_switch(struct trace_event_raw_sched_switch *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_pid_stats *st;
	struct parser_bpf_hist *rq_hist;
	struct parser_bpf_hist rq_fresh = {};
	__u64 now;
	__u32 prev_pid;
	__u32 next_pid;
	__u8 prev_role;
	__u8 next_role;
	__u64 *wake;
	__u64 wait_ns;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	now = bpf_ktime_get_ns();
	prev_pid = ctx->prev_pid;
	next_pid = ctx->next_pid;
	prev_role = parser_bpf_resolve_role(&target_pids, &target_cgroups, prev_pid);
	next_role = parser_bpf_resolve_role(&target_pids, &target_cgroups, next_pid);

	if (prev_role) {
		st = pid_stats_mut(prev_pid, prev_role);
		if (st) {
			st->ctx_switch_out++;
			if (ctx->prev_state > 0)
				st->voluntary_ctx++;
			else
				st->involuntary_ctx++;
			if (st->last_oncpu_ns)
				st->oncpu_ns += now - st->last_oncpu_ns;
			st->last_oncpu_ns = 0;
		}
	}

	if (next_role) {
		st = pid_stats_mut(next_pid, next_role);
		if (st) {
			st->ctx_switch_in++;
			st->last_oncpu_ns = now;
			wake = bpf_map_lookup_elem(&wakeup_ts, &next_pid);
			if (wake && now > *wake) {
				wait_ns = now - *wake;
				st->runqueue_ns += wait_ns;
				st->runqueue_samples++;
				rq_hist = bpf_map_lookup_elem(&runqueue_hist, &next_pid);
				if (!rq_hist) {
					bpf_map_update_elem(&runqueue_hist, &next_pid, &rq_fresh, BPF_NOEXIST);
					rq_hist = bpf_map_lookup_elem(&runqueue_hist, &next_pid);
				}
				if (rq_hist)
					parser_bpf_hist_record(rq_hist, wait_ns);
			}
		}
	}
	return 0;
}

SEC("tracepoint/exceptions/page_fault_user")
int parser_bpf_page_fault_user(struct trace_event_raw_page_fault_user *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_pid_stats *st;
	__u32 pid;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid = bpf_get_current_pid_tgid() >> 32;
	role = parser_bpf_resolve_role(&target_pids, &target_cgroups, pid);
	if (!role)
		return 0;

	st = pid_stats_mut(pid, role);
	if (st)
		st->minor_faults++;
	return 0;
}

SEC("kprobe/tcp_retransmit_skb")
int parser_bpf_tcp_retransmit(struct pt_regs *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_net_key key = {};
	struct parser_bpf_net_stats *st;
	struct parser_bpf_net_stats fresh = {};
	__u32 pid;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid = bpf_get_current_pid_tgid() >> 32;
	if (!parser_bpf_resolve_role(&target_pids, &target_cgroups, pid))
		return 0;

	key.pid = pid;
	key.dport = 0;
	st = bpf_map_lookup_elem(&net_stats, &key);
	if (!st) {
		bpf_map_update_elem(&net_stats, &key, &fresh, BPF_NOEXIST);
		st = bpf_map_lookup_elem(&net_stats, &key);
	}
	if (st)
		st->retrans++;
	return 0;
}

SEC("tracepoint/sched/sched_process_fork")
int parser_bpf_sched_process_fork(struct trace_event_raw_sched_process_fork *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_pid_stats *st;
	__u32 tgid;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	tgid = bpf_get_current_pid_tgid() >> 32;
	role = parser_bpf_resolve_role(&target_pids, &target_cgroups, tgid);
	if (!role)
		return 0;

	st = pid_stats_mut(tgid, role);
	if (st)
		st->thread_fork++;
	return 0;
}

SEC("tracepoint/sched/sched_process_exit")
int parser_bpf_sched_process_exit(struct trace_event_raw_sched_process_exit *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_pid_stats *st;
	__u32 tgid;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	tgid = bpf_get_current_pid_tgid() >> 32;
	role = parser_bpf_resolve_role(&target_pids, &target_cgroups, tgid);
	if (!role)
		return 0;

	st = pid_stats_mut(tgid, role);
	if (st)
		st->thread_exit++;
	return 0;
}

SEC("uprobe/parser_bpf_trace_enter")
int parser_bpf_trace_enter(struct pt_regs *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_marker_ts_key tkey;
	__u64 cookie;
	__u32 marker_id;
	__u32 slot;
	__u64 pid_tgid;
	__u32 pid;
	__u64 ts;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	cookie = bpf_get_attach_cookie(ctx);
	marker_id = (__u32)cookie;
	if (!marker_id)
		return 0;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = pid_tgid >> 32;
	if (!parser_bpf_resolve_role(&target_pids, &target_cgroups, pid))
		return 0;

	slot = parser_bpf_uprobe_slot(ctx);
	ts = bpf_ktime_get_ns();
	tkey.pid_tgid = pid_tgid;
	tkey.marker_id = marker_id;
	bpf_map_update_elem(&marker_enter_ts, &tkey, &ts, BPF_ANY);
	(void)slot;
	return 0;
}

SEC("uprobe/parser_bpf_trace_exit")
int parser_bpf_trace_exit(struct pt_regs *ctx)
{
	struct parser_bpf_config *cfg;
	struct parser_bpf_marker_ts_key tkey;
	struct parser_bpf_marker_hist_key hkey;
	struct parser_bpf_hist *hist;
	struct parser_bpf_hist fresh = {};
	struct parser_bpf_slow_event ev;
	__u64 *enter_ts;
	__u64 cookie;
	__u32 exit_id;
	__u32 enter_id;
	__u32 slot;
	__u64 pid_tgid;
	__u32 pid;
	__u8 role;
	__u64 now;
	__u64 delta;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	cookie = bpf_get_attach_cookie(ctx);
	exit_id = (__u32)cookie;
	if (exit_id < 2 || (exit_id & 1) == 0)
		return 0;
	enter_id = exit_id - 1;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = pid_tgid >> 32;
	role = parser_bpf_resolve_role(&target_pids, &target_cgroups, pid);
	if (!role)
		return 0;

	slot = parser_bpf_uprobe_slot(ctx);
	tkey.pid_tgid = pid_tgid;
	tkey.marker_id = enter_id;
	enter_ts = bpf_map_lookup_elem(&marker_enter_ts, &tkey);
	if (!enter_ts)
		return 0;
	now = bpf_ktime_get_ns();
	delta = now - *enter_ts;
	bpf_map_delete_elem(&marker_enter_ts, &tkey);

	hkey.pid = pid;
	hkey.marker_id = enter_id;
	hkey.context_slot = slot;
	hist = bpf_map_lookup_elem(&marker_hist, &hkey);
	if (!hist) {
		bpf_map_update_elem(&marker_hist, &hkey, &fresh, BPF_NOEXIST);
		hist = bpf_map_lookup_elem(&marker_hist, &hkey);
	}
	if (hist)
		parser_bpf_hist_record(hist, delta);

	if (cfg->slow_syscall_ns && delta >= cfg->slow_syscall_ns) {
		__builtin_memset(&ev, 0, sizeof(ev));
		ev.ts_ns = now;
		ev.pid = pid;
		ev.duration_ns = delta;
		ev.role = role;
		ev.kind = PARSER_BPF_SLOW_KIND_UPROBE;
		ev.context_slot = (__u16)slot;
		ev.marker_id = enter_id;
		bpf_ringbuf_output(&slow_events, &ev, sizeof(ev), 0);
	}
	return 0;
}
