package discover

import (
	"strings"
)

// ProgrammaticDorkDenylist blocks active discover queries aimed at programmatic infra.
var ProgrammaticDorkDenylist = []string{
	"programmatic",
	"openrtb",
	"header bidding",
	"dooh",
	"pdooh",
}

// ViolatingProgrammaticDorks returns queries containing anti-ICP programmatic markers.
func ViolatingProgrammaticDorks(telegram, serp []string) []string {
	var out []string
	for _, q := range append(telegram, serp...) {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		lower := strings.ToLower(q)
		for _, pat := range ProgrammaticDorkDenylist {
			if strings.Contains(lower, pat) {
				out = append(out, q)
				break
			}
		}
	}
	return out
}
