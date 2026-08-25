package github

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/bidshard/parser/internal/discover"
)

const DefaultRotateStatePath = "data/runtime/github_query_rotate.json"

type rotateState struct {
	Index int `json:"index"`
}

// RotateQueriesFromICP advances a round-robin over telegram_search + plain serp dorks.
func RotateQueriesFromICP(icpPath, statePath string, batch int) ([]string, error) {
	if batch <= 0 {
		batch = 2
	}
	resolved := icpPath
	if resolved == "" {
		resolved = discover.ResolveICPPath("")
	}
	icp, err := discover.LoadICP(resolved)
	if err != nil {
		return nil, err
	}
	var pool []string
	for _, q := range icp.TelegramSearch {
		q = strings.TrimSpace(q)
		if q != "" {
			pool = append(pool, q)
		}
	}
	for _, q := range icp.SerpDorks {
		q = strings.TrimSpace(strings.TrimPrefix(q, "site:t.me "))
		if q != "" && !strings.Contains(strings.ToLower(q), "site:") {
			pool = append(pool, q)
		}
	}
	if len(pool) == 0 {
		return nil, nil
	}
	st := loadRotateState(statePath)
	var out []string
	for i := 0; i < batch && i < len(pool); i++ {
		out = append(out, pool[(st.Index+i)%len(pool)])
	}
	st.Index = (st.Index + len(out)) % len(pool)
	if err := saveRotateState(statePath, st); err != nil {
		return out, err
	}
	return out, nil
}

func loadRotateState(path string) rotateState {
	if path == "" {
		path = DefaultRotateStatePath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return rotateState{}
	}
	var st rotateState
	if json.Unmarshal(raw, &st) != nil {
		return rotateState{}
	}
	return st
}

func saveRotateState(path string, st rotateState) error {
	if path == "" {
		path = DefaultRotateStatePath
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
