package coldpath

import "strings"

func maskContactHint(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "@") {
		return maskEmail(v)
	}
	if strings.HasPrefix(v, "@") {
		if len(v) <= 2 {
			return "@***"
		}
		return v[:2] + "***"
	}
	if len(v) <= 3 {
		return "***"
	}
	return v[:1] + "***"
}

func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local := email[:at]
	domain := email[at+1:]
	if len(local) == 0 {
		return "***@" + domain
	}
	return local[:1] + "***@" + domain
}
