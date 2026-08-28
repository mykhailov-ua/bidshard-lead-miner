package scoring

import (
	"strings"
)

const (
	ContactChannelTelegram = "telegram"
	ContactChannelEmail    = "email"
	ContactChannelSkype    = "skype"
	ContactChannelForum    = "forum"
	ContactChannelReddit   = "reddit"
	ContactChannelGitHub   = "github"
	ContactChannelOther    = "other"

	NextActionTelegramDM      = "telegram_dm"
	NextActionColdEmail       = "cold_email"
	NextActionSkypeMessage    = "skype_message"
	NextActionForumManual     = "forum_manual"
	NextActionRedditDM        = "reddit_dm"
	NextActionGitHubReach     = "github_reach"
	NextActionResearchContact = "research_contact"
)

var contactChannelOrder = []string{
	ContactChannelTelegram,
	ContactChannelEmail,
	ContactChannelSkype,
	ContactChannelForum,
	ContactChannelReddit,
	ContactChannelGitHub,
}

// ParseContactChannels returns deduped outreach channels present in formatted contact lines.
func ParseContactChannels(contacts []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, line := range contacts {
		ch := channelFromContactLine(line)
		if ch == "" {
			continue
		}
		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		out = append(out, ch)
	}
	return out
}

func channelFromContactLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "telegram:") || strings.HasPrefix(line, "@"):
		return ContactChannelTelegram
	case strings.HasPrefix(lower, "forum:user/"), strings.HasPrefix(lower, "warrior:user/"):
		return ContactChannelForum
	case strings.HasPrefix(lower, "reddit:"):
		return ContactChannelReddit
	case strings.HasPrefix(lower, "github:"):
		return ContactChannelGitHub
	case strings.HasPrefix(lower, "skype:"):
		return ContactChannelSkype
	case strings.HasPrefix(lower, "email:"):
		return ContactChannelEmail
	}
	if strings.Contains(line, "@") {
		return ContactChannelEmail
	}
	return ""
}

// PickContactChannel chooses the CRM queue bucket (Gemini outreach hint when compatible).
func PickContactChannel(available []string, outreachChannel string) string {
	if len(available) == 0 {
		return ContactChannelOther
	}
	pref := normalizeOutreachChannel(outreachChannel)
	if pref != "" && containsChannel(available, pref) {
		return pref
	}
	for _, ch := range contactChannelOrder {
		if containsChannel(available, ch) {
			return ch
		}
	}
	return ContactChannelOther
}

func normalizeOutreachChannel(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case ContactChannelTelegram, ContactChannelEmail, ContactChannelForum,
		ContactChannelReddit, ContactChannelGitHub, ContactChannelSkype:
		return raw
	case "other":
		return ""
	default:
		return ""
	}
}

func containsChannel(channels []string, want string) bool {
	for _, ch := range channels {
		if ch == want {
			return true
		}
	}
	return false
}

// NextActionForChannel maps a contact channel to a human outreach step.
func NextActionForChannel(channel string) string {
	switch channel {
	case ContactChannelTelegram:
		return NextActionTelegramDM
	case ContactChannelEmail:
		return NextActionColdEmail
	case ContactChannelSkype:
		return NextActionSkypeMessage
	case ContactChannelForum:
		return NextActionForumManual
	case ContactChannelReddit:
		return NextActionRedditDM
	case ContactChannelGitHub:
		return NextActionGitHubReach
	default:
		return NextActionResearchContact
	}
}

// AssignOutreachQueue derives CRM contact_channel and next_action from masked contacts.
func AssignOutreachQueue(contacts []string, outreachChannel string) (contactChannel, nextAction string) {
	avail := ParseContactChannels(contacts)
	contactChannel = PickContactChannel(avail, outreachChannel)
	nextAction = NextActionForChannel(contactChannel)
	return contactChannel, nextAction
}

// EngagePriorityInput feeds outreach queue ordering (higher = sooner CRM action).
type EngagePriorityInput struct {
	Priority         Priority
	Score            int
	DisplacementTier DisplacementTier
	PilotQualified   bool
	Stack            []string
	EntityHeat       float64
	ContactQuality   string
	HasEngageDraft   bool
	SourceFamily     string
}

// ComputeEngagePriority ranks accepted leads for sales outreach, independent of accept gates.
func ComputeEngagePriority(in EngagePriorityInput) int {
	priority := 0
	switch in.Priority {
	case PriorityHigh:
		priority += 100
	case PriorityMedium:
		priority += 50
	default:
		priority += 10
	}
	if in.Score > 40 {
		priority += 40
	} else if in.Score > 0 {
		priority += in.Score
	}
	switch in.DisplacementTier {
	case DisplacementHot:
		priority += 30
	case DisplacementWarm:
		priority += 15
	}
	if len(in.Stack) > 0 {
		priority += 10
	}
	if in.PilotQualified {
		priority += 20
	}
	if in.HasEngageDraft {
		priority += 15
	}
	if in.EntityHeat > 0 {
		priority += int(in.EntityHeat * 20)
	}
	switch in.ContactQuality {
	case "verified":
		priority += 10
	case "role":
		priority -= 10
	}
	switch {
	case strings.HasPrefix(strings.ToLower(in.SourceFamily), "reddit:"):
		priority += 8
	case strings.HasPrefix(strings.ToLower(in.SourceFamily), "forum:"):
		priority += 6
	}
	if priority < 0 {
		return 0
	}
	return priority
}

// ChannelsFromContactTypes maps extract contact types to outreach channel buckets.
func ChannelsFromContactTypes(types []string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(ch string) {
		if ch == "" {
			return
		}
		if _, ok := seen[ch]; ok {
			return
		}
		seen[ch] = struct{}{}
		out = append(out, ch)
	}
	for _, t := range types {
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "email":
			add(ContactChannelEmail)
		case "telegram":
			add(ContactChannelTelegram)
		case "forum_user":
			add(ContactChannelForum)
		case "reddit":
			add(ContactChannelReddit)
		case "github":
			add(ContactChannelGitHub)
		case "skype":
			add(ContactChannelSkype)
		}
	}
	return out
}

// SourcePrefersEmailOutreach nudges B2B site/supply leads toward cold email when email exists.
func SourcePrefersEmailOutreach(source string, channels []string) bool {
	if !containsChannel(channels, ContactChannelEmail) {
		return false
	}
	source = strings.ToLower(strings.TrimSpace(source))
	return strings.HasPrefix(source, "ads_txt:") ||
		strings.HasPrefix(source, "lander:") ||
		strings.HasPrefix(source, "tgweb:") ||
		strings.HasPrefix(source, "supply:")
}
