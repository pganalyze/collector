package util

import pg_query "github.com/lfittl/pg_query_go"

func NormalizeQuery(query string) string {
	normalizedQuery, err := pg_query.Normalize(query)
	if err != nil {
		normalizedQuery = "<truncated query>"
	}
	return normalizedQuery
}
