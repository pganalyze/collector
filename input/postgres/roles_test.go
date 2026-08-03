package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/pganalyze/collector/state"
)

// A multi-schema search_path (e.g. ALTER ROLE ... SET search_path = pg_catalog, public)
// is stored by Postgres as a single, quoted rolconfig element:
//
//	{"search_path=pg_catalog, public"}
//
// This makes sure we parse it back into one element with the surrounding quotes
// stripped, rather than naively splitting on the comma (which used to produce
// ["\"search_path=pg_catalog", " public\""]).
func TestGetRolesParsesMultiSchemaSearchPath(t *testing.T) {
	testDatabaseUrl := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseUrl == "" {
		t.Skipf("Skipping test requiring database connection since TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", testDatabaseUrl)
	if err != nil {
		t.Fatalf("Could not connect to test database: %s", err)
	}
	defer db.Close()

	const roleName = "pganalyze_test_search_path_role"
	if _, err := db.Exec("DROP ROLE IF EXISTS " + roleName); err != nil {
		t.Fatalf("Could not drop pre-existing role: %s", err)
	}
	if _, err := db.Exec("CREATE ROLE " + roleName); err != nil {
		t.Fatalf("Could not create role: %s", err)
	}
	defer db.Exec("DROP ROLE IF EXISTS " + roleName)

	if _, err := db.Exec("ALTER ROLE " + roleName + " SET search_path = pg_catalog, public"); err != nil {
		t.Fatalf("Could not set search_path: %s", err)
	}

	roles, err := getRoles(context.Background(), db, "")
	if err != nil {
		t.Fatalf("getRoles returned error: %s", err)
	}

	var found *state.PostgresRole
	for i := range roles {
		if roles[i].Name == roleName {
			found = &roles[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("role %s not returned by getRoles", roleName)
	}

	expected := []string{"search_path=pg_catalog, public"}
	if len(found.Config) != 1 || found.Config[0] != expected[0] {
		t.Errorf("expected Config %v, got %v", expected, found.Config)
	}
}
