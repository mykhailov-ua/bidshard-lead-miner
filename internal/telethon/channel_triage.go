package telethon

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/gemini"
)

const defaultChannelsPath = "data/runtime/discovered_telegram_channels.json"
const defaultTriageCachePath = "data/runtime/channel_triage_cache.json"

type channelEntry struct {
	Username   string `json:"username,omitempty"`
	InviteHash string `json:"invite_hash,omitempty"`
	Title      string `json:"title,omitempty"`
	Query      string `json:"query,omitempty"`
	Source     string `json:"source,omitempty"`
	Triage     string `json:"triage,omitempty"`
}

type channelFile struct {
	Channels []channelEntry `json:"channels"`
}

type triageCacheFile struct {
	Decisions map[string]string `json:"decisions"`
}

// ChannelTriageConfig controls post-discover Gemini channel filter.
type ChannelTriageConfig struct {
	ChannelsPath string
	CachePath    string
	BatchSize    int
}

// RunChannelTriage classifies unseen channels and drops noise from registry file.
// Cached keep/drop decisions skip repeat Gemini calls; only missing ids are batched.
func RunChannelTriage(ctx context.Context, cfg ChannelTriageConfig, client *gemini.Client) error {
	if client == nil {
		return nil
	}
	if cfg.ChannelsPath == "" {
		cfg.ChannelsPath = defaultChannelsPath
	}
	if cfg.CachePath == "" {
		cfg.CachePath = defaultTriageCachePath
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}

	raw, err := os.ReadFile(cfg.ChannelsPath)
	if err != nil {
		return err
	}
	var file channelFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return err
	}
	cache := readTriageCache(cfg.CachePath)

	var batch []gemini.ChannelTriageInput
	applyBatch := func() {
		if len(batch) == 0 {
			return
		}
		results, err := client.TriageChannels(ctx, batch)
		if err != nil {
			slog.Warn("channel triage gemini failed", "error", err)
			batch = batch[:0]
			return
		}
		for _, res := range results {
			if res.ID == "" {
				continue
			}
			cache.Decisions[res.ID] = res.Action
		}
		batch = batch[:0]
	}

	for _, ch := range file.Channels {
		if err := ctx.Err(); err != nil {
			return err
		}
		id := channelID(ch)
		if id == "" {
			continue
		}
		if action, ok := cache.Decisions[id]; ok && action == "drop" {
			continue
		}
		if ch.Triage == "drop" {
			continue
		}
		if _, ok := cache.Decisions[id]; ok {
			continue
		}
		title := strings.TrimSpace(ch.Title)
		if title == "" {
			title = ch.Username
		}
		batch = append(batch, gemini.ChannelTriageInput{
			ID:    id,
			Title: title,
			Query: ch.Query,
		})
		if len(batch) >= cfg.BatchSize {
			applyBatch()
		}
	}
	applyBatch()

	kept := make([]channelEntry, 0, len(file.Channels))
	dropped := 0
	for _, ch := range file.Channels {
		id := channelID(ch)
		if action, ok := cache.Decisions[id]; ok && action == "drop" {
			dropped++
			continue
		}
		kept = append(kept, ch)
	}
	file.Channels = kept
	if err := writeChannelFile(cfg.ChannelsPath, file); err != nil {
		return err
	}
	if err := writeTriageCache(cfg.CachePath, cache); err != nil {
		return err
	}
	if dropped > 0 {
		slog.Info("channel triage complete", "dropped", dropped, "kept", len(kept))
	}
	return nil
}

func channelID(ch channelEntry) string {
	if u := strings.TrimSpace(ch.Username); u != "" {
		return "user:" + strings.ToLower(strings.TrimPrefix(u, "@"))
	}
	if h := strings.TrimSpace(ch.InviteHash); h != "" {
		return "invite:" + h
	}
	return ""
}

func readTriageCache(path string) triageCacheFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return triageCacheFile{Decisions: map[string]string{}}
	}
	var cache triageCacheFile
	if json.Unmarshal(raw, &cache) != nil || cache.Decisions == nil {
		return triageCacheFile{Decisions: map[string]string{}}
	}
	return cache
}

func writeTriageCache(path string, cache triageCacheFile) error {
	if cache.Decisions == nil {
		cache.Decisions = map[string]string{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeChannelFile(path string, file channelFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// RunChannelTriageOnce is a test/CLI helper with a bounded runtime.
func RunChannelTriageOnce(ctx context.Context, cfg ChannelTriageConfig, client *gemini.Client) error {
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return RunChannelTriage(runCtx, cfg, client)
}
