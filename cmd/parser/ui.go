package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/pretty"
	"github.com/bidshard/parser/internal/sources"
)

func cliColor(w io.Writer) bool {
	return !globalOpts.noColor && pretty.ColorEnabled(w)
}

func printDiscoverFeedback(w io.Writer, result app.DiscoverFeedbackResult) {
	color := cliColor(w)
	pretty.Section(w, color, "Discover feedback")
	rows := [][2]string{}
	if result.OutcomeReportPath != "" {
		rows = append(rows, [2]string{"outcome_report", result.OutcomeReportPath})
	}
	if result.KeywordTunePath != "" {
		rows = append(rows, [2]string{"keyword_tune", result.KeywordTunePath})
	}
	if result.KeywordTuneApplied != "" {
		rows = append(rows, [2]string{"keyword_tune_applied", result.KeywordTuneApplied})
	}
	if result.SalesFeedbackPath != "" {
		rows = append(rows, [2]string{"sales_feedback_ru", result.SalesFeedbackPath})
	}
	if result.PrunedDorks > 0 {
		rows = append(rows, [2]string{"pruned_dorks", fmt.Sprintf("%d", result.PrunedDorks)})
	}
	if len(rows) == 0 {
		pretty.StatusNote(w, color, "no changes this run")
		return
	}
	pretty.PrintLabeled(w, color, rows)
	if result.PrunedDorks > 0 {
		pretty.StatusOK(w, color, "disabled %d underperforming SERP dork(s)", result.PrunedDorks)
	}
	if result.KeywordTuneApplied != "" {
		pretty.StatusOK(w, color, "keyword tune applied (%s)", result.KeywordTuneApplied)
	}
}

func printSalesExport(w io.Writer, result app.SalesExportResult) {
	color := cliColor(w)
	pretty.Section(w, color, "Sales export (RU JSON)")
	rows := [][2]string{}
	if result.LeadsPath != "" {
		rows = append(rows, [2]string{"leads_ru", result.LeadsPath})
	}
	if result.JunkPath != "" {
		rows = append(rows, [2]string{"junk_report_ru", result.JunkPath})
	}
	if len(rows) == 0 {
		pretty.StatusNote(w, color, "nothing exported (check Mongo / GEMINI_API_KEY)")
		return
	}
	pretty.PrintLabeled(w, color, rows)
	pretty.StatusOK(w, color, "wrote %d file(s)", len(rows))
}

func printSourcesList(w io.Writer, catalog []sources.SourceInfo) {
	color := cliColor(w)
	pretty.Section(w, color, "Crawl sources")
	header := []string{"source", "all", "requires", "note"}
	var rows [][]string
	for _, info := range catalog {
		inAll := ""
		if info.InAll {
			inAll = "yes"
		}
		req := strings.Join(info.Requires, ", ")
		rows = append(rows, []string{info.Name, inAll, req, info.Note})
	}
	pretty.PrintTable(w, color, header, rows)
	pretty.StatusNote(w, color, "Set: parser scan --source=forum   or   PARSER_SOURCE=forum")
	pretty.StatusNote(w, color, "Telegram sidecar: parser telegram")
}

func printAutoStatus(w io.Writer, st app.AutoStatus) {
	color := cliColor(w)
	pretty.Section(w, color, "Automation status")
	pretty.StatusInfo(w, color, "at %s", st.At.Format("2006-01-02 15:04:05 UTC"))
	if st.AutoDiscover {
		pretty.StatusOK(w, color, "PARSER_AUTO_DISCOVER=true")
	}
	rows := [][2]string{
		{"forum_threads", fmt.Sprintf("%d", st.ForumThreads)},
		{"telegram_domains", fmt.Sprintf("%d", st.TelegramDomains)},
		{"registry_dropped", fmt.Sprintf("%d", st.RegistryDropped)},
		{"headless_queue", fmt.Sprintf("%d", st.HeadlessQueued)},
	}
	if len(st.SourceRegistry) > 0 {
		rows = append(rows, [2]string{"registry_tgweb", fmt.Sprintf("%d", st.SourceRegistry["tgweb"])})
		rows = append(rows, [2]string{"registry_supply", fmt.Sprintf("%d", st.SourceRegistry["supply"])})
		rows = append(rows, [2]string{"registry_lander", fmt.Sprintf("%d", st.SourceRegistry["lander"])})
	}
	if st.ProxyCapMB > 0 {
		rows = append(rows, [2]string{"proxy_budget_mb", fmt.Sprintf("%.1f / %d", st.ProxyUsedMB, st.ProxyCapMB)})
		if st.ProxyBudgetExceeded {
			pretty.StatusWarn(w, color, "proxy daily budget exceeded")
		}
	}
	if st.AnalysisPending >= 0 {
		rows = append(rows, [2]string{"analysis_pending", fmt.Sprintf("%d", st.AnalysisPending)})
	} else if st.MongoOK {
		rows = append(rows, [2]string{"analysis_pending", "0"})
	}
	pretty.PrintLabeled(w, color, rows)
}

func printVersion(w io.Writer, goVersion string) {
	color := cliColor(w)
	if color {
		_, _ = fmt.Fprintf(w, "%sparser%s %s%s%s (%s)\n", pretty.Bold, pretty.Reset, pretty.Cyan, Version, pretty.Reset, goVersion)
		return
	}
	_, _ = fmt.Fprintf(w, "parser %s (%s)\n", Version, goVersion)
}
