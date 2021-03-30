package util

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v2"
)

// FingerprintQuery - Generates a unique fingerprint for the given query
func FingerprintQuery(query string) (fp uint64) {
	fp, err := pg_query.FingerprintToUInt64(query)
	if err != nil {
		fixedQuery := fixTruncatedQuery(query)

		fp, err = pg_query.FingerprintToUInt64(fixedQuery)
		if err != nil {
			fp = fingerprintError(query)
			return
		}
	}

	return
}

func fixTruncatedQuery(query string) string {
	if strings.Count(query, "'")%2 == 1 { // Odd number of '
		query += "'"
	}
	if strings.Count(query, "\"")%2 == 1 { // Odd number of "
		query += "\""
	}

	openParens := strings.Count(query, "(") - strings.Count(query, ")")
	if openParens > 0 {
		for i := 0; i < openParens; i++ {
			query += ")"
		}
	}

	return query
}

func fingerprintError(query string) (fp uint64) {
	return pg_query.HashXXH3_64([]byte(query), 0xee)
}
