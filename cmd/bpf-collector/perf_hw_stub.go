//go:build !linux

package main

func (r *probeRun) collectHardwarePerf(pidStats []dumpedPIDStats) []map[string]any {
	return nil
}
