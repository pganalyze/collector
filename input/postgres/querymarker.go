package postgres

import (
	"strings"
)

const queryMarker string = "/* pganalyze-collector */"

func WithQueryMarker(sql string) string {
	return sql + "\n" + queryMarker
}

func collectorStatement(query string) bool {
	return strings.Contains(query, queryMarker)
}
