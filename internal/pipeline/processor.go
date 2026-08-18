package pipeline

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/coldpath"
	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/enrich"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

type ICPClassifier interface {
	ClassifyICP(ctx context.Context, text string) (gemini.ICPResult, error)
}

type GeoClassifier interface {
	ClassifyGeo(ctx context.Context, text string, contacts []string, blockedCountries []string) (gemini.GeoResult, error)
}

type Processor struct {
	Registry            *scoring.Registry
	Seen                *dedup.SeenCache
	Store               sink.Store
	MX                  validate.MXValidator
	Junk                *coldpath.Capturer
	SourceRep           *scoring.SourceReputation
	KeywordStats        *sink.KeywordStatsStore
	ICP                 ICPClassifier
	ICPEnabled          bool
	Geo                 GeoClassifier
	GeoEnabled          bool
	GeoBlockCountries   []string
	Enricher            *enrich.Enricher
	TimeDecayEnabled    bool
	PilotTagEnabled     bool
	LeadStatusEnabled   bool
}

type ProcessOutcome struct {
	Accepted     bool
	RejectedGeo  bool
	HardRejected bool
	Dedup        bool
	DroppedMX    bool
	Lead         model.Lead
}

func (p *Processor) Process(ctx context.Context, task Task) ProcessOutcome {
	text := task.Item.Text()
	out := ProcessOutcome{}
	stack := scoring.DetectCompetitorStack(task.Item.CrawlHTML)

	if res := geo.Filter(text, task.Item.Contact, task.Item.ContactTelegram()); !res.OK {
		out.RejectedGeo = true
		slog.Debug("geo reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", res.Reason)
		p.captureJunk(task, coldpath.ReasonGeoReject, res.Reason, 0, nil)
		return out
	}

	if reject, reason := filter.RejectLongCyrillicWithoutLatin(text); reject {
		slog.Debug("lang reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", reason)
		p.captureJunk(task, coldpath.ReasonLangReject, reason, 0, nil)
		return out
	}

	if p.Registry != nil {
		if hit, ok := p.Registry.HardReject(text); ok {
			out.HardRejected = true
			slog.Debug("hard reject", "round_id", task.RoundID, "source", task.Item.Source, "phrase", hit.Phrase)
			return out
		}
	}

	if spam, reason := filter.TelegramSpam(task.Item.Source, text); spam {
		slog.Debug("telegram spam", "round_id", task.RoundID, "source", task.Item.Source, "reason", reason)
		p.captureJunk(task, coldpath.ReasonTelegramSpam, reason, 0, nil)
		return out
	}

	if filter.IsTelegramSource(task.Item.Source) && filter.OnlyContactNoContext(text) {
		slog.Debug("contact-only skip", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(task, coldpath.ReasonTelegramSpam, "contact only", 0, nil)
		return out
	}

	if p.Registry != nil && !p.Registry.Prescan(text) {
		slog.Debug("keyword prescan miss", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(task, coldpath.ReasonKeywordPrescan, "", 0, nil)
		return out
	}

	contacts := extract.Extract(text, task.Item.Contact, task.Item.ContactTelegram())
	if contacts.Rejected {
		slog.Debug("contact reject", "round_id", task.RoundID, "reason", contacts.Reason)
		p.captureJunk(task, coldpath.ReasonContactReject, contacts.Reason, 0, nil)
		return out
	}
	if len(contacts.Contacts) == 0 {
		slog.Debug("no contacts", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(task, coldpath.ReasonNoContacts, "", 0, nil)
		return out
	}
	if hasEmailContact(contacts.Contacts) && validate.EmailWithoutPainContext(text) {
		slog.Debug("email without pain context", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(task, coldpath.ReasonEmailNoContext, "", 0, nil)
		return out
	}
	if blacklisted, detail := blacklistContact(contacts.Contacts); blacklisted {
		slog.Debug("blacklist reject", "round_id", task.RoundID, "source", task.Item.Source, "detail", detail)
		p.captureJunk(task, coldpath.ReasonBlacklist, detail, 0, nil)
		return out
	}

	var enrichResult enrich.Result
	if p.Enricher != nil {
		enrichResult = p.Enricher.Enrich(ctx, enrich.Input{
			Source:      task.Item.Source,
			Contacts:    contacts.Contacts,
			DisplayHint: task.Item.Contact,
		})
		if enrichResult.GeoBlocked {
			out.RejectedGeo = true
			slog.Debug("rdap geo reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", enrichResult.GeoReason)
			p.captureJunk(task, coldpath.ReasonGeoReject, enrichResult.GeoReason, 0, nil)
			return out
		}
		stack = enrich.MergeStack(stack, enrich.CompetitorStackFromResult(enrichResult.Stack))
	}

	leadText := &scoring.LeadText{Context: text, Title: task.Item.Title}
	priority := scoring.ScoreWithBoosts(p.Registry, leadText, task.Item.Source, stack, p.SourceRep, scoring.ScoreOpts{
		PostedAt:  task.Item.PostedAt,
		TimeDecay: p.TimeDecayEnabled,
	})
	if priority == scoring.PriorityLow {
		slog.Debug("low score skip", "round_id", task.RoundID, "score", leadText.Score)
		p.captureJunk(task, coldpath.ReasonLowScore, "", leadText.Score, leadText.Matched)
		return out
	}
	if extract.OnlyRoleEmails(contacts.Contacts) {
		slog.Debug("role email skip", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(task, coldpath.ReasonRoleEmail, "", leadText.Score, leadText.Matched)
		return out
	}

	hashID := sink.LeadHashIDFromExtract(contacts.Contacts)
	if hashID == "" {
		slog.Debug("empty hash_id", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(task, coldpath.ReasonEmptyHash, "", leadText.Score, leadText.Matched)
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

	var geoResult gemini.GeoResult
	if p.GeoEnabled && p.Geo != nil && scoring.MeetsMinPriority(priority, scoring.PriorityMedium) {
		contactStrs := extract.FormatAll(contacts.Contacts)
		blocked := p.GeoBlockCountries
		if len(blocked) == 0 {
			blocked = []string{"RU", "BY"}
		}
		if res, err := p.Geo.ClassifyGeo(ctx, text, contactStrs, blocked); err != nil {
			slog.Warn("geo classify failed", "round_id", task.RoundID, "error", err)
		} else {
			geoResult = res
			if res.ShouldReject(blocked) {
				out.RejectedGeo = true
				slog.Debug("gemini geo reject", "round_id", task.RoundID, "source", task.Item.Source, "detail", res.Detail())
				p.captureJunk(task, coldpath.ReasonGeoGeminiReject, res.Detail(), leadText.Score, leadText.Matched)
				return out
			}
		}
	}

	var icpResult gemini.ICPResult
	if p.ICPEnabled && p.ICP != nil && scoring.MeetsMinPriority(priority, scoring.PriorityMedium) {
		if res, err := p.ICP.ClassifyICP(ctx, text); err != nil {
			slog.Warn("icp classify failed", "round_id", task.RoundID, "error", err)
		} else {
			icpResult = res
			leadText.Score, _ = gemini.ApplyICPToScore(leadText.Score, icpResult, highMinFromReg(p.Registry))
			priority = scoring.PriorityFromScore(p.Registry, leadText.Score)
			if icpResult.ICP == "none" && !icpResult.Hot && priority == scoring.PriorityLow {
				p.captureJunk(task, coldpath.ReasonICPReject, icpResult.Why, leadText.Score, leadText.Matched)
				return out
			}
			if icpResult.Hot && priority == scoring.PriorityMedium {
				priority = scoring.PriorityHigh
			}
		}
	}

	if scoring.MeetsMinPriority(priority, scoring.PriorityMedium) {
		if email := primaryEmail(contacts.Contacts); email != "" && p.MX != nil {
			ok, err := p.MX.HasMX(ctx, email)
			if err != nil || !ok {
				out.DroppedMX = true
				slog.Debug("mx reject", "round_id", task.RoundID, "email", maskEmail(email))
				p.captureJunk(task, coldpath.ReasonMXReject, "mx lookup failed or no records", leadText.Score, leadText.Matched)
				return out
			}
		}
	}

	lead := model.Lead{
		TS:             time.Now().UTC(),
		RoundID:        task.RoundID,
		HashID:         hashID,
		Priority:       string(priority),
		Score:          leadText.Score,
		Source:         task.Item.Source,
		Title:          task.Item.Title,
		Contacts:       extract.FormatAll(contacts.Contacts),
		Matched:        leadText.Matched,
		Snippet:        text,
		ICP:            icpResult.ICP,
		Hot:            icpResult.Hot,
		SpendTier:      icpResult.SpendTier,
		ICPWhy:         icpResult.Why,
		GeoCountry:     geoResult.PersonCountry,
		CompanyCountry: geoResult.CompanyCountry,
		CompanyName:    geoResult.CompanyName,
		GeoSignals:     append(append([]string(nil), geoResult.RegistrationSignals...), geoResult.RUBYSignals...),
		GeoWhy:         geoResult.Why,
		WhoisCountry:   enrichResult.RDAPCountry,
		DomainAgeDays:  enrichResult.DomainAgeDays,
		DisplayName:    enrichResult.DisplayName,
		GravatarName:   enrichResult.GravatarName,
		EmailVerified:  enrichResult.SMTPValid,
		PostedAt:       task.Item.PostedAt,
		Stack:          stack,
		Lang:           filter.DetectLanguage(text),
	}

	if p.PilotTagEnabled {
		qualified, pilotTags := scoring.PilotQualified(icpResult.SpendTier, stack, text)
		lead.PilotQualified = qualified
		lead.Tags = append(lead.Tags, pilotTags...)
	}
	if p.LeadStatusEnabled {
		lead.Status = "new"
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
	if p.SourceRep != nil {
		p.SourceRep.RecordAccepted(task.Item.Source)
	}
	metrics.RecordAccepted(task.Item.Source, string(priority))
	p.recordKeywordOutcomes(ctx, leadText.Matched, false)

	out.Accepted = true
	out.Lead = lead
	return out
}

func highMinFromReg(reg *scoring.Registry) int {
	_, _, _, highMin, _ := reg.Snapshot()
	return highMin
}

func (p *Processor) captureJunk(task Task, reason, detail string, score int, matched []string) {
	metrics.RecordJunk(reason)
	p.recordKeywordOutcomes(context.Background(), matched, true)
	if p.SourceRep != nil {
		p.SourceRep.RecordJunk(task.Item.Source)
	}
	if p == nil || p.Junk == nil {
		return
	}
	contact := task.Item.Contact
	if contact == "" {
		contact = task.Item.ContactTelegram()
	}
	p.Junk.TryCapture(coldpath.Event{
		RoundID:      task.RoundID,
		Source:       task.Item.Source,
		Title:        task.Item.Title,
		Snippet:      task.Item.Text(),
		ContactHint:  contact,
		Reason:       reason,
		ReasonDetail: detail,
		Score:        score,
		Matched:      append([]string(nil), matched...),
	})
}

func (p *Processor) recordKeywordOutcomes(ctx context.Context, matched []string, junk bool) {
	if p == nil || p.KeywordStats == nil || p.Registry == nil || len(matched) == 0 {
		return
	}
	for _, id := range p.Registry.KeywordIDsFromMatched(matched) {
		if err := p.KeywordStats.RecordOutcome(ctx, id, junk); err != nil {
			slog.Debug("keyword stats record failed", "keyword_id", id, "junk", junk, "error", err)
		}
	}
}

func blacklistContact(contacts []extract.Contact) (bool, string) {
	for _, c := range contacts {
		switch c.Type {
		case "email":
			if validate.IsBlacklisted(c.Value, "") {
				return true, "blacklisted email/domain"
			}
		case "domain":
			if validate.IsBlacklisted("", c.Value) {
				return true, "blacklisted domain"
			}
		}
	}
	return false, ""
}

func hasEmailContact(contacts []extract.Contact) bool {
	for _, c := range contacts {
		if c.Type == "email" {
			return true
		}
	}
	return false
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
	email = strings.TrimSpace(email)
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return []string{email[:i], email[i+1:]}
		}
	}
	return []string{email}
}
