package discord

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/httpclient"
)

// DiscoverChannels joins ICP guilds from invite registry and lists readable text channels.
func DiscoverChannels(ctx context.Context, cfg config.Config) error {
	if len(cfg.DiscordBotTokens) == 0 {
		slog.Warn("discord channel discover skipped", "reason", "missing bot tokens")
		return nil
	}
	client := httpclient.Shared(cfg.HTTPTimeout)

	invitePath := cfg.DiscordRegistryPath
	if invitePath == "" {
		invitePath = DefaultRegistryPath
	}
	channelPath := cfg.DiscordChannelsPath
	if channelPath == "" {
		channelPath = DefaultChannelsPath
	}

	invites, err := LoadRegistry(invitePath)
	if err != nil {
		return err
	}

	seenGuild := map[string]struct{}{}
	var batch []ChannelEntry
	joins := 0
	joinLimit := cfg.DiscordJoinDailyLimit
	if joinLimit <= 0 {
		joinLimit = 5
	}

	for _, inv := range invites.Invites {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		code := strings.ToLower(strings.TrimSpace(inv.InviteCode))
		if code == "" {
			continue
		}
		hint := strings.TrimSpace(inv.GuildHint)

		api := NewAPI(NewTokenPool(cfg.DiscordBotTokens), client, apiBase)
		preview, err := api.GetInvite(ctx, code)
		if err != nil {
			slog.Info("discord invite preview failed", "code", code, "error", err)
			continue
		}
		guildID := ""
		guildName := hint
		if preview.Guild != nil {
			guildID = strings.TrimSpace(preview.Guild.ID)
			if preview.Guild.Name != "" {
				guildName = preview.Guild.Name
			}
		}
		if guildID == "" {
			continue
		}
		if !GuildLooksICP(guildName, hint+" "+inv.Query) && !InviteEntryLooksICP(inv) {
			continue
		}
		if _, ok := seenGuild[guildID]; ok {
			continue
		}

		if cfg.DiscordJoinEnabled && joins < joinLimit {
			if _, err := api.AcceptInvite(ctx, code); err != nil {
				slog.Debug("discord invite join skipped", "code", code, "guild", guildName, "error", err)
			} else {
				joins++
				slog.Info("discord guild joined", "code", code, "guild", guildName)
			}
		}
		seenGuild[guildID] = struct{}{}
		batch = append(batch, channelsForGuild(ctx, api, guildID, guildName, code, "invite_registry")...)
	}

	// Sync channels from guilds each bot in the pool already belongs to.
	for _, token := range cfg.DiscordBotTokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		api := NewAPI(NewTokenPool([]string{token}), client, apiBase)
		guilds, err := api.ListMyGuilds(ctx)
		if err != nil {
			slog.Debug("discord list guilds failed", "error", err)
			continue
		}
		for _, g := range guilds {
			guildID := strings.TrimSpace(g.ID)
			if guildID == "" {
				continue
			}
			if _, ok := seenGuild[guildID]; ok {
				continue
			}
			if !GuildLooksICP(g.Name, "") {
				continue
			}
			seenGuild[guildID] = struct{}{}
			batch = append(batch, channelsForGuild(ctx, api, guildID, g.Name, "", "bot_guild")...)
		}
	}

	added, err := MergeChannelEntries(channelPath, batch)
	if err != nil {
		return err
	}
	slog.Info("discord channel discover finished",
		"invites_scanned", len(invites.Invites),
		"guilds", len(seenGuild),
		"joins", joins,
		"channels_added", added,
		"channels_total", len(batch),
	)
	return nil
}

func channelsForGuild(ctx context.Context, api *API, guildID, guildName, inviteCode, source string) []ChannelEntry {
	channels, err := api.ListGuildChannels(ctx, guildID)
	if err != nil {
		slog.Debug("discord list channels failed", "guild", guildName, "error", err)
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var out []ChannelEntry
	for _, ch := range channels {
		if !ChannelLooksReadable(ch.Name, ch.Type) {
			continue
		}
		out = append(out, ChannelEntry{
			ChannelID:   ch.ID,
			GuildID:     guildID,
			GuildName:   guildName,
			ChannelName: ch.Name,
			InviteCode:  inviteCode,
			Source:      source,
			Enabled:     true,
			At:          now,
		})
	}
	return out
}
