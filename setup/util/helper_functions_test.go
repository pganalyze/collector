package util

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/lib/pq"
)

// The helper runs with the caller's search_path, but its own references must not be affected by
// objects in the caller's temporary schema, which is searched first for relations and types.
func TestExplainHelperIgnoresTempSchema(t *testing.T) {
	testDatabaseUrl := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseUrl == "" {
		t.Skipf("Skipping test requiring database connection since TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", testDatabaseUrl)
	if err != nil {
		t.Fatalf("Could not connect to test database: %s", err)
	}
	defer db.Close()

	// Run everything in a transaction that gets rolled back, to leave the test database unchanged
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Could not start transaction: %s", err)
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		"CREATE SCHEMA IF NOT EXISTS pganalyze",
		ExplainHelper.GetDefinition(),
		// Shadow the text type (used for variables) and the unknown type (used for parameter casts)
		"CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK (false)",
		"CREATE DOMAIN pg_temp.unknown AS pg_catalog.text CHECK (false)",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			t.Fatalf("Could not run %q: %s", stmt, err)
		}
	}

	var output string
	err = tx.QueryRow("SELECT pganalyze.explain($1, $2)", "SELECT $1::int", pq.Array([]string{"5"})).Scan(&output)
	if err != nil {
		t.Fatalf("explain with shadowing objects in pg_temp returned error: %s", err)
	}
	if !strings.Contains(output, `"Node Type": "Result"`) {
		t.Errorf("explain with shadowing objects in pg_temp returned unexpected output: %s", output)
	}
}
