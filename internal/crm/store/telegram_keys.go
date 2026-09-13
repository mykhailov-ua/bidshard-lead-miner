package store

import (
	"strconv"
	"strings"
)

const (
	telegramUserKeyPrefix    = "tg_user:"
	telegramChatKeyPrefix    = "tg_chat:"
	telegramMemberEdgePrefix = "tg_member:"
)

// TelegramPersonKey is the stable graph node id for a Telegram account.
func TelegramPersonKey(userID int64) string {
	if userID <= 0 {
		return ""
	}
	return telegramUserKeyPrefix + strconv.FormatInt(userID, 10)
}

// TelegramChatRef namespaces a crawler chat_key (u:, i:, c:, n:) for cross-system graphs.
func TelegramChatRef(chatKey string) string {
	chatKey = strings.TrimSpace(chatKey)
	if chatKey == "" {
		return ""
	}
	lower := strings.ToLower(chatKey)
	if strings.HasPrefix(lower, telegramChatKeyPrefix) {
		return lower
	}
	return telegramChatKeyPrefix + lower
}

// TelegramChatRefFromUsername builds tg_chat:u:{username}.
func TelegramChatRefFromUsername(username string) string {
	username = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(username), "@"))
	if username == "" {
		return ""
	}
	return TelegramChatRef("u:" + username)
}

// TelegramMemberEdgeKey links a person node to a chat node (undirected registry edge).
func TelegramMemberEdgeKey(chatRef, personKey string) string {
	chatRef = strings.TrimSpace(chatRef)
	personKey = strings.TrimSpace(personKey)
	if chatRef == "" || personKey == "" {
		return ""
	}
	return telegramMemberEdgePrefix + chatRef + "|" + personKey
}

// FillTelegramPersonCanonicalKeys sets person_key and chat refs on export when legacy rows omit them.
func FillTelegramPersonCanonicalKeys(doc *TelegramPersonDoc) {
	if doc == nil {
		return
	}
	if doc.PersonKey == "" && doc.UserID > 0 {
		doc.PersonKey = TelegramPersonKey(doc.UserID)
	}
	if doc.SourceChatRef == "" && doc.SourceChat != "" {
		doc.SourceChatRef = TelegramChatRefFromUsername(doc.SourceChat)
	}
	if len(doc.MemberChatRefs) == 0 && len(doc.MemberChats) > 0 {
		refs := make([]string, 0, len(doc.MemberChats))
		seen := make(map[string]struct{}, len(doc.MemberChats))
		for _, chat := range doc.MemberChats {
			ref := TelegramChatRefFromUsername(chat)
			if ref == "" {
				continue
			}
			if _, ok := seen[ref]; ok {
				continue
			}
			seen[ref] = struct{}{}
			refs = append(refs, ref)
		}
		doc.MemberChatRefs = refs
	}
	if len(doc.MemberEdges) == 0 && doc.PersonKey != "" && len(doc.MemberChatRefs) > 0 {
		edges := make([]string, 0, len(doc.MemberChatRefs))
		for _, ref := range doc.MemberChatRefs {
			if edge := TelegramMemberEdgeKey(ref, doc.PersonKey); edge != "" {
				edges = append(edges, edge)
			}
		}
		doc.MemberEdges = edges
	}
}
