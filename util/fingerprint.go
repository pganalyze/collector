package util

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strings"

	pg_query "github.com/lfittl/pg_query_go"
)

// FingerprintQuery - Generates a unique SHA-1 fingerprint for the given query
func FingerprintQuery(query string) (fp [21]byte) {
	fingerprintHex, err := pg_query.FastFingerprint(query)
	if err != nil {
		fixedQuery := fixTruncatedQuery(query)

		fingerprintHex, err = pg_query.FastFingerprint(fixedQuery)
		if err != nil {
			fp = fingerprintError(query)
			return
		}
	}

	fingerprint, err := hex.DecodeString(fingerprintHex)
	if err != nil {
		fp = fingerprintError(query)
		return
	}

	copy(fp[:], fingerprint)

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

func fingerprintError(query string) (fp [21]byte) {
	fp[0] = 0xee
	h := sha1.New()
	io.WriteString(h, query)
	copy(fp[1:], h.Sum(nil))
	return
}
