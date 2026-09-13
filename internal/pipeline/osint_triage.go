package pipeline

import (
	"strings"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
)

func adsTxtPublisherContactTriage(lead *model.Lead, text string, contacts []extract.Contact) {
	if lead == nil {
		return
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(lead.Source)), "ads_txt:") {
		return
	}
	upper := strings.ToUpper(text)
	if !strings.Contains(upper, "CONTACT=") {
		return
	}
	for _, c := range contacts {
		if c.Type == "email" && strings.TrimSpace(c.Value) != "" {
			lead.Tags = entity.AppendUniqueTag(lead.Tags, scoring.TagPublisherSurface)
			lead.Tags = entity.AppendUniqueTag(lead.Tags, scoring.TagPublisherAdsTxtContact)
			lead.Tags = entity.AppendUniqueTag(lead.Tags, "outreach-email")
			return
		}
	}
}

func applySourceFamilyTags(lead *model.Lead) {
	if lead == nil {
		return
	}
	if filter.IsDiscordSource(lead.Source) {
		lead.Tags = entity.AppendUniqueTag(lead.Tags, scoring.TagDiscordCommunityIntel)
	}
}
