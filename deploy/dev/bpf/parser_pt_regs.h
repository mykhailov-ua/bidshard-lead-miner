// Minimal x86_64 pt_regs for uprobe slot reads (BPF target arch).
#ifndef PARSER_PROBE_PT_REGS_H
#define PARSER_PROBE_PT_REGS_H

struct pt_regs {
	__u64 r15;
	__u64 r14;
	__u64 r13;
	__u64 r12;
	__u64 bp;
	__u64 bx;
	__u64 r11;
	__u64 r10;
	__u64 r9;
	__u64 r8;
	__u64 ax;
	__u64 cx;
	__u64 dx;
	__u64 si;
	__u64 di;
	__u64 orig_ax;
	__u64 ip;
	__u64 cs;
	__u64 flags;
	__u64 sp;
	__u64 ss;
};

#endif
