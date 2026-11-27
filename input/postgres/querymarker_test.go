package postgres

import (
	"testing"
)

type collectorStatementTestpair struct {
	query    string
	expected bool
}

var collectorStatementTests = []collectorStatementTestpair{
	{
		"/* pganalyze-collector */ SELECT pg_catalog.version()",
		true,
	},
	{
		WithQueryMarker("SELECT pg_catalog.version()"),
		true,
	},
	{
		"SELECT pg_catalog.version()",
		false,
	},
}

func TestCollectorStatement(t *testing.T) {
	for _, pair := range collectorStatementTests {
		output := collectorStatement(pair.query)
		if output != pair.expected {
			t.Errorf("Incorrect output for query '%s':\n got: %t\n expected: %t", pair.query, output, pair.expected)
		}
	}
}
