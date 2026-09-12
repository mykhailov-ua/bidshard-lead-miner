package filter

import "strings"

// instantDropPhrases: seller consumables, design shops, scam-worker recruitment.
// Instant drop before scoring; buyer pain in the same message is not exempt.
var instantDropPhrases = []string{
	// Agency accounts / farm / FP / cards (EN)
	"agency account", "agency accounts", "ad account for sale", "accounts for sale",
	"buy fp", "buy fb account", "buy facebook account", "facebook account rent",
	"farm account", "farm accounts", "account farm", "farming accounts",
	"virtual card", "virtual cards", "vcc for ads", "virtual cc",
	"drop service", "drops available", "aged accounts", "warmed bm", "warmed bms",
	"warm bm", "warm bms", "business manager rent", "bm for rent",
	"first bill account", "accounts with first bill", "billing ready account",
	// Agency accounts / farm / FP / cards (RU)
	"агентские аккаунты", "агентский аккаунт", "купить фп", "купить фб",
	"фарм аккаунт", "фарм аккаунты", "фарм акков", "фарм акк",
	"виртуальные карты", "виртуалки", "виртуальная карта",
	"дропы", "дроп сервис", "гретые бмы", "гретые bm", "гретый бм",
	"акки под первобил", "аккаунты под первобил", "под первобил",
	"аренда аккаунтов", "продажа аккаунтов", "аккаунты в наличии",
	// Design / content services (EN)
	"creatives on order", "custom creatives", "creative design service",
	"landing page design", "landing design service", "lander design order",
	"reels editing", "reel editing", "video editing service",
	"voiceover service", "video voiceover",
	// Design / content services (RU)
	"креативы на заказ", "дизайн лендингов", "дизайн лендинга на заказ",
	"монтаж reels", "монтаж рилс", "озвучка видео", "озвучка на заказ",
	"лендинг на заказ", "креативы под ключ",
	// Scam workers / course funnels (EN)
	"join our team earn", "looking for workers", "we need workers",
	"training from scratch", "earn from scratch", "earning scheme",
	"profit share scheme", "percent of profit", "% of profit",
	"passive income scheme", "work from phone",
	// Scam workers (RU)
	"набор в тиму", "набор в команду", "ищем воркеров", "нужны воркеры",
	"обучение с нуля", "схема заработка", "процент с профита", "% с профита",
	"заработок с нуля", "пассивный заработок", "работа с телефона",
}

// InstantDropSellerSpam reports consumables sellers and scam-worker outreach.
func InstantDropSellerSpam(text string) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false, ""
	}
	for _, phrase := range instantDropPhrases {
		if strings.Contains(lower, phrase) {
			return true, "seller spam: " + phrase
		}
	}
	return false, ""
}
