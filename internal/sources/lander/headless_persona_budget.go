package lander

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DefaultHeadlessPersonaBudgetPath = "data/runtime/headless_proxy_budget.json"

type personaBudgetFile struct {
	Day   string         `json:"day"`
	Count map[string]int `json:"count_by_persona"`
}

// PersonaDailyBudget caps headless enqueue/fetch per proxy index per UTC day.
type PersonaDailyBudget struct {
	mu     sync.Mutex
	maxDay int
	path   string
	day    string
	counts map[string]int
}

var globalPersonaBudget *PersonaDailyBudget

// InitHeadlessPersonaBudget configures daily caps (maxPerDay 0 disables).
// ResetHeadlessPersonaBudgetForTest clears the global cap (tests only).
func ResetHeadlessPersonaBudgetForTest() {
	globalPersonaBudget = nil
}

func InitHeadlessPersonaBudget(maxPerDay int, path string) {
	if maxPerDay <= 0 {
		globalPersonaBudget = nil
		return
	}
	if path == "" {
		path = DefaultHeadlessPersonaBudgetPath
	}
	globalPersonaBudget = &PersonaDailyBudget{
		maxDay: maxPerDay,
		path:   path,
		counts: map[string]int{},
	}
	globalPersonaBudget.load()
}

func personaKey(proxyIndex int) string {
	if proxyIndex < 0 {
		return "direct"
	}
	return fmt.Sprintf("proxy_%d", proxyIndex)
}

func (b *PersonaDailyBudget) utcDay() string {
	return time.Now().UTC().Format("2006-01-02")
}

func (b *PersonaDailyBudget) load() {
	b.mu.Lock()
	defer b.mu.Unlock()
	today := b.utcDay()
	raw, err := os.ReadFile(b.path)
	if err != nil {
		b.day = today
		b.counts = map[string]int{}
		return
	}
	var file personaBudgetFile
	if json.Unmarshal(raw, &file) != nil || file.Day != today {
		b.day = today
		b.counts = map[string]int{}
		return
	}
	if file.Count == nil {
		file.Count = map[string]int{}
	}
	b.day = today
	b.counts = file.Count
}

func (b *PersonaDailyBudget) save() error {
	if err := os.MkdirAll(filepath.Dir(b.path), 0o755); err != nil {
		return err
	}
	file := personaBudgetFile{Day: b.day, Count: b.counts}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.path, raw, 0o644)
}

// Allow reports whether another headless URL may run for this proxy persona today.
func AllowHeadlessPersona(proxyIndex int) bool {
	b := globalPersonaBudget
	if b == nil {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	today := b.utcDay()
	if b.day != today {
		b.day = today
		b.counts = map[string]int{}
	}
	key := personaKey(proxyIndex)
	return b.counts[key] < b.maxDay
}

// RecordHeadlessPersona increments the daily counter after a successful enqueue or inline fetch.
func RecordHeadlessPersona(proxyIndex int) {
	b := globalPersonaBudget
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	today := b.utcDay()
	if b.day != today {
		b.day = today
		b.counts = map[string]int{}
	}
	key := personaKey(proxyIndex)
	b.counts[key]++
	_ = b.save()
}
