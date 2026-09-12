package telegrambot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/crm/store"
)

type Config struct {
	Token          string
	AllowedChatIDs []int64
	ExportJSONPath string
	PollTimeoutSec int
}

type Bot struct {
	client *Client
	cfg    Config
	store  *store.LeadStore
}

func New(cfg Config, leadStore *store.LeadStore) (*Bot, error) {
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, fmt.Errorf("telegram bot token empty")
	}
	if len(cfg.AllowedChatIDs) == 0 {
		return nil, fmt.Errorf("CRM_TELEGRAM_ALLOWED_CHAT_IDS empty")
	}
	if leadStore == nil {
		return nil, fmt.Errorf("lead store nil")
	}
	timeout := cfg.PollTimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	cfg.PollTimeoutSec = timeout
	return &Bot{
		client: NewClient(token),
		cfg:    cfg,
		store:  leadStore,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("telegram bot nil")
	}
	slog.Info("crm telegram export bot started", "allowed_chats", len(b.cfg.AllowedChatIDs))

	var offset int64
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		updates, err := b.client.GetUpdates(ctx, offset, b.cfg.PollTimeoutSec)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Warn("telegram getUpdates failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		for _, upd := range updates {
			if upd.UpdateID >= offset {
				offset = upd.UpdateID + 1
			}
			if upd.Message == nil {
				continue
			}
			b.handleMessage(ctx, *upd.Message)
		}
	}
}

func (b *Bot) allowed(chatID int64) bool {
	for _, id := range b.cfg.AllowedChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

func (b *Bot) handleMessage(ctx context.Context, msg Message) {
	if !b.allowed(msg.Chat.ID) {
		slog.Warn("telegram export denied", "chat_id", msg.Chat.ID)
		_ = b.client.SendMessage(ctx, msg.Chat.ID, "access denied")
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}
	cmd, args := parseCommand(text)
	switch cmd {
	case "start", "help":
		_ = b.client.SendMessage(ctx, msg.Chat.ID, helpText())
	case "stats":
		b.handleStats(ctx, msg.Chat.ID)
	case "export":
		b.handleExport(ctx, msg.Chat.ID, args)
	case "jsonl":
		b.handleJSONL(ctx, msg.Chat.ID)
	default:
		_ = b.client.SendMessage(ctx, msg.Chat.ID, "unknown command; try /help")
	}
}

func (b *Bot) handleStats(ctx context.Context, chatID int64) {
	stats, err := b.store.DBStats(ctx)
	if err != nil {
		_ = b.client.SendMessage(ctx, chatID, "stats failed: "+err.Error())
		return
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("total=%d", stats.TotalLeads))
	for _, row := range stats.ByStatus {
		lines = append(lines, fmt.Sprintf("%s=%d", row.Status, row.Count))
	}
	_ = b.client.SendMessage(ctx, chatID, strings.Join(lines, "\n"))
}

func (b *Bot) handleExport(ctx context.Context, chatID int64, args []string) {
	status := ""
	limit := int64(0)
	if len(args) > 0 {
		status = args[0]
	}
	if len(args) > 1 {
		if n, err := strconv.ParseInt(args[1], 10, 64); err == nil && n > 0 {
			limit = n
		}
	}

	result, err := b.store.BuildNDJSON(ctx, store.ExportFilter{
		Status: status,
		Limit:  limit,
	})
	if err != nil {
		_ = b.client.SendMessage(ctx, chatID, "export failed: "+err.Error())
		return
	}
	defer func() { _ = os.Remove(result.Path) }()

	caption := fmt.Sprintf("mongo export rows=%d status=%q", result.Rows, statusOrAll(status))
	if err := b.client.SendDocument(ctx, chatID, result.Path, caption); err != nil {
		_ = b.client.SendMessage(ctx, chatID, "send document failed: "+err.Error())
		return
	}
}

func (b *Bot) handleJSONL(ctx context.Context, chatID int64) {
	path := strings.TrimSpace(b.cfg.ExportJSONPath)
	if path == "" {
		_ = b.client.SendMessage(ctx, chatID, "PARSER_EXPORT_JSON path not configured")
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		_ = b.client.SendMessage(ctx, chatID, "jsonl file missing: "+path)
		return
	}
	caption := fmt.Sprintf("parser jsonl bytes=%d", info.Size())
	if err := b.client.SendDocument(ctx, chatID, path, caption); err != nil {
		_ = b.client.SendMessage(ctx, chatID, "send document failed: "+err.Error())
	}
}

func parseCommand(text string) (string, []string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", nil
	}
	text = strings.TrimPrefix(text, "/")
	if i := strings.Index(text, "@"); i >= 0 {
		text = text[:i]
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}
	return strings.ToLower(parts[0]), parts[1:]
}

func statusOrAll(status string) string {
	if strings.TrimSpace(status) == "" {
		return "all"
	}
	return status
}

func helpText() string {
	return strings.TrimSpace(`BidShard CRM export bot

/stats - lead counts by status
/export - NDJSON from Mongo (top by score)
/export new - filter by status
/export new 100 - status + row limit
/jsonl - parser JSONL file (PARSER_EXPORT_JSON)

Auth: only CRM_TELEGRAM_ALLOWED_CHAT_IDS.`)
}

// Run starts the bot until ctx is cancelled.
func Run(ctx context.Context, cfg Config, leadStore *store.LeadStore, wg *sync.WaitGroup) {
	if strings.TrimSpace(cfg.Token) == "" {
		return
	}
	bot, err := New(cfg, leadStore)
	if err != nil {
		slog.Warn("crm telegram bot disabled", "error", err)
		return
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bot.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("crm telegram bot stopped", "error", err)
		}
	}()
}
