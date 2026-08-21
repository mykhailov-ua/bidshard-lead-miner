package store

import (
	"time"
)

const (
	defaultStatsTimeout = 15 * time.Second
	maxTopSources       = 10
	maxKeywordStats     = 20
	maxBoostRows        = 20
)

type Options struct {
	DBName                 string
	LeadsCollection        string
	SourceStatsCollection  string
	KeywordStatsCollection string
	CrmBoostCollection     string
	LeadNotesCollection    string
	LeadCrmMetaCollection  string
	QueryTimeout           time.Duration
	WriteTimeout           time.Duration
	StatsTimeout           time.Duration
	ExportMaxRows          int64
	SearchTimeout          time.Duration
	SearchMaxRows          int64
}

func (o Options) statsTimeout() time.Duration {
	if o.StatsTimeout > 0 {
		return o.StatsTimeout
	}
	return defaultStatsTimeout
}

func (o Options) exportMaxRows() int64 {
	if o.ExportMaxRows > 0 {
		return o.ExportMaxRows
	}
	return defaultExportMaxRows
}

func (o Options) searchMaxRows() int64 {
	if o.SearchMaxRows > 0 {
		return o.SearchMaxRows
	}
	return defaultSearchMaxRows
}
