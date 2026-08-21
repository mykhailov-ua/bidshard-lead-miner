package pipeline

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/coldpath"
	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/diag"
	"github.com/bidshard/parser/internal/enrich"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/sources/tgweb"
	"github.com/bidshard/parser/internal/validate"
	"github.com/bidshard/parser/internal/warmpath"
)

type ICPClassifier interface {
	ClassifyICP(ctx context.Context, text string) (gemini.ICPResult, error)
}

type GeoClassifier interface {
	ClassifyGeo(ctx context.Context, text string, contacts []string, blockedCountries []string) (gemini.GeoResult, error)
}

type EngagementClassifier interface {
	ClassifyEngagement(ctx context.Context, in gemini.EngagementInput) (gemini.EngagementResult, error)
}

type LeadClusterer interface {
	CheckDuplicate(ctx context.Context, hashID, text string) (bool, string, error)
	Record(ctx context.Context, hashID, text string) error
}

type EmbedPrescanner interface {
	EvaluatePain(ctx context.Context, text string) (gemini.PrescanVerdict, error)
	EvaluateSpam(ctx context.Context, text string) (gemini.PrescanVerdict, error)
}

type EnrichSynthesizer interface {
	SynthesizeEnrichment(ctx context.Context, in gemini.EnrichSynthInput) (gemini.EnrichSynthResult, error)
}

type Processor struct {
	Registry              *scoring.Registry
	Seen                  *dedup.SeenCache
	Store                 sink.Store
	MX                    validate.MXValidator
	Junk                  *coldpath.Capturer
	SourceRep             *scoring.SourceReputation
	KeywordStats          *sink.KeywordStatsStore
	ICP                   ICPClassifier
	ICPEnabled            bool
	ICPTgWebEnabled       bool
	TgWebPrescanMode      scoring.TgWebPrescanMode
	Geo                   GeoClassifier
	GeoEnabled            bool
	GeoBlockCountries     []string
	Engage                EngagementClassifier
	EngageEnabled         bool
	Prescan               EmbedPrescanner
	PrescanEnabled        bool
	LeadCluster           LeadClusterer
	LeadClusterEnabled    bool
	Enricher              *enrich.Enricher
	EnrichSynth           EnrichSynthesizer
	EnrichSynthEnabled    bool
	TimeDecayEnabled      bool
	PilotTagEnabled       bool
	LeadStatusEnabled     bool
	GeminiDefer           bool
	WarmPath              *warmpath.Capturer
	EntityRecorder        entity.Recorder
	EntitySightings       bool
	CrossSourceHot        bool
	CrossSourceWindow     time.Duration
	CrossSourceBoost      int
	EntityClassifyEnabled bool
	EntityClassify        *warmpath.EntityClassifyCapturer
	EntityHeatEnabled     bool
	EntityHeat            entity.HeatConfig
	hashInflight          sync.Map // hash_id -> struct{}
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
	task.Item.CrawlHTML = model.LimitCrawlHTML(task.Item.CrawlHTML)
	text := task.Item.Text()
	out := ProcessOutcome{}
	stack := scoring.DetectCompetitorStack(task.Item.CrawlHTML)

	if strings.HasPrefix(task.Item.Source, "fixture:") {
		slog.Debug("fixture skip", "round_id", task.RoundID, "source", task.Item.Source)
		return out
	}

	if res := geo.Filter(text, task.Item.Contact, task.Item.ContactTelegram()); !res.OK {
		out.RejectedGeo = true
		slog.Debug("geo reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", res.Reason)
		p.captureJunk(ctx, task, coldpath.ReasonGeoReject, res.Reason, 0, nil)
		return out
	}

	if filter.IsTgWebSource(task.Item.Source) {
		if domain := tgweb.SiteDomainFromSource(task.Item.Source); geo.IsBlockedTLD(domain) {
			out.RejectedGeo = true
			slog.Debug("geo reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", "ru/by tld")
			p.captureJunk(ctx, task, coldpath.ReasonGeoReject, "ru/by tld", 0, nil)
			return out
		}
		if reject, reason := filter.RejectHTMLBoilerplate(text); reject {
			logTgWebReject(task, reason, nil)
			p.captureJunk(ctx, task, coldpath.ReasonLangReject, reason, 0, nil)
			return out
		}
	}

	if reject, reason := filter.RejectLongCyrillicWithoutLatin(text); reject {
		slog.Debug("lang reject", "round_id", task.RoundID, "source", task.Item.Source, "reason", reason)
		p.captureJunk(ctx, task, coldpath.ReasonLangReject, reason, 0, nil)
		return out
	}

	if p.Registry != nil {
		if hit, ok := p.Registry.HardReject(text); ok {
			// Aggressive tgweb: site LPR contacts bypass global hard-reject phrases (e.g. "casino" on affiliate pages).
			if filter.IsTgWebSource(task.Item.Source) && tgweb.AggressivePrescanFromContact(task.Item.Source, task.Item.Contact) {
				logTgWebInfo(task, "tgweb hard reject bypassed for site lpr", "phrase", hit.Phrase)
			} else {
				if filter.IsTgWebSource(task.Item.Source) {
					logTgWebReject(task, "hard reject", []any{"phrase", hit.Phrase})
				} else {
					slog.Debug("hard reject",
						"round_id", task.RoundID,
						"source", task.Item.Source,
						"phrase", hit.Phrase,
						"text_preview", diag.Preview(text, diag.DefaultPreview),
					)
				}
				out.HardRejected = true
				return out
			}
		}
	}

	if spam, reason := filter.TelegramSpam(task.Item.Source, text); spam {
		slog.Debug("telegram spam", "round_id", task.RoundID, "source", task.Item.Source, "reason", reason)
		p.captureJunk(ctx, task, coldpath.ReasonTelegramSpam, reason, 0, nil)
		return out
	}

	if filter.IsTelegramSource(task.Item.Source) && filter.OnlyContactNoContext(text) {
		slog.Debug("contact-only skip", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(ctx, task, coldpath.ReasonTelegramSpam, "contact only", 0, nil)
		return out
	}

	prescanOK := p.Registry == nil || scoring.PrescanPasses(task.Item.Source, p.Registry, text)
	if !prescanOK && p.TgWebPrescanMode.Aggressive() && tgweb.AggressivePrescanFromContact(task.Item.Source, task.Item.Contact) {
		// Site LPR with aggressive prescan skips keyword prescan gate.
		prescanOK = true
		logTgWebInfo(task, "tgweb aggressive prescan pass")
	}
	if p.PrescanEnabled && p.Prescan != nil {
		if !prescanOK {
			if verdict, err := p.Prescan.EvaluatePain(ctx, text); err != nil {
				slog.Debug("embed prescan pain failed", "round_id", task.RoundID, "source", task.Item.Source, "error", err)
			} else if verdict.PainMatch {
				prescanOK = true
				slog.Debug("embed prescan pass", "round_id", task.RoundID, "source", task.Item.Source, "pain_score", verdict.PainScore)
			}
		} else if filter.IsTelegramSource(task.Item.Source) {
			if verdict, err := p.Prescan.EvaluateSpam(ctx, text); err != nil {
				slog.Debug("embed prescan spam failed", "round_id", task.RoundID, "source", task.Item.Source, "error", err)
			} else if verdict.SpamMatch {
				slog.Debug("embed spam reject", "round_id", task.RoundID, "source", task.Item.Source, "spam_score", verdict.SpamScore)
				p.captureJunk(ctx, task, coldpath.ReasonEmbedSpam, "", 0, nil)
				return out
			}
		}
	}
	if p.Registry != nil && !prescanOK {
		logTgWebReject(task, "keyword prescan miss", nil)
		p.captureJunk(ctx, task, coldpath.ReasonKeywordPrescan, "", 0, nil)
		return out
	}

	contacts := extract.Extract(text, task.Item.Contact, task.Item.ContactTelegram())
	if filter.IsTgWebSource(task.Item.Source) {
		// Collapse to one on-domain LPR; drop channel telegram and role emails.
		contacts.Contacts = tgweb.FilterPipelineContacts(task.Item.Source, task.Item.Contact, contacts.Contacts)
	}
	if contacts.Rejected {
		slog.Debug("contact reject", "round_id", task.RoundID, "reason", contacts.Reason)
		p.captureJunk(ctx, task, coldpath.ReasonContactReject, contacts.Reason, 0, nil)
		return out
	}
	if len(contacts.Contacts) == 0 {
		if filter.IsTgWebSource(task.Item.Source) {
			logTgWebReject(task, "no contacts after tgweb filter", nil)
		} else {
			slog.Debug("no contacts", "round_id", task.RoundID, "source", task.Item.Source)
		}
		p.captureJunk(ctx, task, coldpath.ReasonNoContacts, "", 0, nil)
		return out
	}
	if hasEmailContact(contacts.Contacts) && validate.EmailWithoutPainContext(text) {
		// Allow bare site emails when tgweb already validated an on-domain LPR at crawl time.
		if filter.IsTgWebSource(task.Item.Source) && tgwebEmailAllowed(task.Item, contacts.Contacts, p.TgWebPrescanMode, text) {
			logTgWebInfo(task, "tgweb email allowed with site lpr")
		} else {
			logTgWebReject(task, "email without pain context", nil)
			p.captureJunk(ctx, task, coldpath.ReasonEmailNoContext, "", 0, nil)
			return out
		}
	}
	if blacklisted, detail := blacklistContact(contacts.Contacts); blacklisted {
		slog.Debug("blacklist reject", "round_id", task.RoundID, "source", task.Item.Source, "detail", detail)
		p.captureJunk(ctx, task, coldpath.ReasonBlacklist, detail, 0, nil)
		return out
	}
	if extract.OnlyRoleEmails(contacts.Contacts) {
		if filter.IsTgWebSource(task.Item.Source) {
			logTgWebReject(task, "role email only", nil)
		} else {
			slog.Debug("role email skip", "round_id", task.RoundID, "source", task.Item.Source)
		}
		p.captureJunk(ctx, task, coldpath.ReasonRoleEmail, "", 0, nil)
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
			p.captureJunk(ctx, task, coldpath.ReasonGeoReject, enrichResult.GeoReason, 0, nil)
			return out
		}
		stack = enrich.MergeStack(stack, enrich.CompetitorStackFromResult(enrichResult.Stack))
	}

	leadText := &scoring.LeadText{Context: text, Title: task.Item.Title}
	priority := scoring.ScoreWithBoosts(p.Registry, leadText, task.Item.Source, stack, p.SourceRep, scoring.ScoreOpts{
		PostedAt:  task.Item.PostedAt,
		TimeDecay: p.TimeDecayEnabled,
	})
	if priority == scoring.PriorityLow && p.TgWebPrescanMode.Aggressive() && filter.IsTgWebSource(task.Item.Source) && tgweb.HasSiteLPRContact(task.Item.Source, contacts.Contacts) {
		// Lift site LPR leads only when text has tracker pain, not generic affiliate HTML.
		if validate.HasStrictPainContext(text) {
			if min := mediumMinFromReg(p.Registry); leadText.Score < min {
				leadText.Score = min
				priority = scoring.PriorityFromScore(p.Registry, leadText.Score)
				slog.Debug("tgweb aggressive score floor", "round_id", task.RoundID, "source", task.Item.Source, "score", leadText.Score)
			}
		}
	}
	if priority == scoring.PriorityLow {
		slog.Debug("low score skip", "round_id", task.RoundID, "score", leadText.Score)
		p.captureJunk(ctx, task, coldpath.ReasonLowScore, "", leadText.Score, leadText.Matched)
		return out
	}
	hashID := leadHashID(task, contacts.Contacts)
	if hashID == "" {
		slog.Debug("empty hash_id", "round_id", task.RoundID, "source", task.Item.Source)
		p.captureJunk(ctx, task, coldpath.ReasonEmptyHash, "", leadText.Score, leadText.Matched)
		return out
	}
	if p.Seen != nil && p.Seen.Seen(hashID) {
		result := p.recordEntitySighting(ctx, entitySightingInput(task, contacts.Contacts, hashID, leadText.Matched, stack, text, leadText.Score, entity.ResolveInput{}))
		p.maybePatchCanonicalLead(ctx, hashID, result)
		out.Dedup = true
		slog.Debug("seen cache hit", "round_id", task.RoundID, "hash_id", hashID)
		return out
	}

	if !p.acquireHash(hashID) {
		// Another worker is already past Seen/Exists for this hash_id; skip duplicate Gemini/Mongo work.
		result := p.recordEntitySighting(ctx, entitySightingInput(task, contacts.Contacts, hashID, leadText.Matched, stack, text, leadText.Score, entity.ResolveInput{}))
		p.maybePatchCanonicalLead(ctx, hashID, result)
		out.Dedup = true
		slog.Debug("hash inflight dedup", "round_id", task.RoundID, "hash_id", hashID)
		return out
	}
	defer p.releaseHash(hashID)

	if p.Store != nil {
		exists, err := p.Store.Exists(ctx, hashID)
		if err != nil {
			slog.Warn("store exists failed", "hash_id", hashID, "error", err)
			return out
		}
		if exists {
			result := p.recordEntitySighting(ctx, entitySightingInput(task, contacts.Contacts, hashID, leadText.Matched, stack, text, leadText.Score, entity.ResolveInput{}))
			p.maybePatchCanonicalLead(ctx, hashID, result)
			out.Dedup = true
			if p.Seen != nil {
				p.Seen.Mark(hashID)
			}
			slog.Debug("mongo exists", "round_id", task.RoundID, "hash_id", hashID)
			return out
		}
	}

	// Reserve hash before Gemini/MX so parallel workers and MX rejects do not repeat API work.
	if p.Seen != nil {
		p.Seen.Mark(hashID)
	}

	if p.LeadClusterEnabled && p.LeadCluster != nil && scoring.MeetsMinPriority(priority, scoring.PriorityHigh) {
		if dup, clusterOf, err := p.LeadCluster.CheckDuplicate(ctx, hashID, text); err != nil {
			slog.Warn("lead cluster check failed", "round_id", task.RoundID, "hash_id", hashID, "error", err)
		} else if dup {
			result := p.recordEntitySighting(ctx, entitySightingInput(task, contacts.Contacts, hashID, leadText.Matched, stack, text, leadText.Score, entity.ResolveInput{}))
			p.maybePatchCanonicalLead(ctx, hashID, result)
			out.Dedup = true
			slog.Debug("semantic lead dedup", "round_id", task.RoundID, "hash_id", hashID, "cluster_of", clusterOf)
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
		if res, err := p.Geo.ClassifyGeo(ctx, geoClassifyText(task.Item, text), contactStrs, blocked); err != nil {
			slog.Warn("geo classify failed", "round_id", task.RoundID, "error", err)
		} else {
			geoResult = res
			if res.ShouldReject(blocked) {
				out.RejectedGeo = true
				slog.Debug("gemini geo reject", "round_id", task.RoundID, "source", task.Item.Source, "detail", res.Detail())
				p.captureJunk(ctx, task, coldpath.ReasonGeoGeminiReject, res.Detail(), leadText.Score, leadText.Matched)
				return out
			}
		}
	}

	var icpResult gemini.ICPResult
	if p.icpClassifierEnabled(task.Item.Source) && scoring.MeetsMinPriority(priority, scoring.PriorityMedium) {
		if res, err := p.ICP.ClassifyICP(ctx, text); err != nil {
			slog.Warn("icp classify failed", "round_id", task.RoundID, "error", err)
		} else {
			icpResult = res
			leadText.Score, _ = gemini.ApplyICPToScore(leadText.Score, icpResult, highMinFromReg(p.Registry))
			priority = scoring.PriorityFromScore(p.Registry, leadText.Score)
			if filter.IsTgWebSource(task.Item.Source) && p.ICPTgWebEnabled {
				// Tgweb sync ICP runs even under PARSER_GEMINI_DEFER; reject non-ICP site leads inline.
				if icpResult.ICP == "none" && !icpResult.Hot {
					logTgWebReject(task, "icp reject", []any{"why", icpResult.Why})
					p.captureJunk(ctx, task, coldpath.ReasonICPReject, icpResult.Why, leadText.Score, leadText.Matched)
					return out
				}
			} else if icpResult.ICP == "none" && !icpResult.Hot && priority == scoring.PriorityLow {
				p.captureJunk(ctx, task, coldpath.ReasonICPReject, icpResult.Why, leadText.Score, leadText.Matched)
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
				p.captureJunk(ctx, task, coldpath.ReasonMXReject, "mx lookup failed or no records", leadText.Score, leadText.Matched)
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

	if p.PilotTagEnabled && !p.GeminiDefer {
		p.applyPilotAndOutreach(ctx, &lead, priority, text, stack, icpResult, contacts.Contacts)
	} else if p.PilotTagEnabled {
		// Defer mode: rule-based pilot tags only; LLM engage runs in warm path.
		qualified, pilotTags := scoring.PilotQualified("", stack, text)
		lead.PilotQualified = qualified
		lead.Tags = append(lead.Tags, pilotTags...)
	}
	if !p.GeminiDefer {
		p.applyEnrichSynth(ctx, &lead, priority, text, task.Item.Source, enrichResult, geoResult, icpResult)
	}
	if p.LeadStatusEnabled {
		lead.Status = "new"
	}
	if p.GeminiDefer {
		lead.AnalysisStatus = "pending"
	}
	if strings.HasPrefix(strings.ToLower(lead.Source), "ads_txt:") {
		// Rule tag only; does not change accept gates or entity keys (those use SupplyDomainFromSource).
		lead.Tags = entity.AppendUniqueTag(lead.Tags, scoring.TagPublisherSurface)
	}

	entityResult := p.recordEntitySighting(ctx, entitySightingInput(task, contacts.Contacts, hashID, leadText.Matched, stack, text, leadText.Score, entity.ResolveInput{
		CompanyName:  lead.CompanyName,
		DisplayName:  lead.DisplayName,
		GravatarName: lead.GravatarName,
		Source:       lead.Source,
	}))
	if entityResult.EntityID != "" {
		lead.EntityID = entityResult.EntityID
		lead.EntitySightingCount = entityResult.SightingCount
		lead.EntitySourceCount = entityResult.SourceCount
	}
	if p.EntityHeatEnabled {
		entity.ApplyEntityHeatToLead(&lead, entityResult, p.Registry, p.EntityHeat)
		if entityResult.CrossSourceHot {
			priority = scoring.Priority(lead.Priority)
			slog.Info("entity heat boost",
				"round_id", task.RoundID,
				"entity_id", entityResult.EntityID,
				"hash_id", hashID,
				"heat_tier", entityResult.HeatTier,
				"heat_score", entityResult.HeatScore,
				"source_count", entityResult.SourceCount,
				"score", lead.Score,
				"priority", lead.Priority,
			)
		}
	} else if p.CrossSourceHot && entityResult.CrossSourceHot {
		entity.ApplyCrossSourceHotBoost(&lead, p.Registry, p.CrossSourceBoost)
		priority = scoring.Priority(lead.Priority)
		slog.Info("cross-source hot boost",
			"round_id", task.RoundID,
			"entity_id", entityResult.EntityID,
			"hash_id", hashID,
			"source_count", entityResult.SourceCount,
			"score", lead.Score,
			"priority", lead.Priority,
		)
	}

	if p.Store != nil {
		if err := p.Store.Upsert(ctx, lead); err != nil {
			if sink.IsDuplicateKey(err) {
				dupResult := p.recordEntitySighting(ctx, entitySightingInput(task, contacts.Contacts, hashID, leadText.Matched, stack, text, leadText.Score, entity.ResolveInput{
					CompanyName:  lead.CompanyName,
					DisplayName:  lead.DisplayName,
					GravatarName: lead.GravatarName,
					Source:       lead.Source,
				}))
				p.maybePatchCanonicalLead(ctx, hashID, dupResult)
				out.Dedup = true
				if p.Seen != nil {
					p.Seen.Mark(hashID)
				}
				slog.Debug("mongo duplicate key",
					"round_id", task.RoundID,
					"hash_id", hashID,
					"source", task.Item.Source,
				)
				return out
			}
			slog.Warn("store upsert failed",
				"round_id", task.RoundID,
				"hash_id", hashID,
				"source", task.Item.Source,
				"priority", string(priority),
				"score", leadText.Score,
				"contact", task.Item.MaskedContact(),
				"snippet_preview", diag.Preview(text, 300),
				"error", err,
			)
			return out
		}
		metrics.RecordLeadWritten()
	}
	if p.LeadClusterEnabled && p.LeadCluster != nil && scoring.MeetsMinPriority(priority, scoring.PriorityHigh) {
		if err := p.LeadCluster.Record(ctx, hashID, text); err != nil {
			slog.Debug("lead cluster record failed", "round_id", task.RoundID, "hash_id", hashID, "error", err)
		}
	}
	if p.Seen != nil {
		p.Seen.Mark(hashID)
	}
	if p.GeminiDefer && p.WarmPath != nil {
		// Queue for warm-path Gemini after Mongo upsert; lead already stored with analysis_status=pending.
		p.WarmPath.TryCapture(warmpath.Event{
			HashID:        hashID,
			RoundID:       task.RoundID,
			Source:        task.Item.Source,
			Title:         task.Item.Title,
			Snippet:       text,
			Contacts:      extract.FormatAll(contacts.Contacts),
			ContactTypes:  contactTypes(contacts.Contacts),
			Stack:         append([]string(nil), stack...),
			Score:         leadText.Score,
			Priority:      string(priority),
			Matched:       append([]string(nil), leadText.Matched...),
			Domain:        enrichResult.Domain,
			RDAPCountry:   enrichResult.RDAPCountry,
			DomainAgeDays: enrichResult.DomainAgeDays,
			DisplayName:   enrichResult.DisplayName,
			EntityID:      lead.EntityID,
			EntityHeat:    lead.EntityHeat,
			HeatTier:      lead.HeatTier,
		})
	}
	if p.SourceRep != nil {
		p.SourceRep.RecordAccepted(task.Item.Source)
	}
	metrics.RecordAccepted(task.Item.Source, string(priority))
	p.recordKeywordOutcomes(ctx, leadText.Matched, false)

	out.Accepted = true
	out.Lead = lead
	slog.Info("lead accepted",
		"round_id", task.RoundID,
		"source", task.Item.Source,
		"priority", string(priority),
		"score", leadText.Score,
		"hash_id", hashID,
		"matched", leadText.Matched,
		"contact", task.Item.MaskedContact(),
		"snippet_preview", diag.Preview(text, 300),
	)
	return out
}

func highMinFromReg(reg *scoring.Registry) int {
	_, _, _, highMin, _ := reg.Snapshot()
	return highMin
}

func mediumMinFromReg(reg *scoring.Registry) int {
	if reg == nil {
		return 25
	}
	_, _, _, _, mediumMin := reg.Snapshot()
	return mediumMin
}

func tgwebEmailAllowed(item model.RawItem, contacts []extract.Contact, mode scoring.TgWebPrescanMode, text string) bool {
	if hasDirectMessengerContact(contacts) {
		return validate.HasStrictPainContext(text)
	}
	if !mode.Aggressive() {
		return false
	}
	if tgweb.HasSiteLPRContact(item.Source, contacts) {
		return validate.HasStrictPainContext(text)
	}
	return tgweb.AggressivePrescanFromContact(item.Source, item.Contact) && validate.HasStrictPainContext(text)
}

func (p *Processor) applyPilotAndOutreach(
	ctx context.Context,
	lead *model.Lead,
	priority scoring.Priority,
	text string,
	stack []string,
	icpResult gemini.ICPResult,
	contacts []extract.Contact,
) {
	if p.EngageEnabled && p.Engage != nil && priority == scoring.PriorityHigh {
		res, err := p.Engage.ClassifyEngagement(ctx, gemini.EngagementInput{
			Text:         text,
			Stack:        stack,
			ICP:          icpResult.ICP,
			SpendTier:    icpResult.SpendTier,
			Hot:          icpResult.Hot,
			ContactTypes: contactTypes(contacts),
			Source:       lead.Source,
		})
		if err != nil {
			slog.Warn("engagement classify failed", "round_id", lead.RoundID, "source", lead.Source, "error", err)
		} else {
			qualified, pilotTags := gemini.ApplyEngagementPilot(res)
			lead.PilotQualified = qualified
			lead.PilotWhy = res.PilotWhy
			lead.Tags = append(lead.Tags, pilotTags...)
			lead.OutreachChannel = res.OutreachChannel
			lead.OutreachAngle = res.OutreachAngle
			lead.OutreachDraft = res.OutreachDraft
			return
		}
	}

	qualified, pilotTags := scoring.PilotQualified(icpResult.SpendTier, stack, text)
	lead.PilotQualified = qualified
	lead.Tags = append(lead.Tags, pilotTags...)
}

func (p *Processor) applyEnrichSynth(
	ctx context.Context,
	lead *model.Lead,
	priority scoring.Priority,
	text, source string,
	enrichResult enrich.Result,
	geoResult gemini.GeoResult,
	icpResult gemini.ICPResult,
) {
	if !p.EnrichSynthEnabled || p.EnrichSynth == nil || priority != scoring.PriorityHigh {
		return
	}
	synth, err := p.EnrichSynth.SynthesizeEnrichment(ctx, gemini.EnrichSynthInput{
		Snippet:        text,
		Source:         source,
		Domain:         enrichResult.Domain,
		RDAPCountry:    enrichResult.RDAPCountry,
		DomainAgeDays:  enrichResult.DomainAgeDays,
		Stack:          enrichResult.Stack,
		DisplayName:    enrichResult.DisplayName,
		GeoCountry:     geoResult.PersonCountry,
		CompanyCountry: geoResult.CompanyCountry,
		ICP:            icpResult.ICP,
	})
	if err != nil {
		slog.Warn("enrich synth failed", "round_id", lead.RoundID, "source", lead.Source, "error", err)
		return
	}
	lead.CompanyType = synth.CompanyType
	lead.EnrichSummary = synth.Summary
	lead.GeoConfidence = synth.GeoConfidence
}

func contactTypes(contacts []extract.Contact) []string {
	seen := make(map[string]struct{}, len(contacts))
	out := make([]string, 0, len(contacts))
	for _, c := range contacts {
		t := strings.ToLower(strings.TrimSpace(c.Type))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func (p *Processor) captureJunk(ctx context.Context, task Task, reason, detail string, score int, matched []string) {
	slog.Debug("pipeline reject",
		"round_id", task.RoundID,
		"source", task.Item.Source,
		"contact", task.Item.MaskedContact(),
		"junk_reason", reason,
		"junk_detail", detail,
		"score", score,
		"matched", matched,
		"text_preview", diag.Preview(task.Item.Text(), diag.DefaultPreview),
	)
	metrics.RecordJunk(reason)
	p.recordKeywordOutcomes(ctx, matched, true)
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

func hasDirectMessengerContact(contacts []extract.Contact) bool {
	for _, c := range contacts {
		switch c.Type {
		case "telegram", "skype":
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

func leadHashID(task Task, contacts []extract.Contact) string {
	// Scope dedup by site domain so the same email on two affiliate sites stays distinct.
	if filter.IsTgWebSource(task.Item.Source) {
		if domain := tgweb.SiteDomainFromSource(task.Item.Source); domain != "" {
			return sink.LeadHashIDWithScope(domain, contacts)
		}
	}
	return sink.LeadHashIDFromExtract(contacts)
}

func (p *Processor) icpClassifierEnabled(source string) bool {
	if p.ICP == nil {
		return false
	}
	if p.ICPEnabled {
		return true
	}
	// When gemini defer disables inline ICP, tgweb still classifies via ICPTgWebEnabled.
	return p.ICPTgWebEnabled && filter.IsTgWebSource(source)
}

func entitySightingInput(task Task, contacts []extract.Contact, hashID string, matched, stack []string, text string, score int, resolve entity.ResolveInput) entity.SightingInput {
	if resolve.Source == "" {
		resolve.Source = task.Item.Source
	}
	resolve = entity.EnrichForumIdentity(resolve, task.Item.Username, task.Item.Title, task.Item.ForumUserID)
	resolve.Contacts = contacts
	return entity.SightingInput{
		ResolveInput: resolve,
		HashID:       hashID,
		Matched:      matched,
		Stack:        stack,
		Text:         text,
		Score:        score,
		PostedAt:     task.Item.PostedAt,
		SeenAt:       time.Now().UTC(),
	}
}

func (p *Processor) recordEntitySighting(ctx context.Context, in entity.SightingInput) entity.RecordResult {
	if !p.EntitySightings || p.EntityRecorder == nil {
		return entity.RecordResult{}
	}
	result, err := p.EntityRecorder.RecordSighting(ctx, in)
	if err != nil {
		slog.Warn("entity sighting failed",
			"hash_id", in.HashID,
			"source", in.Source,
			"error", err,
		)
		return entity.RecordResult{}
	}
	if result.EntityID == "" {
		return result
	}
	if result.NewEntity {
		slog.Debug("entity created",
			"entity_id", result.EntityID,
			"hash_id", in.HashID,
			"source", in.Source,
		)
	} else if result.NewSourceFamily {
		slog.Info("entity cross-source sighting",
			"entity_id", result.EntityID,
			"hash_id", in.HashID,
			"source", in.Source,
			"source_count", result.SourceCount,
			"sighting_count", result.SightingCount,
		)
	} else if result.CrossSourceHot {
		slog.Info("entity cross-source hot",
			"entity_id", result.EntityID,
			"hash_id", in.HashID,
			"source", in.Source,
			"source_count", result.SourceCount,
			"sighting_count", result.SightingCount,
		)
	} else {
		slog.Debug("entity sighting recorded",
			"entity_id", result.EntityID,
			"hash_id", in.HashID,
			"source", in.Source,
			"sighting_count", result.SightingCount,
		)
	}
	p.maybeCaptureEntityClassify(result)
	return result
}

func (p *Processor) maybeCaptureEntityClassify(result entity.RecordResult) {
	if !p.EntityClassifyEnabled || p.EntityClassify == nil {
		return
	}
	if !entity.ShouldTriggerEntityClassify(result) {
		return
	}
	p.EntityClassify.TryCapture(warmpath.EntityClassifyEvent{
		EntityID:   result.EntityID,
		ForceFresh: result.NewSourceFamily,
	})
}

func (p *Processor) maybePatchCanonicalLead(ctx context.Context, currentHash string, result entity.RecordResult) {
	canonical := strings.TrimSpace(result.CanonicalHash)
	if canonical == "" || canonical == currentHash || p.Store == nil {
		return
	}
	if p.EntityHeatEnabled {
		if entity.HeatTierRank(result.HeatTier) < entity.HeatTierRank(entity.HeatTierHot) {
			return
		}
		patcher, ok := p.Store.(sink.EntityHeatPatcher)
		if !ok {
			return
		}
		boost := entity.HeatBoostForTier(result.HeatTier, p.EntityHeat)
		if err := patcher.ApplyEntityHeat(ctx, canonical, sink.EntityHeatPatch{
			HeatScore:     result.HeatScore,
			HeatTier:      result.HeatTier,
			SightingCount: result.SightingCount,
			SourceCount:   result.SourceCount,
			Boost:         boost,
		}); err != nil {
			slog.Warn("entity heat patch failed",
				"entity_id", result.EntityID,
				"canonical_hash", canonical,
				"error", err,
			)
			return
		}
		slog.Info("entity heat patched canonical lead",
			"entity_id", result.EntityID,
			"canonical_hash", canonical,
			"heat_tier", result.HeatTier,
		)
		return
	}
	if !p.CrossSourceHot || !result.CrossSourceHot {
		return
	}
	patcher, ok := p.Store.(sink.CrossSourceHotPatcher)
	if !ok {
		return
	}
	if err := patcher.ApplyCrossSourceHot(ctx, canonical, p.CrossSourceBoost); err != nil {
		slog.Warn("cross-source hot patch failed",
			"entity_id", result.EntityID,
			"canonical_hash", canonical,
			"error", err,
		)
		return
	}
	slog.Info("cross-source hot patched canonical lead",
		"entity_id", result.EntityID,
		"canonical_hash", canonical,
	)
}

func logTgWebReject(task Task, msg string, extra []any) {
	// Surface tgweb reject reasons at INFO for crawl tuning; keep other sources at DEBUG.
	if !filter.IsTgWebSource(task.Item.Source) {
		slog.Debug(msg, tgWebLogArgs(task, extra)...)
		return
	}
	slog.Info(msg, tgWebLogArgs(task, extra)...)
}

func logTgWebInfo(task Task, msg string, extra ...any) {
	if !filter.IsTgWebSource(task.Item.Source) {
		slog.Debug(msg, tgWebLogArgs(task, extra)...)
		return
	}
	slog.Info(msg, tgWebLogArgs(task, extra)...)
}

func (p *Processor) acquireHash(hashID string) bool {
	if hashID == "" {
		return true
	}
	_, loaded := p.hashInflight.LoadOrStore(hashID, struct{}{})
	return !loaded
}

func (p *Processor) releaseHash(hashID string) {
	if hashID == "" {
		return
	}
	p.hashInflight.Delete(hashID)
}

func tgWebLogArgs(task Task, extra []any) []any {
	args := []any{
		"round_id", task.RoundID,
		"source", task.Item.Source,
		"contact", task.Item.MaskedContact(),
		"snippet_preview", diag.Preview(task.Item.Raw, 200),
	}
	return append(args, extra...)
}

const geoClassifyAboutMax = 500

// geoClassifyText prepends Telegram channel about (from sidecar) for Gemini geo when present.
func geoClassifyText(item model.RawItem, text string) string {
	about := strings.TrimSpace(item.ChannelAbout)
	if about == "" {
		return text
	}
	if len(about) > geoClassifyAboutMax {
		about = about[:geoClassifyAboutMax]
	}
	return "Channel about: " + about + "\n\n" + text
}
