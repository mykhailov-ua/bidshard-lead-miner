package telegrambot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/crm/store"
)

type Config struct {
	Token                    string
	AllowedChatIDs           []int64
	ExportJSONPath           string
	TelegramPeopleCollection string
	PollTimeoutSec           int
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
			LogAPIError("crm telegram bot getUpdates failed", err)
			sleep := 2 * time.Second
			if sec := RetryAfterSec(err); sec > 0 {
				sleep = time.Duration(sec) * time.Second
			}
			time.Sleep(sleep)
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
		b.reply(ctx, msg.Chat.ID, "access denied")
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}
	cmd, args := parseCommand(text)
	switch cmd {
	case "start", "help":
		b.reply(ctx, msg.Chat.ID, helpText())
	case "stats":
		b.handleStats(ctx, msg.Chat.ID)
	case "list":
		b.handleList(ctx, msg.Chat.ID, args)
	case "export":
		b.handleExport(ctx, msg.Chat.ID, args)
	case "jsonl":
		b.handleJSONL(ctx, msg.Chat.ID)
	case "llm":
		b.handleLLM(ctx, msg.Chat.ID, args)
	default:
		b.reply(ctx, msg.Chat.ID, "unknown command; try /help")
	}
}

func (b *Bot) reply(ctx context.Context, chatID int64, text string) {
	if err := b.client.SendMessage(ctx, chatID, text); err != nil {
		LogAPIError("crm telegram bot sendMessage failed", err, "chat_id", chatID)
	}
}

func (b *Bot) replyHTML(ctx context.Context, chatID int64, text string) {
	if err := b.client.SendHTMLMessageRetry(ctx, chatID, text); err != nil {
		LogAPIError("crm telegram bot sendMessage failed", err, "chat_id", chatID)
		b.reply(ctx, chatID, "send failed: "+userFacingAPIError(err))
	}
}

func (b *Bot) handleStats(ctx context.Context, chatID int64) {
	stats, err := b.store.DBStats(ctx)
	if err != nil {
		b.reply(ctx, chatID, "stats failed: "+err.Error())
		return
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("<b>Leads</b> total=%d", stats.TotalLeads))
	for _, row := range stats.ByStatus {
		lines = append(lines, fmt.Sprintf("%s=%d", html.EscapeString(row.Status), row.Count))
	}
	b.replyHTML(ctx, chatID, strings.Join(lines, "\n"))
}

func (b *Bot) handleList(ctx context.Context, chatID int64, args []string) {
	status := "new"
	limit := int64(10)
	rawOnly := b.exportRawOnly(ctx, args)
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" && !isExportModeArg(args[0]) {
		status = strings.TrimSpace(args[0])
	}
	argIdx := 1
	if len(args) > 0 && isExportModeArg(args[0]) {
		argIdx = 1
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			status = strings.TrimSpace(args[1])
			argIdx = 2
		}
	}
	if len(args) > argIdx {
		if n, err := strconv.ParseInt(args[argIdx], 10, 64); err == nil && n > 0 {
			limit = n
		}
	}
	sort := "engage"
	inboxOnly := status == "new"
	if rawOnly {
		sort = "score"
		inboxOnly = false
	}
	result, err := b.store.List(ctx, store.ListFilter{
		Status:    status,
		Limit:     limit,
		InboxOnly: inboxOnly,
		RawOnly:   rawOnly,
		Sort:      sort,
	})
	if err != nil {
		b.reply(ctx, chatID, "list failed: "+err.Error())
		return
	}
	if len(result.Leads) == 0 {
		b.reply(ctx, chatID, fmt.Sprintf("no leads for status=%q", status))
		return
	}
	mode := "scored"
	if rawOnly {
		mode = "raw"
	}
	header := fmt.Sprintf("<b>Top %d</b> status=<code>%s</code> mode=%s sort=%s", len(result.Leads), html.EscapeString(status), mode, sort)
	var lines []string
	lines = append(lines, header)
	for _, doc := range result.Leads {
		lines = append(lines, FormatLeadListLine(doc))
	}
	if result.NextCursor != nil {
		lines = append(lines, "", "more rows: use /export or Mongo on VPS")
	}
	b.replyHTML(ctx, chatID, strings.Join(lines, "\n"))
}

func (b *Bot) handleExport(ctx context.Context, chatID int64, args []string) {
	if len(args) > 0 && strings.EqualFold(args[0], "people") {
		b.handleExportPeople(ctx, chatID)
		return
	}
	status := ""
	limit := int64(0)
	rawOnly := b.exportRawOnly(ctx, args)
	argIdx := 0
	if len(args) > 0 && isExportModeArg(args[0]) {
		argIdx = 1
	}
	if len(args) > argIdx {
		status = args[argIdx]
	}
	if len(args) > argIdx+1 {
		if n, err := strconv.ParseInt(args[argIdx+1], 10, 64); err == nil && n > 0 {
			limit = n
		}
	}

	result, err := b.store.BuildNDJSON(ctx, store.ExportFilter{
		Status:  status,
		Limit:   limit,
		RawOnly: rawOnly,
	})
	if err != nil {
		b.reply(ctx, chatID, "export failed: "+err.Error())
		return
	}
	defer func() { _ = os.Remove(result.Path) }()

	mode := "all"
	if rawOnly {
		mode = "raw"
	}
	caption := fmt.Sprintf("mongo export rows=%d status=%q mode=%s", result.Rows, statusOrAll(status), mode)
	if err := b.client.SendDocument(ctx, chatID, result.Path, caption); err != nil {
		LogAPIError("crm telegram bot sendDocument failed", err, "chat_id", chatID)
		b.reply(ctx, chatID, "send document failed: "+userFacingAPIError(err))
		return
	}
}

func (b *Bot) handleExportPeople(ctx context.Context, chatID int64) {
	coll := strings.TrimSpace(b.cfg.TelegramPeopleCollection)
	if coll == "" {
		coll = "telegram_people"
	}
	result, err := b.store.ExportTelegramPeopleNDJSON(ctx, coll, 0)
	if err != nil {
		b.reply(ctx, chatID, "people export failed: "+err.Error())
		return
	}
	defer func() { _ = os.Remove(result.Path) }()
	caption := fmt.Sprintf("telegram_people rows=%d", result.Rows)
	if err := b.client.SendDocument(ctx, chatID, result.Path, caption); err != nil {
		LogAPIError("crm telegram bot sendDocument failed", err, "chat_id", chatID)
		b.reply(ctx, chatID, "send document failed: "+userFacingAPIError(err))
	}
}

func (b *Bot) handleJSONL(ctx context.Context, chatID int64) {
	path := strings.TrimSpace(b.cfg.ExportJSONPath)
	if path == "" {
		b.reply(ctx, chatID, "PARSER_EXPORT_JSON path not configured")
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		b.reply(ctx, chatID, "jsonl file missing: "+path)
		return
	}
	caption := fmt.Sprintf("parser jsonl bytes=%d", info.Size())
	if err := b.client.SendDocument(ctx, chatID, path, caption); err != nil {
		LogAPIError("crm telegram bot sendDocument failed", err, "chat_id", chatID)
		b.reply(ctx, chatID, "send document failed: "+userFacingAPIError(err))
	}
}

func userFacingAPIError(err error) string {
	if ae, ok := AsAPIError(err); ok {
		if hint := ae.Hint(); hint != "" {
			return hint
		}
		if ae.Description != "" {
			return ae.Description
		}
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
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

func (b *Bot) handleLLM(ctx context.Context, chatID int64, args []string) {
	if len(args) == 0 {
		b.replyLLMStatus(ctx, chatID)
		return
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "on", "enable", "1", "true":
		if err := b.store.SetLLMScoringEnabled(ctx, true); err != nil {
			b.reply(ctx, chatID, "llm on failed: "+err.Error())
			return
		}
		b.reply(ctx, chatID, "LLM scoring ON (Gemini warm path + engage/ICP/geo on accept). /export returns all analyzed leads.")
	case "off", "disable", "0", "false", "raw":
		if err := b.store.SetLLMScoringEnabled(ctx, false); err != nil {
			b.reply(ctx, chatID, "llm off failed: "+err.Error())
			return
		}
		b.reply(ctx, chatID, "LLM scoring OFF. New accepts are keyword-only (analysis_status=raw). /list and /export default to raw leads.")
	default:
		b.replyLLMStatus(ctx, chatID)
	}
}

func (b *Bot) replyLLMStatus(ctx context.Context, chatID int64) {
	gs, err := b.store.GetGlobalSettings(ctx)
	if err != nil {
		b.reply(ctx, chatID, "llm status failed: "+err.Error())
		return
	}
	state := "ON"
	hint := "Use /llm off for keyword-only raw leads."
	if !gs.LLMScoringEnabled {
		state = "OFF"
		hint = "Use /llm on to re-enable Gemini scoring."
	}
	b.reply(ctx, chatID, fmt.Sprintf("LLM scoring: %s\n%s", state, hint))
}

func (b *Bot) exportRawOnly(ctx context.Context, args []string) bool {
	if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "all") {
		return false
	}
	gs, err := b.store.GetGlobalSettings(ctx)
	if err != nil {
		return false
	}
	return !gs.LLMScoringEnabled
}

func isExportModeArg(arg string) bool {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "all", "raw":
		return true
	default:
		return false
	}
}

func helpText() string {
	return strings.TrimSpace(`BidShard CRM export bot

/stats - lead counts by status
/llm - show LLM scoring toggle (Gemini)
/llm off - keyword-only accepts + raw /list /export
/llm on - restore Gemini scoring
/list - top new inbox leads (HTML)
/list new 15 - status + limit (max 50)
/export - NDJSON from Mongo (top by score)
/export new - filter by status
/export all new - include LLM-scored rows when LLM is off
/export new 100 - status + row limit
/export people - telegram_people NDJSON (CRM_TELEGRAM_PEOPLE_COLLECTION)
/jsonl - parser JSONL file (PARSER_EXPORT_JSON)

Live lead cards: CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS (same bot token).

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
