package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// PendingICPDiff is a manual-approve patch for discover.icp.json.
type PendingICPDiff struct {
	AddTelegramSearch []string `json:"add_telegram_search"`
	AddSerpDorks      []string `json:"add_serp_dorks"`
	Summary           string   `json:"summary"`
	ReportID          string   `json:"report_id,omitempty"`
	GeneratedAt       string   `json:"generated_at,omitempty"`
	Status            string   `json:"status"`
}

// WritePendingJSON writes <name>_<reportID>.json under dir.
func WritePendingJSON(dir, name, reportID string, v any) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+"_"+reportID+".json")
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// WritePending writes discover_icp_pending_<reportID>.json under dir.
func WritePending(dir, reportID string, diff PendingICPDiff) (string, error) {
	diff.Status = "pending"
	return WritePendingJSON(dir, "discover_icp_pending", reportID, diff)
}

// MergeSuggestions returns a new ICPConfig with pending additions deduped against current.
func MergeSuggestions(current ICPConfig, addTelegram, addSerp []string) ICPConfig {
	out := current
	out.TelegramSearch = UniqueFolded(append([]string(nil), current.TelegramSearch...), addTelegram)
	out.SerpDorks = UniqueFolded(append([]string(nil), current.SerpDorks...), addSerp)
	return out
}

// UniqueFolded appends extra strings to base, deduping by trimmed lowercase key.
func UniqueFolded(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, s := range base {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	for _, s := range extra {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}
