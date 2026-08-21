package supply

import (
	"encoding/json"
	"strconv"
	"strings"
)

type AdsTxtLine struct {
	Domain   string
	PubID    string
	Relation string
	CertID   string
	Comment  string
}

func ParseAdsTxtLine(line string) (AdsTxtLine, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return AdsTxtLine{}, false
	}
	if rest, ok := strings.CutPrefix(line, "#"); ok {
		return AdsTxtLine{Comment: strings.TrimSpace(rest)}, false
	}

	comment := ""
	if idx := strings.Index(line, "#"); idx >= 0 {
		comment = strings.TrimSpace(line[idx+1:])
		line = strings.TrimSpace(line[:idx])
	}

	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return AdsTxtLine{Comment: comment}, false
	}

	entry := AdsTxtLine{
		Domain:   strings.TrimSpace(parts[0]),
		PubID:    strings.TrimSpace(parts[1]),
		Relation: strings.ToUpper(strings.TrimSpace(parts[2])),
		Comment:  comment,
	}
	if len(parts) > 3 {
		entry.CertID = strings.TrimSpace(parts[3])
	}
	if entry.Domain == "" || entry.PubID == "" {
		return AdsTxtLine{}, false
	}
	if entry.Relation != "DIRECT" && entry.Relation != "RESELLER" {
		return AdsTxtLine{}, false
	}
	return entry, true
}

func ParseAdsTxt(body string) []AdsTxtLine {
	var out []AdsTxtLine
	for _, line := range strings.Split(body, "\n") {
		if entry, ok := ParseAdsTxtLine(line); ok {
			out = append(out, entry)
		}
	}
	return out
}

func ExtractContactDirective(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "CONTACT=") || strings.HasPrefix(strings.ToUpper(line), "# CONTACT=") || strings.HasPrefix(strings.ToUpper(line), "#CONTACT=") {
			idx := strings.Index(line, "=")
			if idx >= 0 {
				val := strings.TrimSpace(line[idx+1:])
				if strings.Contains(val, "@") {
					return val
				}
			}
		}
	}
	return ""
}

type SellerContact struct {
	Name         string
	Domain       string
	ContactEmail string
}

type sellersJSON struct {
	ContactEmail string `json:"contact_email"`
	Sellers      []struct {
		Name         string `json:"name"`
		Domain       string `json:"domain"`
		ContactEmail string `json:"contact_email"`
	} `json:"sellers"`
}

func ParseSellersJSON(body []byte) []SellerContact {
	var doc sellersJSON
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil
	}

	var out []SellerContact
	if email := strings.TrimSpace(doc.ContactEmail); email != "" {
		out = append(out, SellerContact{ContactEmail: email})
	}
	for _, s := range doc.Sellers {
		email := strings.TrimSpace(s.ContactEmail)
		if email == "" {
			continue
		}
		out = append(out, SellerContact{
			Name:         s.Name,
			Domain:       s.Domain,
			ContactEmail: email,
		})
	}
	return out
}

func BuildSnippet(domain string, ads []AdsTxtLine, sellers []SellerContact) string {
	return strings.TrimSpace(strings.Join([]string{
		"Supply crawl on " + domain + ".",
		"ads.txt entries: " + strconv.Itoa(len(ads)) + ".",
		"sellers.json contacts: " + strconv.Itoa(len(sellers)) + ".",
	}, " "))
}
