package telegrambot

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// LeadNotifier posts accepted leads to Telegram group chats.
type LeadNotifier struct {
	client              *Client
	chatIDs             []int64
	minScore            int
	minScoreNonTelegram int
}

func NewLeadNotifier(client *Client, chatIDs []int64, minScore, minScoreNonTelegram int) *LeadNotifier {
	if client == nil || len(chatIDs) == 0 {
		return nil
	}
	return &LeadNotifier{
		client:              client,
		chatIDs:             chatIDs,
		minScore:            minScore,
		minScoreNonTelegram: minScoreNonTelegram,
	}
}

// NotifyLead sends the lead card asynchronously (fire-and-forget).
func (n *LeadNotifier) NotifyLead(ctx context.Context, doc sink.LeadDoc) {
	if n == nil || n.client == nil || len(n.chatIDs) == 0 {
		return
	}
	minScore := n.notifyMinScore(doc.Source)
	if minScore > 0 && doc.Score < minScore {
		return
	}
	text := FormatLeadNotifyHTML(doc)
	go n.deliver(ctx, doc.HashID, text)
}

func (n *LeadNotifier) notifyMinScore(source string) int {
	src := strings.ToLower(strings.TrimSpace(source))
	if strings.HasPrefix(src, "telegram:") {
		return n.minScore
	}
	if n.minScoreNonTelegram > 0 {
		return n.minScoreNonTelegram
	}
	return n.minScore
}

func (n *LeadNotifier) deliver(_ context.Context, hashID string, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, chatID := range n.chatIDs {
		if err := n.client.SendHTMLMessage(ctx, chatID, text); err != nil {
			slog.Warn("crm telegram lead notify failed",
				"hash_id", hashID,
				"chat_id", chatID,
				"error", err,
			)
		}
	}
}
