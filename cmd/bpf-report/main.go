package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bidshard/parser/internal/perf"
)

func main() {
	outDir := flag.String("out", "", "session directory (contains bpf/ subdir)")
	gate := flag.Bool("gate", false, "run release gate on summary.json and exit")
	leakGate := flag.Bool("leak-gate", false, "run leak-focused gate (stricter FD/thread thresholds)")
	flag.Parse()
	dir := *outDir
	if dir == "" && flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	if dir == "" {
		fmt.Fprintln(os.Stderr, "usage: bpf-report [-gate|-leak-gate] <session_dir>")
		os.Exit(2)
	}
	summary := filepath.Join(dir, "bpf", "maps", "summary.json")
	if *gate || *leakGate {
		cfg := perf.DefaultBPFGateConfig()
		if *leakGate {
			cfg = perf.LeakGateConfig()
		}
		if err := perf.EvaluateBPFGate(summary, cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *leakGate {
			fmt.Println("bpf leak gate: ok")
		} else {
			fmt.Println("bpf release gate: ok")
		}
		return
	}
	path, err := perf.WriteBPFReport(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}
