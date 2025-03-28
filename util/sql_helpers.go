package util

import (
	_ "embed"
)

//go:embed helpers/explain_analyze.sql
var ExplainAnalyzeHelper string

//go:embed helpers/get_stat_statements.sql
var GetStatStatementsHelper string

//go:embed helpers/get_column_stats.sql
var GetColumnStatsHelper string

//go:embed helpers/get_relation_stats_ext.sql
var GetRelationStatsExtHelper string
