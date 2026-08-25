package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultAutoApplyStatePath = "data/runtime/discover_auto_apply.json"

// DefaultAutoApplyMaxPerWeek caps new dorks applied without operator review.
const DefaultAutoApplyMaxPerWeek = 30

// DiscoverDorkDenylist blocks junk queries from auto-apply (case-insensitive substring).
var DiscoverDorkDenylist = []string{
	"linkedin.com",
	"facebook.com",
	"instagram.com",
	"indeed.com",
	"glassdoor",
	"monster.com",
	"ziprecruiter",
}

// AutoApplyState tracks weekly auto-apply quota consumption.
type AutoApplyState struct {
	WeekStart string `json:"week_start"`
	Applied   int    `json:"applied"`
}

// BlockedDiscoverQuery reports whether a discover query must stay in manual review.
func BlockedDiscoverQuery(q string) (bool, string) {
	q = strings.TrimSpace(q)
	if len(q) < 4 {
		return true, "too_short"
	}
	lower := strings.ToLower(q)
	for _, pat := range DiscoverDorkDenylist {
		if strings.Contains(lower, pat) {
			return true, pat
		}
	}
	return false, ""
}

// FilterDiscoverAdditions splits proposed queries into keep vs blocked-by-denylist.
func FilterDiscoverAdditions(telegram, serp []string) (keepTelegram, keepSerp []string, blocked map[string]string) {
	blocked = make(map[string]string)
	for _, q := range telegram {
		if ok, reason := BlockedDiscoverQuery(q); ok {
			blocked[q] = reason
			continue
		}
		keepTelegram = append(keepTelegram, q)
	}
	for _, q := range serp {
		if ok, reason := BlockedDiscoverQuery(q); ok {
			blocked[q] = reason
			continue
		}
		keepSerp = append(keepSerp, q)
	}
	return keepTelegram, keepSerp, blocked
}

// NewDiscoverAdditions returns items from telegram/serp not already in base (case-fold dedup).
func NewDiscoverAdditions(base ICPConfig, telegram, serp []string) (newTelegram, newSerp []string) {
	seenTG := make(map[string]struct{}, len(base.TelegramSearch))
	for _, s := range base.TelegramSearch {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		seenTG[strings.ToLower(s)] = struct{}{}
	}
	seenSerp := make(map[string]struct{}, len(base.SerpDorks))
	for _, s := range base.SerpDorks {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		seenSerp[strings.ToLower(s)] = struct{}{}
	}
	for _, s := range telegram {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seenTG[key]; ok {
			continue
		}
		seenTG[key] = struct{}{}
		newTelegram = append(newTelegram, s)
	}
	for _, s := range serp {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seenSerp[key]; ok {
			continue
		}
		seenSerp[key] = struct{}{}
		newSerp = append(newSerp, s)
	}
	return newTelegram, newSerp
}

// TrimDiscoverQuota limits how many new queries may auto-apply this call.
func TrimDiscoverQuota(telegram, serp []string, quota int) (keepTelegram, keepSerp, deferred []string) {
	if quota <= 0 {
		deferred = append(deferred, telegram...)
		deferred = append(deferred, serp...)
		return nil, nil, deferred
	}
	for _, q := range telegram {
		if quota == 0 {
			deferred = append(deferred, q)
			continue
		}
		keepTelegram = append(keepTelegram, q)
		quota--
	}
	for _, q := range serp {
		if quota == 0 {
			deferred = append(deferred, q)
			continue
		}
		keepSerp = append(keepSerp, q)
		quota--
	}
	return keepTelegram, keepSerp, deferred
}

func weekStartUTC(now time.Time) time.Time {
	now = now.UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
}

// LoadAutoApplyState reads quota state; missing file starts a fresh week.
func LoadAutoApplyState(path string, now time.Time) AutoApplyState {
	if path == "" {
		path = DefaultAutoApplyStatePath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return AutoApplyState{WeekStart: weekStartUTC(now).Format(time.RFC3339)}
	}
	var st AutoApplyState
	if err := json.Unmarshal(raw, &st); err != nil {
		return AutoApplyState{WeekStart: weekStartUTC(now).Format(time.RFC3339)}
	}
	start, err := time.Parse(time.RFC3339, st.WeekStart)
	if err != nil || !start.Equal(weekStartUTC(now)) {
		return AutoApplyState{WeekStart: weekStartUTC(now).Format(time.RFC3339)}
	}
	return st
}

// SaveAutoApplyState persists quota state.
func SaveAutoApplyState(path string, st AutoApplyState) error {
	if path == "" {
		path = DefaultAutoApplyStatePath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// RemainingWeeklyQuota returns how many new queries may still auto-apply this week.
func RemainingWeeklyQuota(st AutoApplyState, now time.Time, maxPerWeek int) int {
	st = normalizeAutoApplyWeek(st, now)
	if maxPerWeek <= 0 {
		return 0
	}
	rem := maxPerWeek - st.Applied
	if rem < 0 {
		return 0
	}
	return rem
}

func normalizeAutoApplyWeek(st AutoApplyState, now time.Time) AutoApplyState {
	start, err := time.Parse(time.RFC3339, st.WeekStart)
	if err != nil || !start.Equal(weekStartUTC(now)) {
		return AutoApplyState{WeekStart: weekStartUTC(now).Format(time.RFC3339)}
	}
	return st
}
