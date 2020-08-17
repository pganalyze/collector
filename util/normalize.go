package util

import pg_query "github.com/lfittl/pg_query_go"

func NormalizeQuery(query string, filterQueryText string) string {
	normalizedQuery, err := pg_query.Normalize(query)
	if err != nil {
		if filterQueryText == "none" {
			normalizedQuery = query
		} else {
			normalizedQuery = "<unparsable query>"
		}
	}
	return normalizedQuery
}
