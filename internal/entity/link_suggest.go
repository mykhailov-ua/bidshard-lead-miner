package entity

import "strings"

// LinkSuggestPair is two entities sharing a domain alias with conflicting graph signal.
type LinkSuggestPair struct {
	EntityA      string
	EntityB      string
	SharedDomain string
	PainA        string
	PainB        string
}

// FindLinkSuggestPairs returns entity pairs that may need merge/split review.
func FindLinkSuggestPairs(docs []EntityDoc) []LinkSuggestPair {
	var out []LinkSuggestPair
	out = append(out, findDomainPainPairs(docs)...)
	out = append(out, findIdentityKeyPairs(docs)...)
	return dedupeLinkPairs(out)
}

func findDomainPainPairs(docs []EntityDoc) []LinkSuggestPair {
	if len(docs) < 2 {
		return nil
	}
	byDomain := make(map[string][]EntityDoc)
	for _, doc := range docs {
		for _, key := range doc.AliasKeys {
			key = strings.TrimSpace(key)
			if !strings.HasPrefix(key, KindDomain+":") {
				continue
			}
			domain := strings.TrimPrefix(key, KindDomain+":")
			if domain == "" {
				continue
			}
			byDomain[domain] = append(byDomain[domain], doc)
		}
	}

	var out []LinkSuggestPair
	seen := make(map[string]struct{})
	for domain, group := range byDomain {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				if a.EntityID == b.EntityID {
					continue
				}
				painA := strings.TrimSpace(a.UnifiedPain)
				painB := strings.TrimSpace(b.UnifiedPain)
				if painA == "" || painB == "" || strings.EqualFold(painA, painB) {
					continue
				}
				key := pairKey(a.EntityID, b.EntityID, domain)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, LinkSuggestPair{
					EntityA:      a.EntityID,
					EntityB:      b.EntityID,
					SharedDomain: domain,
					PainA:        painA,
					PainB:        painB,
				})
			}
		}
	}
	return out
}

// findIdentityKeyPairs surfaces forum_user/telegram/email collisions across entity nodes.
func findIdentityKeyPairs(docs []EntityDoc) []LinkSuggestPair {
	if len(docs) < 2 {
		return nil
	}
	byKey := make(map[string][]EntityDoc)
	for _, doc := range docs {
		for _, key := range doc.AliasKeys {
			key = strings.TrimSpace(key)
			if key == "" || strings.HasPrefix(key, KindDomain+":") {
				continue
			}
			if strings.HasPrefix(key, KindForumUser+":") ||
				strings.HasPrefix(key, KindForumUID+":") ||
				strings.HasPrefix(key, KindTelegram+":") ||
				strings.HasPrefix(key, "email:") {
				byKey[key] = append(byKey[key], doc)
			}
		}
	}
	var out []LinkSuggestPair
	seen := make(map[string]struct{})
	for sharedKey, group := range byKey {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				if a.EntityID == b.EntityID {
					continue
				}
				idKey := pairKey(a.EntityID, b.EntityID, sharedKey)
				if _, ok := seen[idKey]; ok {
					continue
				}
				seen[idKey] = struct{}{}
				out = append(out, LinkSuggestPair{
					EntityA:      a.EntityID,
					EntityB:      b.EntityID,
					SharedDomain: sharedKey,
					PainA:        strings.TrimSpace(a.UnifiedPain),
					PainB:        strings.TrimSpace(b.UnifiedPain),
				})
			}
		}
	}
	return out
}

func dedupeLinkPairs(pairs []LinkSuggestPair) []LinkSuggestPair {
	seen := make(map[string]struct{}, len(pairs))
	var out []LinkSuggestPair
	for _, p := range pairs {
		key := pairKey(p.EntityA, p.EntityB, p.SharedDomain)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

func pairKey(a, b, domain string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b + "|" + domain
}
