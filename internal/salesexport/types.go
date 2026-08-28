package salesexport

import "time"

// LeadsBundleRU is a sales-facing export of accepted leads with Gemini outreach fields.
type LeadsBundleRU struct {
	Title       string       `json:"заголовок"`
	GeneratedAt string       `json:"сгенерировано"`
	LeadCount   int          `json:"количество_лидов"`
	Leads       []LeadCardRU `json:"лиды"`
	Note        string       `json:"примечание,omitempty"`
}

// LeadCardRU is one lead card for a non-technical sales manager.
type LeadCardRU struct {
	HashID          string   `json:"id_лида"`
	Priority        string   `json:"приоритет"`
	Score           int      `json:"балл"`
	HeatTier        string   `json:"уровень_интереса,omitempty"`
	Source          string   `json:"источник"`
	PostedAt        string   `json:"дата_публикации,omitempty"`
	Contacts        []string `json:"контакты"`
	MatchedKeywords []string `json:"ключевые_фразы"`
	ContextSnippet  string   `json:"контекст"`
	ICP             string   `json:"тип_клиента,omitempty"`
	ICPExplanation  string   `json:"почему_подходит,omitempty"`
	GeoCountry      string   `json:"страна,omitempty"`
	CompanyType     string   `json:"тип_компании,omitempty"`
	EnrichSummary   string   `json:"профиль_клиента,omitempty"`
	PilotQualified  bool     `json:"подходит_для_пилота"`
	PilotWhy        string   `json:"почему_пилот,omitempty"`
	PilotSignals    []string `json:"сигналы_пилота,omitempty"`
	OutreachChannel string   `json:"канал_связи,omitempty"`
	NextAction      string   `json:"следующее_действие,omitempty"`
	OutreachSubject string   `json:"тема_письма,omitempty"`
	OutreachAngle   string   `json:"угол_подхода,omitempty"`
	OutreachDraft   string   `json:"текст_сообщения,omitempty"`
	EntityProof     string   `json:"доказательства_сущности,omitempty"`
	Outcome         string   `json:"результат_crm,omitempty"`
	OutcomeNote     string   `json:"комментарий_результата,omitempty"`
	Tags            []string `json:"теги,omitempty"`
}

// JunkReportRU is a Russian cold-path tuning report for operators/sales context.
type JunkReportRU struct {
	Title                   string         `json:"заголовок"`
	PeriodFrom              string         `json:"период_с"`
	PeriodTo                string         `json:"период_до"`
	SampleCount             int            `json:"количество_отклонений"`
	Summary                 string         `json:"краткое_резюме"`
	TopReasons              []ReasonRowRU  `json:"топ_причин"`
	FalseNegativeCandidates int            `json:"возможные_пропуски"`
	Recommendations         []string       `json:"рекомендации"`
	KeywordSuggestions      []string       `json:"идеи_ключевых_фраз,omitempty"`
	SourceStats             []SourceStatRU `json:"источники,omitempty"`
}

type ReasonRowRU struct {
	Reason string `json:"причина"`
	Count  int    `json:"количество"`
	Why    string `json:"пояснение,omitempty"`
}

type SourceStatRU struct {
	Source string `json:"источник"`
	Count  int    `json:"количество"`
}

// FeedbackBundleRU wraps discover feedback artifacts for sales review.
type FeedbackBundleRU struct {
	Title       string             `json:"заголовок"`
	GeneratedAt string             `json:"сгенерировано"`
	DorkRows    []DorkOutcomeRowRU `json:"поисковые_запросы,omitempty"`
	KeywordRows []KeywordTuneRowRU `json:"ключевые_слова,omitempty"`
}

type DorkOutcomeRowRU struct {
	Query            string  `json:"запрос"`
	Accepted         int     `json:"принято"`
	Junk             int     `json:"отклонено"`
	AcceptRatePct    float64 `json:"доля_принятых_процент"`
	OutcomeContacted int64   `json:"связались"`
	OutcomeReplied   int64   `json:"ответили"`
	OutcomePilot     int64   `json:"пилоты"`
	OutcomeMigration int64   `json:"миграции"`
}

type KeywordTuneRowRU struct {
	KeywordID       string  `json:"id_фразы"`
	Accepted        int     `json:"принято"`
	Junk            int     `json:"отклонено"`
	JunkRatePct     float64 `json:"доля_мусора_процент"`
	SuggestedWeight int     `json:"рекомендуемый_вес,omitempty"`
	RecommendOff    bool    `json:"рекомендуется_отключить"`
}

func FormatTimeUTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
