package sink

import (
	"strings"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type LeadDoc struct {
	HashID   string          `bson:"hash_id" json:"hash_id"`
	TS       time.Time       `bson:"ts" json:"ts"`
	RoundID  string          `bson:"round_id" json:"round_id"`
	Priority string          `bson:"priority" json:"priority"`
	Score    int             `bson:"score" json:"score"`
	Source   string          `bson:"source" json:"source"`
	Title    string          `bson:"title" json:"title"`
	Contacts []StoredContact `bson:"contacts" json:"contacts"`
	Matched  []string        `bson:"matched" json:"matched"`
	Snippet  string          `bson:"snippet" json:"snippet"`
}

func ToLeadDoc(lead model.Lead) LeadDoc {
	contacts := ToStoredContacts(ParseFormattedContacts(lead.Contacts))
	ts := lead.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	snippet := strings.Join(strings.Fields(lead.Snippet), " ")
	if len(snippet) > 500 {
		snippet = snippet[:497] + "..."
	}
	return LeadDoc{
		HashID:   lead.HashID,
		TS:       ts,
		RoundID:  lead.RoundID,
		Priority: lead.Priority,
		Score:    lead.Score,
		Source:   lead.Source,
		Title:    lead.Title,
		Contacts: contacts,
		Matched:  append([]string(nil), lead.Matched...),
		Snippet:  snippet,
	}
}

func LeadJSONMap(lead model.Lead) map[string]any {
	doc := ToLeadDoc(lead)
	return map[string]any{
		"hash_id":  doc.HashID,
		"ts":       doc.TS.Format(time.RFC3339),
		"round_id": doc.RoundID,
		"priority": doc.Priority,
		"score":    doc.Score,
		"source":   doc.Source,
		"title":    doc.Title,
		"contacts": doc.Contacts,
		"matched":  doc.Matched,
		"snippet":  doc.Snippet,
	}
}
