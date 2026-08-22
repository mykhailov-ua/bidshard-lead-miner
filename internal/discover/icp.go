package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ICPFile default path for shared discovery queries (Telegram search + SERP dorks).
const ICPFile = "config/discover.icp.json"

type ICPConfig struct {
	TelegramSearch []string `json:"telegram_search"`
	SerpDorks      []string `json:"serp_dorks"`
}

func LoadICP(path string) (ICPConfig, error) {
	if path == "" {
		path = ICPFile
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ICPConfig{}, err
	}
	var cfg ICPConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ICPConfig{}, err
	}
	return cfg, nil
}

// SaveICP writes discover ICP config to path (caller should backup first).
func SaveICP(path string, cfg ICPConfig) error {
	if path == "" {
		path = ICPFile
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// ResolveICPPath finds discover.icp.json from cwd or repo root.
func ResolveICPPath(start string) string {
	if start == "" {
		start, _ = os.Getwd()
	}
	dir := start
	for {
		p := filepath.Join(dir, ICPFile)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ICPFile
		}
		dir = parent
	}
}
