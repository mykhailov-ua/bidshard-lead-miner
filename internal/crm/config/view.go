package config

import (
	"fmt"
	"io"
)

func (v ConfigView) WriteText(out io.Writer) error {
	lines := []string{
		fmt.Sprintf("mongo_uri=%s", v.MongoURI),
		fmt.Sprintf("mongo_db=%s", v.MongoDB),
		fmt.Sprintf("mongo_collection=%s", v.MongoCollection),
		fmt.Sprintf("source_stats_collection=%s", v.SourceStatsCollection),
		fmt.Sprintf("keyword_stats_collection=%s", v.KeywordStatsCollection),
		fmt.Sprintf("crm_boost_collection=%s", v.CrmBoostCollection),
		fmt.Sprintf("lead_notes_collection=%s", v.LeadNotesCollection),
		fmt.Sprintf("lead_crm_meta_collection=%s", v.LeadCrmMetaCollection),
		fmt.Sprintf("shutdown_timeout=%s", v.ShutdownTimeout),
		fmt.Sprintf("query_timeout=%s", v.QueryTimeout),
		fmt.Sprintf("write_timeout=%s", v.WriteTimeout),
		fmt.Sprintf("stats_timeout=%s", v.StatsTimeout),
		fmt.Sprintf("webhook_addr=%s", v.WebhookAddr),
		fmt.Sprintf("webhook_secret=%s", v.WebhookSecret),
		fmt.Sprintf("metrics_addr=%s", v.MetricsAddr),
		fmt.Sprintf("pprof_addr=%s", v.PprofAddr),
		fmt.Sprintf("export_max_rows=%d", v.ExportMaxRows),
		fmt.Sprintf("search_timeout=%s", v.SearchTimeout),
		fmt.Sprintf("search_max_rows=%d", v.SearchMaxRows),
		fmt.Sprintf("log_format=%s", v.LogFormat),
		fmt.Sprintf("log_level=%s", v.LogLevel),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}
