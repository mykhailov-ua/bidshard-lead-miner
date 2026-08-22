package warmpath

import "time"

// Event is a deferred accepted lead queued for batch Gemini analysis.
type Event struct {
	HashID        string
	RoundID       string
	Source        string
	Title         string
	Snippet       string
	Contacts      []string
	ContactTypes  []string
	Stack         []string
	Score         int
	Priority      string
	Matched       []string
	Domain        string
	RDAPCountry   string
	DomainAgeDays int
	DisplayName   string
	EntityID      string
	EntityHeat    float64
	HeatTier      string
	InlineICP       string
	TS              time.Time
}
