package discord

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultChannelsPath = "data/runtime/discovered_discord_channels.json"

type ChannelEntry struct {
	ChannelID   string `json:"channel_id"`
	GuildID     string `json:"guild_id"`
	GuildName   string `json:"guild_name,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
	InviteCode  string `json:"invite_code,omitempty"`
	Source      string `json:"source,omitempty"`
	Enabled     bool   `json:"enabled"`
	At          string `json:"at"`
}

type ChannelFile struct {
	Channels []ChannelEntry `json:"channels"`
}

func LoadChannelRegistry(path string) (ChannelFile, error) {
	if path == "" {
		path = DefaultChannelsPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ChannelFile{}, nil
		}
		return ChannelFile{}, err
	}
	var f ChannelFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return ChannelFile{}, err
	}
	return f, nil
}

func SaveChannelRegistry(path string, f ChannelFile) error {
	if path == "" {
		path = DefaultChannelsPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func MergeChannelEntries(path string, entries []ChannelEntry) (added int, err error) {
	if len(entries) == 0 {
		return 0, nil
	}
	f, err := LoadChannelRegistry(path)
	if err != nil {
		return 0, err
	}
	seen := map[string]struct{}{}
	for _, e := range f.Channels {
		seen[e.ChannelID] = struct{}{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, e := range entries {
		id := strings.TrimSpace(e.ChannelID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if e.At == "" {
			e.At = now
		}
		if !e.Enabled {
			e.Enabled = true
		}
		f.Channels = append(f.Channels, e)
		added++
	}
	if added == 0 {
		return 0, nil
	}
	return added, SaveChannelRegistry(path, f)
}

func EnabledChannelIDs(path string) ([]string, error) {
	f, err := LoadChannelRegistry(path)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []string
	for _, e := range f.Channels {
		if !e.Enabled {
			continue
		}
		id := strings.TrimSpace(e.ChannelID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func ResolveChannelIDs(envIDs []string, registryPath string, auto bool) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, id := range envIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if auto {
		fromReg, err := EnabledChannelIDs(registryPath)
		if err != nil {
			return out, err
		}
		for _, id := range fromReg {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out, nil
}
