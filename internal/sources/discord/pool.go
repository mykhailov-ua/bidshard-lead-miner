package discord

import (
	"strings"
	"sync/atomic"
)

// TokenPool round-robins bot tokens on rate limits (DISCORD_BOT_TOKENS).
type TokenPool struct {
	tokens []string
	cursor atomic.Uint64
}

func NewTokenPool(tokens []string) *TokenPool {
	clean := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok != "" {
			clean = append(clean, tok)
		}
	}
	return &TokenPool{tokens: clean}
}

func (p *TokenPool) Len() int {
	if p == nil {
		return 0
	}
	return len(p.tokens)
}

func (p *TokenPool) Pick() string {
	if p == nil || len(p.tokens) == 0 {
		return ""
	}
	idx := p.cursor.Load() % uint64(len(p.tokens))
	return p.tokens[idx]
}

func (p *TokenPool) Rotate() {
	if p == nil || len(p.tokens) <= 1 {
		return
	}
	p.cursor.Add(1)
}
