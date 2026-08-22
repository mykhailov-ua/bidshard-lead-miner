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

// FindLinkSuggestPairs returns domain-shared entity pairs with differing unified_pain.
func FindLinkSuggestPairs(docs []EntityDoc) []LinkSuggestPair {
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

func pairKey(a, b, domain string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b + "|" + domain
}
