package telegrambot

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/sink"
)

// LeadNotifier posts accepted leads to Telegram group chats (queued, rate-safe).
type LeadNotifier struct {
	client              *Client
	chatIDs             []int64
	minScore            int
	minScoreNonTelegram int
	notifyHeatMin       string

	jobs chan notifyJob
	once sync.Once
}

type notifyJob struct {
	hashID string
	text   string
}

func NewLeadNotifier(client *Client, chatIDs []int64, minScore, minScoreNonTelegram int, notifyHeatMin string) *LeadNotifier {
	if client == nil || len(chatIDs) == 0 {
		return nil
	}
	n := &LeadNotifier{
		client:              client,
		chatIDs:             chatIDs,
		minScore:            minScore,
		minScoreNonTelegram: minScoreNonTelegram,
		notifyHeatMin:       strings.TrimSpace(notifyHeatMin),
		jobs:                make(chan notifyJob, 512),
	}
	n.once.Do(func() { go n.worker() })
	return n
}

// NotifyLead enqueues a lead card (non-blocking for webhook handler).
func (n *LeadNotifier) NotifyLead(ctx context.Context, doc sink.LeadDoc) {
	if n == nil || n.client == nil || len(n.chatIDs) == 0 {
		return
	}
	minScore := n.notifyMinScore(doc.Source)
	if minScore > 0 && doc.Score < minScore {
		return
	}
	if n.notifyHeatMin != "" && !entity.HeatTierMeetsMin(doc.HeatTier, n.notifyHeatMin) {
		return
	}
	text := FormatLeadNotifyHTML(doc)
	job := notifyJob{hashID: doc.HashID, text: text}
	select {
	case n.jobs <- job:
	default:
		LogAPIError("crm telegram lead notify queue full", errQueueFull, "hash_id", doc.HashID)
	}
}

var errQueueFull = &queueFullError{}

type queueFullError struct{}

func (e *queueFullError) Error() string { return "notify queue full" }

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

func (n *LeadNotifier) worker() {
	for job := range n.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		for _, chatID := range n.chatIDs {
			if err := n.client.SendHTMLMessageRetry(ctx, chatID, job.text); err != nil {
				LogAPIError(
					"crm telegram lead notify failed",
					err,
					"hash_id", job.hashID,
					"chat_id", chatID,
				)
			}
			time.Sleep(120 * time.Millisecond)
		}
		cancel()
	}
}
