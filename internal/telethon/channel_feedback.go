package telethon

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
)

// RecordInviteICPRejected disables an invite channel in triage cache and crawler.db.
// No-op when the channel is already marked drop or the source is not telegram:invite:*.
func RecordInviteICPRejected(cfg ChannelTriageConfig, source string) error {
	id := inviteSourceToChannelID(source)
	if id == "" {
		return nil
	}
	if cfg.ChannelsPath == "" {
		cfg.ChannelsPath = defaultChannelsPath
	}
	if cfg.CachePath == "" {
		cfg.CachePath = defaultTriageCachePath
	}
	if cfg.CursorDBPath == "" {
		cfg.CursorDBPath = defaultCursorDBPath
	}

	cache := readTriageCache(cfg.CachePath)
	if action, ok := cache.Decisions[id]; ok && action == "drop" {
		return nil
	}
	cache.Decisions[id] = "drop"
	if err := writeTriageCache(cfg.CachePath, cache); err != nil {
		return err
	}

	key := channelIDToCursorKey(id)
	if key != "" {
		if err := setChannelsEnabled(cfg.CursorDBPath, []string{key}, false); err != nil {
			return err
		}
	}
	if err := removeChannelFromRegistry(cfg.ChannelsPath, id); err != nil {
		return err
	}
	if err := ExportChannelsJSON(cfg.CursorDBPath, cfg.ChannelsPath); err != nil {
		slog.Warn("invite icp reject registry export failed", "error", err, "channel_id", id)
	}
	slog.Info("invite channel disabled after icp_rejected", "channel_id", id)
	return nil
}

func inviteSourceToChannelID(source string) string {
	lower := strings.ToLower(strings.TrimSpace(source))
	if !strings.HasPrefix(lower, "telegram:invite:") {
		return ""
	}
	return "invite:" + strings.TrimPrefix(lower, "telegram:invite:")
}

func removeChannelFromRegistry(path, id string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var file channelFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return err
	}
	kept := make([]channelEntry, 0, len(file.Channels))
	for _, ch := range file.Channels {
		if channelID(ch) == id {
			continue
		}
		kept = append(kept, ch)
	}
	if len(kept) == len(file.Channels) {
		return nil
	}
	file.Channels = kept
	return writeChannelFile(path, file)
}
