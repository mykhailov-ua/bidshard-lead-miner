package suggestions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/gemini"
)

// PendingKind classifies a pending suggestion file.
type PendingKind string

const (
	KindKeywords PendingKind = "keywords"
	KindDiscover PendingKind = "discover"
)

// PendingFile is one manual-approve JSON under data/suggestions/.
type PendingFile struct {
	Path   string
	Kind   PendingKind
	Status string
}

// ListPending returns suggestion files with status=pending (or empty status).
func ListPending(dir string) ([]PendingFile, error) {
	if dir == "" {
		dir = "data/suggestions"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]PendingFile, 0, len(entries))
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		kind, status, err := peekPending(path)
		if err != nil {
			continue
		}
		if status != "" && status != "pending" {
			continue
		}
		out = append(out, PendingFile{Path: path, Kind: kind, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// PeekPending returns file kind and status without applying.
func PeekPending(path string) (PendingKind, string, error) {
	return peekPending(path)
}

func peekPending(path string) (PendingKind, string, error) {
	name := filepath.Base(path)
	switch {
	case strings.HasPrefix(name, "keywords_pending_"), strings.HasPrefix(name, "pain_vocab_pending_"):
		var diff gemini.KeywordDiff
		if err := readJSON(path, &diff); err != nil {
			return "", "", err
		}
		return KindKeywords, diff.Status, nil
	case strings.HasPrefix(name, "discover_icp_pending_"):
		var diff discover.PendingICPDiff
		if err := readJSON(path, &diff); err != nil {
			return "", "", err
		}
		return KindDiscover, diff.Status, nil
	default:
		return "", "", fmt.Errorf("unknown pending file: %s", name)
	}
}

// PreviewKeywords prints a human-readable diff summary.
func PreviewKeywords(pendingPath, keywordsPath string) (string, error) {
	diff, base, err := loadKeywordPending(pendingPath, keywordsPath)
	if err != nil {
		return "", err
	}
	merged, addedKW, addedHR := mergeKeywords(base, diff)
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "keywords: %s\n", keywordsPath)
	_, _ = fmt.Fprintf(&b, "pending: %s\n", pendingPath)
	_, _ = fmt.Fprintf(&b, "add_keywords: %d new (of %d proposed)\n", addedKW, len(diff.AddKeywords))
	_, _ = fmt.Fprintf(&b, "add_hard_reject: %d new (of %d proposed)\n", addedHR, len(diff.AddHardReject))
	_, _ = fmt.Fprintf(&b, "total keywords after: %d hard_reject: %d\n", len(merged.Keywords), len(merged.HardReject))
	if diff.Summary != "" {
		_, _ = fmt.Fprintf(&b, "summary: %s\n", diff.Summary)
	}
	return b.String(), nil
}

// ApplyKeywords merges pending diff into keywords.json (backup first unless dry-run).
func ApplyKeywords(pendingPath, keywordsPath string, dryRun bool) (string, error) {
	diff, base, err := loadKeywordPending(pendingPath, keywordsPath)
	if err != nil {
		return "", err
	}
	merged, addedKW, addedHR := mergeKeywords(base, diff)
	summary := fmt.Sprintf("add_keywords=%d add_hard_reject=%d", addedKW, addedHR)
	if dryRun {
		return summary + " (dry-run)", nil
	}
	if err := backupFile(keywordsPath); err != nil {
		return "", err
	}
	if err := writeJSON(keywordsPath, merged); err != nil {
		return "", err
	}
	diff.Status = "applied"
	if err := writeJSON(pendingPath, diff); err != nil {
		return "", err
	}
	return summary, nil
}

// PreviewDiscover summarizes discover ICP pending diff.
func PreviewDiscover(pendingPath, icpPath string) (string, error) {
	diff, base, err := loadDiscoverPending(pendingPath, icpPath)
	if err != nil {
		return "", err
	}
	merged := discover.MergeSuggestions(base, diff.AddTelegramSearch, diff.AddSerpDorks)
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "icp: %s\n", icpPath)
	_, _ = fmt.Fprintf(&b, "pending: %s\n", pendingPath)
	_, _ = fmt.Fprintf(&b, "add_telegram_search: %d -> %d total\n", len(diff.AddTelegramSearch), len(merged.TelegramSearch))
	_, _ = fmt.Fprintf(&b, "add_serp_dorks: %d -> %d total\n", len(diff.AddSerpDorks), len(merged.SerpDorks))
	if diff.Summary != "" {
		_, _ = fmt.Fprintf(&b, "summary: %s\n", diff.Summary)
	}
	return b.String(), nil
}

// ApplyDiscover merges pending diff into discover.icp.json.
func ApplyDiscover(pendingPath, icpPath string, dryRun bool) (string, error) {
	diff, base, err := loadDiscoverPending(pendingPath, icpPath)
	if err != nil {
		return "", err
	}
	merged := discover.MergeSuggestions(base, diff.AddTelegramSearch, diff.AddSerpDorks)
	summary := fmt.Sprintf("telegram_search=%d serp_dorks=%d", len(merged.TelegramSearch), len(merged.SerpDorks))
	if dryRun {
		return summary + " (dry-run)", nil
	}
	if err := backupFile(icpPath); err != nil {
		return "", err
	}
	if err := discover.SaveICP(icpPath, merged); err != nil {
		return "", err
	}
	diff.Status = "applied"
	if err := writeJSON(pendingPath, diff); err != nil {
		return "", err
	}
	return summary, nil
}

// RejectPending marks a pending file rejected without merging.
func RejectPending(path string) error {
	name := filepath.Base(path)
	switch {
	case strings.HasPrefix(name, "keywords_pending_"), strings.HasPrefix(name, "pain_vocab_pending_"):
		var diff gemini.KeywordDiff
		if err := readJSON(path, &diff); err != nil {
			return err
		}
		diff.Status = "rejected"
		return writeJSON(path, diff)
	case strings.HasPrefix(name, "discover_icp_pending_"):
		var diff discover.PendingICPDiff
		if err := readJSON(path, &diff); err != nil {
			return err
		}
		diff.Status = "rejected"
		return writeJSON(path, diff)
	default:
		return fmt.Errorf("unknown pending file: %s", name)
	}
}

type keywordsFile struct {
	Priority   priorityBlock         `json:"priority"`
	Keywords   []gemini.KeywordEntry `json:"keywords"`
	Titles     []gemini.KeywordEntry `json:"titles"`
	Negative   []gemini.KeywordEntry `json:"negative"`
	HardReject []gemini.KeywordEntry `json:"hard_reject"`
}

type priorityBlock struct {
	HighMin   int `json:"high_min"`
	MediumMin int `json:"medium_min"`
}

func loadKeywordPending(pendingPath, keywordsPath string) (gemini.KeywordDiff, keywordsFile, error) {
	var diff gemini.KeywordDiff
	if err := readJSON(pendingPath, &diff); err != nil {
		return diff, keywordsFile{}, err
	}
	var base keywordsFile
	if err := readJSON(keywordsPath, &base); err != nil {
		return diff, keywordsFile{}, err
	}
	return diff, base, nil
}

func loadDiscoverPending(pendingPath, icpPath string) (discover.PendingICPDiff, discover.ICPConfig, error) {
	var diff discover.PendingICPDiff
	if err := readJSON(pendingPath, &diff); err != nil {
		return diff, discover.ICPConfig{}, err
	}
	base, err := discover.LoadICP(icpPath)
	if err != nil {
		return diff, discover.ICPConfig{}, err
	}
	return diff, base, nil
}

func mergeKeywords(base keywordsFile, diff gemini.KeywordDiff) (keywordsFile, int, int) {
	seen := map[string]struct{}{}
	for _, e := range base.Keywords {
		seen[strings.ToLower(strings.TrimSpace(e.ID))] = struct{}{}
	}
	for _, e := range base.HardReject {
		seen[strings.ToLower(strings.TrimSpace(e.ID))] = struct{}{}
	}
	addedKW := 0
	for _, e := range diff.AddKeywords {
		id := strings.ToLower(strings.TrimSpace(e.ID))
		if id == "" || e.Phrase == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		base.Keywords = append(base.Keywords, e)
		addedKW++
	}
	addedHR := 0
	for _, e := range diff.AddHardReject {
		id := strings.ToLower(strings.TrimSpace(e.ID))
		if id == "" || e.Phrase == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if e.Tag == "" {
			e.Tag = "hard-reject"
		}
		base.HardReject = append(base.HardReject, e)
		addedHR++
	}
	return base, addedKW, addedHR
}

func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func backupFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	backup := fmt.Sprintf("%s.bak.%s", path, time.Now().UTC().Format("20060102T150405Z"))
	return os.WriteFile(backup, raw, 0o644)
}
