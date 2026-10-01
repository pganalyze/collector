package util

import (
	pg_query "github.com/pganalyze/pg_query_go/v18"
)

// FingerprintOpts - Options controlling how query fingerprints are calculated
type FingerprintOpts = pg_query.FingerprintOption

// FingerprintOptsPG17Compat - Fingerprint options matching how Postgres 17 and
// earlier calculate query IDs (relation names are fingerprinted including their
// schema, and aliases are ignored)
const FingerprintOptsPG17Compat = pg_query.FingerprintRangeVarPG17Compat

// FingerprintOptsDefault - Fingerprint options matching how Postgres 18 and
// newer calculate query IDs (aliases are fingerprinted instead of relation
// names when present, and schema names are ignored)
const FingerprintOptsDefault = pg_query.FingerprintDefault

// TryFingerprintQuery - Generates a unique fingerprint for the given query,
// and whether the query text had to be modified to generate a fingerprint.
//
// This matters for some callers to decide what to do with the fingerprint, for
// example if we see a truncated version of a query in pg_stat_activity, and then
// we see the full query text later via pg_stat_statements.
func TryFingerprintQuery(query string, filterQueryText string, trackActivityQuerySize int, opts FingerprintOpts) (fp uint64, virtual bool) {
	fp, err := pg_query.FingerprintToUInt64WithOpts(query, opts)
	if err != nil {
		virtual = true
		fixedQuery := fixTruncatedQuery(query)

		fp, err = pg_query.FingerprintToUInt64WithOpts(fixedQuery, opts)
		if err != nil {
			fp = fingerprintError(query, filterQueryText, trackActivityQuerySize)
			return
		}
	}

	return
}

// FingerprintQuery - Generates a unique fingerprint for the given query
func FingerprintQuery(query string, filterQueryText string, trackActivityQuerySize int, opts FingerprintOpts) (fp uint64) {
	fp, _ = TryFingerprintQuery(query, filterQueryText, trackActivityQuerySize, opts)
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
