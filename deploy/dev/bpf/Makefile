# eBPF parser_probe object (dev perf only).
#
# Usage:
#   make -C deploy/dev/bpf
#   make -C deploy/dev/bpf disasm   # parser_probe.disasm.txt (gitignored)
#
# Requires: clang, libbpf headers. Or: bash scripts/perf/bpf_build.sh (docker fallback).

CLANG ?= clang
ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
BPF_CFLAGS := -O2 -g -Wall -Werror -target bpf -D__TARGET_ARCH_$(ARCH)
BPF_INCLUDES := -I. -I/usr/include -I/usr/include/$(shell uname -m)-linux-gnu -I/usr/local/include

OUT_DIR := .
OBJ := $(OUT_DIR)/parser_probe.o

.PHONY: all clean disasm

all: $(OBJ)

$(OBJ): parser_probe.bpf.c parser_probe.h parser_pt_regs.h parser_trace.h
	$(CLANG) $(BPF_CFLAGS) $(BPF_INCLUDES) -c parser_probe.bpf.c -o $@

clean:
	rm -f $(OBJ)

disasm: $(OBJ)
	llvm-objdump -d -S $(OBJ) > parser_probe.disasm.txt
