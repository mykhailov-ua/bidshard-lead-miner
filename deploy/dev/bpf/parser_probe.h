// Shared types and helpers for parser dev BPF probe (syscall/sched/net).
#ifndef PARSER_PROBE_H
#define PARSER_PROBE_H

// Role ids must match scripts/perf/bpf_resolve_targets.sh and cmd/bpf-collector/track.go
#define PARSER_BPF_ROLE_PARSER 1
#define PARSER_BPF_ROLE_TELEGRAM 2
#define PARSER_BPF_ROLE_MONGO 3
#define PARSER_BPF_ROLE_LOADGEN 4
#define PARSER_BPF_ROLE_WORKER 5

// Optional uprobe marker pairs (enter id odd, exit id = enter + 1).
#define PARSER_BPF_MARKER_HOT_ENTER 1
#define PARSER_BPF_MARKER_HOT_EXIT 2

#define PARSER_BPF_SLOW_KIND_SYSCALL 1
#define PARSER_BPF_SLOW_KIND_UPROBE 2

#define PARSER_BPF_HIST_BUCKETS 32
#define PARSER_BPF_SLOW_SYSCALL_NS 10000000ULL
#define PARSER_BPF_DEFAULT_SAMPLE_RATE 1

struct parser_bpf_hist {
	__u64 buckets[PARSER_BPF_HIST_BUCKETS];
	__u64 count;
	__u64 sum_ns;
	__u64 max_ns;
};

struct parser_bpf_pid_stats {
	__u8 role;
	__u8 _pad[7];
	__u64 ctx_switch_out;
	__u64 ctx_switch_in;
	__u64 voluntary_ctx;
	__u64 involuntary_ctx;
	__u64 runqueue_ns;
	__u64 runqueue_samples;
	__u64 oncpu_ns;
	__u64 last_oncpu_ns;
	__u64 major_faults;
	__u64 fd_open;
	__u64 fd_close;
	__u64 socket_open;
	__u64 socket_accept;
	__u64 thread_fork;
	__u64 thread_exit;
	__u64 minor_faults;
};

// x86_64 syscall numbers used by hot-path filter and connect tracking.
#define PARSER_BPF_NR_read 0
#define PARSER_BPF_NR_write 1
#define PARSER_BPF_NR_writev 19
#define PARSER_BPF_NR_connect 42
#define PARSER_BPF_NR_fsync 74
#define PARSER_BPF_NR_fdatasync 75
#define PARSER_BPF_NR_sendto 44
#define PARSER_BPF_NR_recvfrom 45
#define PARSER_BPF_NR_futex 202
#define PARSER_BPF_NR_epoll_wait 232
#define PARSER_BPF_NR_close 3
#define PARSER_BPF_NR_dup 32
#define PARSER_BPF_NR_socket 41
#define PARSER_BPF_NR_accept 43
#define PARSER_BPF_NR_openat 257
#define PARSER_BPF_NR_accept4 288
#define PARSER_BPF_NR_dup3 292
#define PARSER_BPF_NR_pipe2 293

#define PARSER_BPF_AF_INET 2
#define PARSER_BPF_MONGO_PORT 27017

static __always_inline __u16 parser_bpf_read_sockaddr_port(void *addr)
{
	__u16 family;
	__u16 port_be;

	if (!addr)
		return 0;
	if (bpf_probe_read_user(&family, sizeof(family), addr) < 0)
		return 0;
	if (family != PARSER_BPF_AF_INET)
		return 0;
	if (bpf_probe_read_user(&port_be, sizeof(port_be), (char *)addr + 2) < 0)
		return 0;
	return bpf_ntohs(port_be);
}

static __always_inline int parser_bpf_is_hot_syscall(long syscall_id)
{
	switch (syscall_id) {
	case PARSER_BPF_NR_read:
	case PARSER_BPF_NR_write:
	case PARSER_BPF_NR_writev:
	case PARSER_BPF_NR_fsync:
	case PARSER_BPF_NR_fdatasync:
	case PARSER_BPF_NR_connect:
	case PARSER_BPF_NR_sendto:
	case PARSER_BPF_NR_recvfrom:
	case PARSER_BPF_NR_futex:
	case PARSER_BPF_NR_epoll_wait:
		return 1;
	default:
		return 0;
	}
}

// Count successful open/close syscalls for leak gates (paired with /proc fd sampling in collector).
static __always_inline void parser_bpf_account_fd_exit(struct parser_bpf_pid_stats *st, long syscall_id, long ret)
{
	if (!st)
		return;
	if (syscall_id == PARSER_BPF_NR_close) {
		if (ret == 0)
			st->fd_close++;
		return;
	}
	if (ret < 0)
		return;
	switch (syscall_id) {
	case PARSER_BPF_NR_openat:
	case PARSER_BPF_NR_dup:
	case PARSER_BPF_NR_dup3:
	case PARSER_BPF_NR_pipe2:
		st->fd_open++;
		break;
	case PARSER_BPF_NR_socket:
		st->fd_open++;
		st->socket_open++;
		break;
	case PARSER_BPF_NR_accept:
	case PARSER_BPF_NR_accept4:
		st->fd_open++;
		st->socket_accept++;
		break;
	default:
		break;
	}
}

struct parser_bpf_syscall_hist_key {
	__u32 pid;
	__u32 syscall_id;
};

struct parser_bpf_net_key {
	__u32 pid;
	__u16 dport;
	__u16 _pad;
};

struct parser_bpf_net_stats {
	__u64 connects;
	__u64 connect_ns_sum;
	__u64 connect_samples;
	__u64 retrans;
	__u64 rst;
	__u64 sendto_calls;
	__u64 sendto_bytes;
};

struct parser_bpf_syscall_peer {
	__u16 dport;
	__u16 _pad;
	__u32 sendto_len;
};

struct parser_bpf_slow_event {
	__u64 ts_ns;
	__u32 pid;
	__u32 syscall_id;
	__u64 duration_ns;
	__u8 role;
	__u8 kind;
	__u16 context_slot;
	__u32 marker_id;
};

struct parser_bpf_config {
	__u32 sample_rate;
	__u32 slow_syscall_ns;
	__u32 enabled;
	__u32 _pad;
};

struct parser_bpf_marker_ts_key {
	__u64 pid_tgid;
	__u32 marker_id;
	__u32 _pad;
};

struct parser_bpf_marker_hist_key {
	__u32 pid;
	__u32 marker_id;
	__u32 context_slot;
	__u32 _pad;
};

static __always_inline int parser_bpf_target_role(void *targets, __u32 pid)
{
	__u8 *role;

	if (!targets || !pid)
		return 0;
	role = bpf_map_lookup_elem(targets, &pid);
	if (!role || !*role)
		return 0;
	return (int)*role;
}

static __always_inline int parser_bpf_cgroup_role(void *cgroups, __u64 cgid)
{
	__u8 *role;

	if (!cgroups || !cgid)
		return 0;
	role = bpf_map_lookup_elem(cgroups, &cgid);
	if (!role || !*role)
		return 0;
	return (int)*role;
}

// Resolve role from cgroup map first, then PID map (docker/native targets).
static __always_inline int parser_bpf_resolve_role(void *targets, void *cgroups, __u32 pid)
{
	__u64 cgid;
	__u8 role;

	cgid = bpf_get_current_cgroup_id();
	if (cgid) {
		role = parser_bpf_cgroup_role(cgroups, cgid);
		if (role)
			return role;
	}
	return parser_bpf_target_role(targets, pid);
}

static __always_inline int parser_bpf_should_sample(const struct parser_bpf_config *cfg)
{
	__u32 rate;

	if (!cfg || !cfg->enabled)
		return 0;
	rate = cfg->sample_rate;
	if (!rate)
		rate = 1;
	if (rate == 1)
		return 1;
	return (bpf_get_prandom_u32() % rate) == 0;
}

static __always_inline void parser_bpf_hist_record(struct parser_bpf_hist *hist, __u64 delta_ns)
{
	__u32 bucket;
	__u64 v;

	if (!hist || !delta_ns)
		return;
	v = delta_ns;
	bucket = 0;
	while (bucket + 1 < PARSER_BPF_HIST_BUCKETS && v > 1) {
		v >>= 1;
		bucket++;
	}
	hist->buckets[bucket]++;
	hist->count++;
	hist->sum_ns += delta_ns;
	if (delta_ns > hist->max_ns)
		hist->max_ns = delta_ns;
}

#endif
