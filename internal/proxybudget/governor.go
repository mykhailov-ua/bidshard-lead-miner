package proxybudget

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/metrics"
)

const DefaultStatePath = "data/runtime/proxy_budget.json"

// GovernorConfig wires daily proxy egress cap and persistence path.
type GovernorConfig struct {
	DailyCapMB int
	StatePath  string
}

// Governor tracks proxy egress bytes per UTC day and blocks new proxy crawls when over cap.
type Governor struct {
	mu       sync.Mutex
	capMB    int
	capBytes int64
	path     string
	state    stateFile
}

type stateFile struct {
	Day   string `json:"day"`
	Bytes int64  `json:"bytes"`
}

var (
	current   *Governor
	currentMu sync.RWMutex
)

// Configure installs the process-wide governor from config fields.
func Configure(dailyCapMB int, statePath string) *Governor {
	if statePath == "" {
		statePath = DefaultStatePath
	}
	g := &Governor{
		capMB: dailyCapMB,
		path:  statePath,
	}
	if dailyCapMB > 0 {
		g.capBytes = int64(dailyCapMB) * 1024 * 1024
		g.state = g.loadState(time.Now().UTC())
	}
	currentMu.Lock()
	current = g
	currentMu.Unlock()
	return g
}

// Current returns the configured governor (may be nil or disabled).
func Current() *Governor {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}

// Enabled reports whether a positive daily cap is configured.
func (g *Governor) Enabled() bool {
	return g != nil && g.capBytes > 0
}

// Allow reports whether another proxy response may be fetched today.
func (g *Governor) Allow() bool {
	if !g.Enabled() {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resetIfNewDayLocked(time.Now().UTC())
	return g.state.Bytes < g.capBytes
}

// Record adds response bytes to today's total and persists state.
func (g *Governor) Record(n int64) {
	if !g.Enabled() || n <= 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now().UTC()
	g.resetIfNewDayLocked(now)
	g.state.Bytes += n
	_ = g.persistLocked()
	if g.state.Bytes >= g.capBytes {
		metrics.SetProxyBudgetExceeded(true)
	}
	metrics.SetProxyBudgetDailyBytes(g.state.Bytes)
}

// UsedBytes returns today's recorded proxy egress bytes.
func (g *Governor) UsedBytes() int64 {
	if !g.Enabled() {
		return 0
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resetIfNewDayLocked(time.Now().UTC())
	return g.state.Bytes
}

// CapBytes returns configured daily cap in bytes (0 when disabled).
func (g *Governor) CapBytes() int64 {
	if g == nil {
		return 0
	}
	return g.capBytes
}

func (g *Governor) dayKey(now time.Time) string {
	return now.UTC().Format("2006-01-02")
}

func (g *Governor) resetIfNewDayLocked(now time.Time) {
	day := g.dayKey(now)
	if g.state.Day == day {
		return
	}
	g.state = stateFile{Day: day}
	metrics.SetProxyBudgetExceeded(false)
	metrics.SetProxyBudgetDailyBytes(0)
}

func (g *Governor) loadState(now time.Time) stateFile {
	st := stateFile{Day: g.dayKey(now)}
	raw, err := os.ReadFile(g.path)
	if err != nil {
		return st
	}
	var loaded stateFile
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return st
	}
	if loaded.Day != st.Day {
		return st
	}
	if loaded.Bytes < 0 {
		loaded.Bytes = 0
	}
	return loaded
}

func (g *Governor) persistLocked() error {
	if g.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(g.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(g.state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(g.path, raw, 0o644)
}
