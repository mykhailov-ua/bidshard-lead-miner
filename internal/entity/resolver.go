package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/tgweb"
)

// ResolveInput carries lead fields used for entity key extraction.
type ResolveInput struct {
	CompanyName  string
	DisplayName  string
	GravatarName string
	Source       string
	ForumUser    string
	ForumUID     string
	Contacts     []extract.Contact
}

// ResolveInputFromLead builds resolver input from an accepted lead and extracted contacts.
func ResolveInputFromLead(lead model.Lead, contacts []extract.Contact) ResolveInput {
	return ResolveInput{
		CompanyName:  lead.CompanyName,
		DisplayName:  lead.DisplayName,
		GravatarName: lead.GravatarName,
		Source:       lead.Source,
		Contacts:     contacts,
	}
}

// ResolveKeys extracts normalized identity keys for cross-source entity linking.
// Keys are deduplicated and sorted strongest-first.
func ResolveKeys(in ResolveInput) []EntityKey {
	var keys []EntityKey

	if company := NormalizeCompany(in.CompanyName); company != "" {
		keys = append(keys, EntityKey{Kind: KindCompany, Value: company})
	}
	if in.CompanyName == "" {
		// Fall back to display/gravatar names only when they look like orgs, not personal names.
		for _, candidate := range []string{in.GravatarName, in.DisplayName} {
			if company := NormalizeCompany(candidate); company != "" && isOrgLikeName(candidate) {
				keys = append(keys, EntityKey{Kind: KindCompany, Value: company})
				break
			}
		}
	}
	if filter.IsTelegramSource(in.Source) {
		if channel := TelegramChannelFromSource(in.Source); channel != "" {
			keys = append(keys, EntityKey{Kind: KindTelegramChannel, Value: channel})
		}
	}
	if filter.IsTgWebSource(in.Source) {
		// Site domain from tgweb:@channel:example.com label is the primary cross-source key for affiliate crawls.
		if domain := NormalizeDomain(tgweb.SiteDomainFromSource(in.Source)); domain != "" {
			keys = append(keys, EntityKey{Kind: KindDomain, Value: domain})
		}
	}
	if domain := SupplyDomainFromSource(in.Source); domain != "" {
		// Supply crawler labels items ads_txt:{host}; domain key links publisher surface to tgweb/email.
		keys = append(keys, EntityKey{Kind: KindDomain, Value: domain})
	}
	keys = appendForumIdentityKeys(keys, in)

	for _, c := range in.Contacts {
		switch c.Type {
		case "email":
			if domain := EmailRootDomain(c.Value); domain != "" {
				keys = append(keys, EntityKey{Kind: KindDomain, Value: domain})
			}
		case "domain":
			if domain := NormalizeDomain(c.Value); domain != "" {
				keys = append(keys, EntityKey{Kind: KindDomain, Value: domain})
			}
		case "telegram":
			if handle := NormalizeTelegram(c.Value); handle != "" {
				keys = append(keys, EntityKey{Kind: KindTelegram, Value: handle})
			}
		}
	}

	keys = dedupeKeys(keys)
	sortKeys(keys)
	return keys
}

// SupplyDomainFromSource returns the publisher domain from ads_txt:{domain} source labels.
// Matches internal/sources/supply Crawler emit prefix, not a generic supply: family string.
func SupplyDomainFromSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	const prefix = "ads_txt:"
	if !strings.HasPrefix(source, prefix) {
		return ""
	}
	return NormalizeDomain(strings.TrimPrefix(source, prefix))
}

// PrimaryKey returns the strongest identity key or false when none were resolved.
func PrimaryKey(keys []EntityKey) (EntityKey, bool) {
	if len(keys) == 0 {
		return EntityKey{}, false
	}
	return keys[0], true
}

// EntityID returns a stable 32-char hex id from the primary entity key.
// Hash the strongest key only so secondary aliases do not fork entity IDs.
func EntityID(keys []EntityKey) string {
	pk, ok := PrimaryKey(keys)
	if !ok {
		return ""
	}
	return entityIDFromPrimary(pk)
}

// EntityIDForSplit returns the detached graph node id after ops split.
// When keys still map to sourceEntityID (shared-domain false merge), fork from
// primary+hash_id so InsertOne does not collide with the parent Mongo row.
func EntityIDForSplit(keys []EntityKey, hashID, sourceEntityID string) string {
	base := EntityID(keys)
	if base == "" {
		return ""
	}
	if base != strings.TrimSpace(sourceEntityID) {
		return base
	}
	pk, ok := PrimaryKey(keys)
	if !ok {
		return base
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return base
	}
	return entityIDFromPrimaryWithSuffix(pk, ":split:"+hashID)
}

func entityIDFromPrimary(pk EntityKey) string {
	sum := sha256.Sum256([]byte(pk.Canonical()))
	return hex.EncodeToString(sum[:16])
}

func entityIDFromPrimaryWithSuffix(pk EntityKey, suffix string) string {
	sum := sha256.Sum256([]byte(pk.Canonical() + suffix))
	return hex.EncodeToString(sum[:16])
}

// AliasTokens returns canonical key strings for secondary lookup indexes.
func AliasTokens(keys []EntityKey) []string {
	if len(keys) == 0 {
		return nil
	}
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = k.Canonical()
	}
	return out
}

// SourceFamily groups a source label into a coarse family for cross-source counting.
// tgweb and telegram are distinct families even though both mention Telegram.
//
// Forum graph labels (HEAT-P1-02):
//
//	forum:affiliatefix.com/{thread-slug}  -> forum
//	forum:blackhatworld.com/{thread-slug} -> forum
//	warrior:{thread-slug}                 -> warrior
//	reddit:{subreddit}                    -> reddit
func SourceFamily(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(source, "telegram:"):
		return "telegram"
	case strings.HasPrefix(source, "tgweb:"):
		return "tgweb"
	case strings.HasPrefix(source, "reddit:"):
		return "reddit"
	case strings.HasPrefix(source, "github:"):
		return "github"
	case strings.HasPrefix(source, "discord:"):
		return "discord"
	case strings.HasPrefix(source, "serp:"):
		return "serp"
	case strings.HasPrefix(source, "lander:"):
		return "lander"
	case strings.HasPrefix(source, "warrior:"):
		return "warrior"
	case strings.HasPrefix(source, "reviews:"):
		return "reviews"
	case strings.HasPrefix(source, "ads_txt:"):
		// Distinct from tgweb so cross-source-hot counts publisher vs site crawl separately.
		return "supply"
	case strings.HasPrefix(source, "forum:"):
		return "forum"
	default:
		if idx := strings.Index(source, ":"); idx > 0 {
			return source[:idx]
		}
		return source
	}
}
