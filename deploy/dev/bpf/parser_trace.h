// Tracepoint argument layouts (match kernel trace event structs for BPF reads).
#ifndef PARSER_PROBE_TRACE_H
#define PARSER_PROBE_TRACE_H

typedef int pid_t;

struct trace_event_raw_sys_enter {
	__u64 unused;
	long id;
	unsigned long args[6];
};

struct trace_event_raw_sys_exit {
	__u64 unused;
	long id;
	long ret;
};

struct trace_event_raw_sched_wakeup {
	__u64 unused;
	char comm[16];
	pid_t pid;
	int prio;
	int success;
	int target_cpu;
};

struct trace_event_raw_sched_switch {
	__u64 unused;
	char prev_comm[16];
	pid_t prev_pid;
	int prev_prio;
	long prev_state;
	char next_comm[16];
	pid_t next_pid;
	int next_prio;
};

struct trace_event_raw_page_fault_user {
	__u64 unused;
	unsigned long address;
	unsigned long ip;
	unsigned long error_code;
};

struct trace_event_raw_sched_process_fork {
	__u64 unused;
	char parent_comm[16];
	pid_t parent_pid;
	char child_comm[16];
	pid_t child_pid;
};

struct trace_event_raw_sched_process_exit {
	__u64 unused;
	char comm[16];
	pid_t pid;
	int prio;
};

#endif
