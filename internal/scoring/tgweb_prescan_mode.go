package scoring

import "strings"

// TgWebPrescanMode controls how tgweb leads pass keyword prescan.
type TgWebPrescanMode string

const (
	TgWebPrescanStrict     TgWebPrescanMode = "strict"
	TgWebPrescanAggressive TgWebPrescanMode = "aggressive"
)

// ParseTgWebPrescanMode parses PARSER_TGWEB_PRESCAN_MODE (default aggressive).
func ParseTgWebPrescanMode(raw string) TgWebPrescanMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict", "conservative":
		// "conservative" is an alias for strict keyword prescan on tgweb.
		return TgWebPrescanStrict
	default:
		return TgWebPrescanAggressive
	}
}

func (m TgWebPrescanMode) Aggressive() bool {
	return m == TgWebPrescanAggressive
}
