package serp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/metrics"
)

const defaultTGChannelsPath = "data/runtime/discovered_telegram_channels.json"

type tgChannelEntry struct {
	Username   string `json:"username,omitempty"`
	InviteHash string `json:"invite_hash,omitempty"`
	Source     string `json:"source"`
	Query      string `json:"query,omitempty"`
	At         string `json:"at"`
}

type tgChannelFile struct {
	Channels []tgChannelEntry `json:"channels"`
}

func appendTelegramChannelDiscoveries(path string, dork string, results []SERPResult) error {
	if path == "" {
		path = defaultTGChannelsPath
	}
	var handles []string
	var invites []string
	for _, res := range results {
		blob := res.Title + " " + res.Snippet + " " + res.URL
		handles = append(handles, extract.TelegramHandles(blob)...)
		invites = append(invites, extract.TelegramInviteHashes(blob)...)
	}
	if len(handles) == 0 && len(invites) == 0 {
		return nil
	}

	existing := readTGChannelFile(path)
	seenUser := make(map[string]struct{}, len(existing.Channels))
	seenInvite := make(map[string]struct{}, len(existing.Channels))
	for _, e := range existing.Channels {
		if e.Username != "" {
			seenUser[strings.ToLower(e.Username)] = struct{}{}
		}
		if e.InviteHash != "" {
			seenInvite[e.InviteHash] = struct{}{}
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	added := 0
	for _, u := range handles {
		u = strings.ToLower(strings.TrimPrefix(u, "@"))
		if _, ok := seenUser[u]; ok {
			continue
		}
		seenUser[u] = struct{}{}
		existing.Channels = append(existing.Channels, tgChannelEntry{
			Username: u,
			Source:   "serp",
			Query:    dork,
			At:       now,
		})
		added++
	}
	for _, h := range invites {
		if _, ok := seenInvite[h]; ok {
			continue
		}
		seenInvite[h] = struct{}{}
		existing.Channels = append(existing.Channels, tgChannelEntry{
			InviteHash: h,
			Source:     "serp",
			Query:      dork,
			At:         now,
		})
		added++
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("telegram", added)
	}

	return writeTGChannelFile(path, existing)
}

func readTGChannelFile(path string) tgChannelFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return tgChannelFile{}
	}
	var f tgChannelFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return tgChannelFile{}
	}
	return f
}

func writeTGChannelFile(path string, f tgChannelFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
