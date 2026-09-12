package telegrambot

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/classify"
	"github.com/bidshard/parser/internal/sink"
)

// FormatLeadNotifyHTML renders a compact lead card for Telegram HTML parse mode.
func FormatLeadNotifyHTML(doc sink.LeadDoc) string {
	ts := doc.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	geo := strings.TrimSpace(doc.GeoCountry)
	if geo == "" {
		geo = strings.TrimSpace(doc.CompanyCountry)
	}
	if geo == "" {
		geo = "-"
	}
	keywords := strings.Join(doc.Matched, ", ")
	if len(keywords) > 80 {
		keywords = keywords[:77] + "..."
	}
	if keywords == "" {
		keywords = "-"
	}
	snippet := strings.Join(strings.Fields(doc.Snippet), " ")
	if len(snippet) > 280 {
		snippet = snippet[:277] + "..."
	}
	contact := formatLeadContact(doc.Contacts)
	icp := strings.TrimSpace(doc.ICP)
	if icp == "" {
		icp = "-"
	}
	heat := strings.TrimSpace(doc.HeatTier)
	if heat == "" {
		heat = "-"
	}
	title := strings.TrimSpace(doc.Title)
	if title != "" {
		title = "\nTitle: " + html.EscapeString(title)
	}

	header := "<b>New lead</b>"
	painText := strings.TrimSpace(doc.Snippet)
	if painText == "" {
		painText = strings.TrimSpace(doc.Title)
	}
	if pain := classify.ClassifyBidShardPain(painText); pain != nil {
		header = fmt.Sprintf(
			"<b>[HOT] POTENTIAL CLIENT [%s]</b>",
			html.EscapeString(classify.TierLabel(pain.TierHint)),
		)
	}

	lines := []string{
		header,
		fmt.Sprintf("Score: <b>%d</b> %s | heat=%s", doc.Score, html.EscapeString(doc.Priority), html.EscapeString(heat)),
		fmt.Sprintf("Source: <code>%s</code>", html.EscapeString(doc.Source)),
		fmt.Sprintf("Geo: %s | ICP: %s", html.EscapeString(geo), html.EscapeString(icp)),
		fmt.Sprintf("Contact: %s", html.EscapeString(contact)),
	}
	if pain := classify.ClassifyBidShardPain(painText); pain != nil {
		lines = append(lines, fmt.Sprintf("Pain: %s", html.EscapeString(classify.PainBucketLabel(pain.PainBucket))))
		lines = append(lines, fmt.Sprintf("BidShard angle: %s", html.EscapeString(pain.PitchLine)))
	}
	lines = append(lines,
		fmt.Sprintf("Keywords: %s", html.EscapeString(keywords)),
		fmt.Sprintf("hash: <code>%s</code>", html.EscapeString(doc.HashID)),
		fmt.Sprintf("Time: %s UTC", ts.Format("2006-01-02 15:04")),
	)
	if title != "" {
		lines = append(lines, title)
	}
	if snippet != "" {
		lines = append(lines, "", html.EscapeString(snippet))
	}
	return strings.Join(lines, "\n")
}

func formatLeadContact(contacts []sink.StoredContact) string {
	if len(contacts) == 0 {
		return "-"
	}
	c := contacts[0]
	if c.Type != "" && c.Value != "" {
		return c.Type + ":" + c.Value
	}
	return c.Value
}
