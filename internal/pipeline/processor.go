package pipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

type Processor struct {
	Registry *scoring.Registry
	Seen     *dedup.SeenCache
	Store    sink.Store
	MX       validate.MXValidator
}

type ProcessOutcome struct {
	Accepted    bool
	RejectedGeo bool
	Dedup       bool
	DroppedMX   bool
	Lead        model.Lead
}

func (p *Processor) Process(ctx context.Context, task Task) ProcessOutcome {
	text := task.Item.Text()
	out := ProcessOutcome{}

	if res := geo.Filter(text, task.Item.Contact, task.Item.ContactTelegram()); !res.OK {
		out.RejectedGeo = true
		slog.Debug("geo reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", res.Reason)
		return out
	}

	if p.Registry != nil && !p.Registry.Prescan(text) {
		slog.Debug("keyword prescan miss", "round_id", task.RoundID, "source", task.Item.Source)
		return out
	}

	contacts := extract.Extract(text, task.Item.Contact, task.Item.ContactTelegram())
	if contacts.Rejected {
		slog.Debug("contact reject", "round_id", task.RoundID, "reason", contacts.Reason)
		return out
	}
	if len(contacts.Contacts) == 0 {
		slog.Debug("no contacts", "round_id", task.RoundID, "source", task.Item.Source)
		return out
	}

	leadText := &scoring.LeadText{Context: text, Title: task.Item.Title}
	priority := scoring.ScoreText(p.Registry, leadText)
	if priority == scoring.PriorityLow {
		slog.Debug("low score skip", "round_id", task.RoundID, "score", leadText.Score)
		return out
	}
	if extract.OnlyRoleEmails(contacts.Contacts) {
		slog.Debug("role email skip", "round_id", task.RoundID, "source", task.Item.Source)
		return out
	}

	hashID := sink.LeadHashIDFromExtract(contacts.Contacts)
	if hashID == "" {
		slog.Debug("empty hash_id", "round_id", task.RoundID, "source", task.Item.Source)
		return out
	}
	if p.Seen != nil && p.Seen.Seen(hashID) {
		out.Dedup = true
		slog.Debug("seen cache hit", "round_id", task.RoundID, "hash_id", hashID)
		return out
	}

	if p.Store != nil {
		exists, err := p.Store.Exists(ctx, hashID)
		if err != nil {
			slog.Warn("store exists failed", "hash_id", hashID, "error", err)
			return out
		}
		if exists {
			out.Dedup = true
			if p.Seen != nil {
				p.Seen.Mark(hashID)
			}
			slog.Debug("mongo exists", "round_id", task.RoundID, "hash_id", hashID)
			return out
		}
	}

	if priority >= scoring.PriorityMedium {
		if email := primaryEmail(contacts.Contacts); email != "" && p.MX != nil {
			ok, err := p.MX.HasMX(ctx, email)
			if err != nil || !ok {
				out.DroppedMX = true
				slog.Debug("mx reject", "round_id", task.RoundID, "email", maskEmail(email))
				return out
			}
		}
	}

	lead := model.Lead{
		TS:       time.Now().UTC(),
		RoundID:  task.RoundID,
		HashID:   hashID,
		Priority: string(priority),
		Score:    leadText.Score,
		Source:   task.Item.Source,
		Title:    task.Item.Title,
		Contacts: extract.FormatAll(contacts.Contacts),
		Matched:  leadText.Matched,
		Snippet:  text,
	}

	if p.Store != nil {
		if err := p.Store.Upsert(ctx, lead); err != nil {
			if sink.IsDuplicateKey(err) {
				out.Dedup = true
				if p.Seen != nil {
					p.Seen.Mark(hashID)
				}
				slog.Debug("mongo duplicate key", "round_id", task.RoundID, "hash_id", hashID)
				return out
			}
			slog.Warn("store upsert failed", "hash_id", hashID, "error", err)
			return out
		}
	}
	if p.Seen != nil {
		p.Seen.Mark(hashID)
	}

	out.Accepted = true
	out.Lead = lead
	return out
}

func primaryEmail(contacts []extract.Contact) string {
	for _, c := range contacts {
		if c.Type == "email" {
			return c.Value
		}
	}
	return ""
}

func maskEmail(email string) string {
	parts := splitEmail(email)
	if len(parts) != 2 {
		return "***"
	}
	if len(parts[0]) == 0 {
		return "***@" + parts[1]
	}
	return parts[0][:1] + "***@" + parts[1]
}

func splitEmail(email string) []string {
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return []string{email[:i], email[i+1:]}
		}
	}
	return []string{email}
}
