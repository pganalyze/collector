package util

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// TryFingerprintQuery - Generates a unique fingerprint for the given query,
// and whether the query text had to be massaged to generate a fingerprint
func TryFingerprintQuery(query string, filterQueryText string, trackActivityQuerySize int) (fp uint64, virtual bool) {
	fp, err := pg_query.FingerprintToUInt64(query)
	if err != nil {
		virtual = true
		fixedQuery := fixTruncatedQuery(query)

		fp, err = pg_query.FingerprintToUInt64(fixedQuery)
		if err != nil {
			fp = fingerprintError(query, filterQueryText, trackActivityQuerySize)
			return
		}
	}

	return
}

// FingerprintQuery - Generates a unique fingerprint for the given query
func FingerprintQuery(query string, filterQueryText string, trackActivityQuerySize int) (fp uint64) {
	fp, _ = TryFingerprintQuery(query, filterQueryText, trackActivityQuerySize)
	return
}

// FingerprintText - Generates a fingerprint for static texts (used for error scenarios)
func FingerprintText(query string) (fp uint64) {
	return pg_query.HashXXH3_64([]byte(query), 0xee)
}

func fingerprintError(query string, filterQueryText string, trackActivityQuerySize int) (fp uint64) {
	if filterQueryText == "none" {
		return FingerprintText(query)
	} else if len(query) == trackActivityQuerySize-1 {
		return FingerprintText(QueryTextTruncated)
	} else {
		return FingerprintText(QueryTextUnparsable)
	}
}
