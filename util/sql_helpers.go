package util

import (
	_ "embed"
)

//go:embed helpers/explain_analyze.sql
var ExplainAnalyzeHelper string

//go:embed helpers/get_stat_statements.sql
var GetStatStatementsHelper string
