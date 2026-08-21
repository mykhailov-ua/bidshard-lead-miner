package coldpath

import (
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// Reject reasons written by the hot-path processor.
const (
	ReasonGeoReject       = "geo_reject"
	ReasonGeoGeminiReject = "geo_gemini_reject"
	ReasonKeywordPrescan  = "keyword_prescan"
	ReasonContactReject   = "contact_reject"
	ReasonNoContacts      = "no_contacts"
	ReasonLowScore        = "low_score"
	ReasonRoleEmail       = "role_email"
	ReasonEmptyHash       = "empty_hash"
	ReasonMXReject        = "mx_reject"
	ReasonHardReject      = "hard_reject"
	ReasonTelegramSpam    = "telegram_spam"
	ReasonEmailNoContext  = "email_no_context"
	ReasonLangReject      = "lang_reject"
	ReasonICPReject       = "icp_reject"
	ReasonBlacklist       = "blacklist"
	ReasonEmbedSpam       = "embed_spam"
	ReasonSemanticDedup   = "semantic_dedup"
)

// Event is a rejected raw item captured on the cold path (non-blocking).
type Event struct {
	TS           time.Time
	RoundID      string
	Source       string
	Title        string
	Snippet      string
	ContactHint  string
	Reason       string
	ReasonDetail string
	Score        int
	Matched      []string
}

func JunkDocFromEvent(ev Event) sink.JunkDoc {
	ts := ev.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return sink.JunkDoc{
		TS:           ts,
		RoundID:      ev.RoundID,
		Source:       ev.Source,
		Title:        ev.Title,
		Snippet:      ev.Snippet,
		ContactHint:  ev.ContactHint,
		Reason:       ev.Reason,
		ReasonDetail: ev.ReasonDetail,
		Score:        ev.Score,
		Matched:      append([]string(nil), ev.Matched...),
	}
}
