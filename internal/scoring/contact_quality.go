package scoring

import (
	"strings"
)

const (
	ContactQualityNamed       = "named"
	ContactQualityRoleAccount = "role_account"
	ContactQualityGeneric     = "generic"
)

var roleLocalParts = map[string]struct{}{
	"info": {}, "support": {}, "sales": {}, "contact": {}, "hello": {},
	"admin": {}, "help": {}, "team": {}, "office": {}, "noreply": {},
	"no-reply": {}, "marketing": {}, "billing": {}, "accounts": {},
}

// ScoreContactQuality classifies masked contact lines for CRM inbox sort.
func ScoreContactQuality(contacts []string) string {
	if len(contacts) == 0 {
		return ContactQualityGeneric
	}
	for _, line := range contacts {
		line = strings.TrimSpace(strings.ToLower(line))
		if strings.HasPrefix(line, "email:") {
			if q := scoreEmailLocal(strings.TrimPrefix(line, "email:")); q != "" {
				return q
			}
		}
		if strings.HasPrefix(line, "telegram:") || strings.HasPrefix(line, "@") {
			return ContactQualityNamed
		}
	}
	return ContactQualityGeneric
}

func scoreEmailLocal(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return ContactQualityGeneric
	}
	local := email[:at]
	if local == "" {
		return ContactQualityGeneric
	}
	if _, ok := roleLocalParts[local]; ok {
		return ContactQualityRoleAccount
	}
	if strings.Contains(local, ".") || strings.Contains(local, "_") || len(local) >= 4 {
		return ContactQualityNamed
	}
	return ContactQualityGeneric
}
