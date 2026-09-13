package pipeline

import (
	"strings"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
)

func isTelegramCPASupply(item model.RawItem) bool {
	return strings.EqualFold(strings.TrimSpace(item.LeadType), "cpa_supply")
}

// scoreTeamHiringLead uses hiring intent only; no tracker/Keitaro keyword scoring.
func scoreTeamHiringLead(reg *scoring.Registry, title, text string) (*scoring.LeadText, scoring.Priority) {
	_ = strings.TrimSpace(title + " " + text)
	lt := &scoring.LeadText{
		Context: text,
		Title:   title,
		Score:   mediumMinFromReg(reg),
		Matched: []string{"team_hiring"},
	}
	return lt, scoring.PriorityFromScore(reg, lt.Score)
}

func scoreTelegramColdTeamLead(reg *scoring.Registry, title, text string) (*scoring.LeadText, scoring.Priority) {
	return scoreTeamHiringLead(reg, title, text)
}
