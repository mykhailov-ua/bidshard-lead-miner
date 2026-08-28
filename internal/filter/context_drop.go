package filter

import (
	"regexp"
	"strings"
)

var (
	jobContextRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:we are|we're|i am|i'm)\s+hiring\b`),
		regexp.MustCompile(`(?i)\bhiring\s+(?:a\s+)?(?:media\s+buyer|affiliate\s+manager|developer)`),
		regexp.MustCompile(`(?i)\bjob\s+offer\b`),
		regexp.MustCompile(`(?i)\b(?:open|job)\s+position\b`),
		regexp.MustCompile(`(?i)\brecruiting\s+(?:for|a)\b`),
		regexp.MustCompile(`(?i)\bвакансия\b`),
		regexp.MustCompile(`(?i)\bищем\s+(?:медиабайера|media\s+buyer|разработчик)`),
		regexp.MustCompile(`(?i)\bнабор\s+в\s+команду\b`),
	}
	tutorialContextRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bhow\s+to\s+(?:build|create|make|start)\b`),
		regexp.MustCompile(`(?i)\b(?:step\s+by\s+step|tutorial|guide)\b`),
		regexp.MustCompile(`(?i)\b(?:manual|documentation)\s+for\b`),
		regexp.MustCompile(`(?i)\bкак\s+(?:сделать|создать|запустить)\b`),
		regexp.MustCompile(`(?i)\b(?:гайд|мануал|инструкция)\b`),
		regexp.MustCompile(`(?i)\bпошагов`),
	}
	githubNoiseRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:documentation|readme|contributing)\s+update\b`),
		regexp.MustCompile(`(?i)\bbump\s+version\b`),
		regexp.MustCompile(`(?i)\bmerge\s+pull\s+request\b`),
	}
)

// RejectNonBuyerContext drops job posts, tutorials, programmatic vertical, and github maintenance noise without buyer pain.
func RejectNonBuyerContext(source, text, title string) (bool, string) {
	combined := strings.TrimSpace(title + " " + text)
	if combined == "" {
		return true, "empty context"
	}
	if drop, reason := RejectProgrammaticContext(source, text, title); drop {
		return true, reason
	}
	if matchesAnyPattern(combined, jobContextRe) && !HasCommercialPainIntent(combined) && !HasBuyerQuestionPattern(combined) {
		return true, "job or recruiting context"
	}
	if matchesAnyPattern(combined, tutorialContextRe) && !HasCommercialPainIntent(combined) {
		return true, "tutorial or guide context"
	}
	if IsGitHubSource(source) {
		if matchesAnyPattern(combined, githubNoiseRe) && !HasCommercialPainIntent(combined) {
			return true, "github maintenance noise"
		}
		if matchesAnyPattern(combined, jobContextRe) && !HasCommercialPainIntent(combined) {
			return true, "github job context"
		}
	}
	return false, ""
}

// TelegramChannelBroadcastReject drops broadcast channel posts without dialog structure.
func TelegramChannelBroadcastReject(source, chatType string, replyToMessageID int64, text string) bool {
	if !isTelegramSource(source) {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(chatType))
	if kind == "" || kind == "group" || kind == "supergroup" {
		return false
	}
	if kind != "channel" {
		return false
	}
	if replyToMessageID > 0 {
		return false
	}
	if HasCommercialPainIntent(text) || HasBuyerQuestionPattern(text) {
		return false
	}
	return true
}

func matchesAnyPattern(text string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
