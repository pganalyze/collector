package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/util"
)

type helperTestpair struct {
	query          string
	params         []null.String
	paramTypes     []string
	analyzeFlags   []string
	expectedOutput string
	expectedError  string
}

var helperTests = []helperTestpair{
	{
		"SELECT 1",
		[]null.String{},
		[]string{},
		[]string{"VERBOSE OFF", "COSTS OFF", "ANALYZE", "TIMING OFF"},
		`[
  {
    "Plan": {
      "Node Type": "Result",
      "Parallel Aware": false,
      "Async Capable": false,
      "Actual Rows": 1,
      "Actual Loops": 1
    },
    "Planning Time": XXXXX,
    "Triggers": [
    ],
    "Execution Time": XXXXX
  }
]`,
		"",
	},
	{
		"SELECT pg_reload_conf()",
		[]null.String{},
		[]string{},
		[]string{"VERBOSE OFF", "COSTS OFF", "ANALYZE", "TIMING OFF"},
		"",
		"pq: permission denied for function pg_reload_conf",
	},
	{
		"UPDATE test SET id = 123",
		[]null.String{},
		[]string{},
		[]string{"VERBOSE OFF", "COSTS OFF", "ANALYZE", "TIMING OFF"},
		"",
		"pq: cannot execute UPDATE in a read-only transaction",
	},
	{
		"SELECT 1; UPDATE test SET id = 123",
		[]null.String{},
		[]string{},
		[]string{"VERBOSE OFF", "COSTS OFF", "ANALYZE", "TIMING OFF"},
		"",
		"pq: cannot run pganalyze.explain_analyze helper with a multi-statement query",
	},
	{
		"SELECT $1",
		/* EXECUTE pganalyze_explain_analyze (*/ []null.String{null.StringFrom("1); SELECT ('data'")}, /* ) */
		[]string{},
		[]string{"COSTS OFF"},
		`[
  {
    "Plan": {
      "Node Type": "Result",
      "Parallel Aware": false,
      "Async Capable": false,
      "Output": ["'1); SELECT (''data'''::text"]
    }
  }
]`,
		"",
	},
	{
		"SELECT $1",
		[]null.String{null.StringFrom("dummy")},
		/* PREPARE pganalyze_explain_analyze (*/ []string{"text) AS SELECT 'data'; PREPARE dummy (text"}, /*) AS [query] */
		[]string{"COSTS OFF"},
		"",
		`pq: type "text) AS SELECT 'data'; PREPARE dummy (text" does not exist`,
	},
	{
		"SELECT 'data'",
		[]null.String{},
		[]string{},
		[]string{"FORMAT JSON) SELECT 1; UPDATE test SET id = 123; EXPLAIN (COSTS OFF"},
		"",
		"pq: cannot run pganalyze.explain_analyze helper with invalid flag",
	},
	// Cases that are worth documenting by test (but they are not bugs, just things worth noting)
	{
		// DML statements for EXPLAIN (without ANALYZE) are permitted, if access is granted (they don't violate the rules of a read only transaction)
		"UPDATE test SET id = 123",
		[]null.String{},
		[]string{},
		[]string{"VERBOSE OFF", "COSTS OFF"},
		`[
  {
    "Plan": {
      "Node Type": "ModifyTable",
      "Operation": "Update",
      "Parallel Aware": false,
      "Async Capable": false,
      "Relation Name": "test",
      "Alias": "test",
      "Plans": [
        {
          "Node Type": "Seq Scan",
          "Parent Relationship": "Outer",
          "Parallel Aware": false,
          "Async Capable": false,
          "Relation Name": "test",
          "Alias": "test"
        }
      ]
    }
  }
]`,
		"",
	},
}

func TestExplainAnalyzeHelper(t *testing.T) {
	db := setupTest(t)
	defer db.Close()

	for _, pair := range helperTests {
		var output string
		var errStr string
		err := db.QueryRow("SELECT pganalyze.explain_analyze($1, $2, $3, $4)", pair.query, pq.Array(pair.params), pq.Array(pair.paramTypes), pq.Array(pair.analyzeFlags)).Scan(&output)
		if err != nil {
			errStr = fmt.Sprintf("%s", err)
		}

		// Avoid differences in test runs by masking total planning/execution time stats
		re := regexp.MustCompile(`("(Planning|Execution) Time":) [\d.]+`)
		output = re.ReplaceAllString(output, "$1 XXXXX")

		if output != pair.expectedOutput {
			t.Errorf("Incorrect output for query '%s' (direct):\n got: %s\n expected: %s", pair.query, output, pair.expectedOutput)
		}

		if errStr != pair.expectedError {
			t.Errorf("Incorrect error for query '%s' (direct):\n got: %s\n expected: %s", pair.query, errStr, pair.expectedError)
		}
	}
}

type queryRunTestpair struct {
	query          string
	params         []null.String
	paramTypes     []string
	expectedOutput string
	expectedError  string
}

var queryRunTests = []queryRunTestpair{
	{
		"SELECT 1",
		[]null.String{},
		[]string{},
		`[
  {
    "Plan": {
      "Node Type": "Result",
      "Parallel Aware": false,
      "Async Capable": false,
      "Startup Cost": XXXXX,
      "Total Cost": XXXXX,
      "Plan Rows": 1,
      "Plan Width": 4,
      "Actual Startup Time": XXXXX,
      "Actual Total Time": XXXXX,
      "Actual Rows": 1,
      "Actual Loops": 1,
      "Output": ["1"],
      "Shared Hit Blocks": 0,
      "Shared Read Blocks": 0,
      "Shared Dirtied Blocks": 0,
      "Shared Written Blocks": 0,
      "Local Hit Blocks": 0,
      "Local Read Blocks": 0,
      "Local Dirtied Blocks": 0,
      "Local Written Blocks": 0,
      "Temp Read Blocks": 0,
      "Temp Written Blocks": 0
    },
    "Planning": {
      "Shared Hit Blocks": 0,
      "Shared Read Blocks": 0,
      "Shared Dirtied Blocks": 0,
      "Shared Written Blocks": 0,
      "Local Hit Blocks": 0,
      "Local Read Blocks": 0,
      "Local Dirtied Blocks": 0,
      "Local Written Blocks": 0,
      "Temp Read Blocks": 0,
      "Temp Written Blocks": 0
    },
    "Planning Time": XXXXX,
    "Triggers": [
    ],
    "Execution Time": XXXXX
  }
]`,
		"",
	},
	{
		"SELECT pg_reload_conf()",
		[]null.String{},
		[]string{},
		"",
		"pq: permission denied for function pg_reload_conf",
	},
	{
		"UPDATE test SET id = 123",
		[]null.String{},
		[]string{},
		"",
		"query is not permitted to run - DML statement",
	},
	{
		"SELECT 1; UPDATE test SET id = 123",
		[]null.String{},
		[]string{},
		"",
		"query is not permitted to run - multi-statement query string",
	},
	{
		"SELECT dblink_exec('host=myhost user=myuser password=mypass dbname=mydb', dblink_build_sql_insert('secret_table', '1', 1, '{\"1\"}', '{\"1\"}'))",
		[]null.String{},
		[]string{},
		"",
		"query is not permitted to run - function not allowed: dblink_exec",
	},
	{
		"SELECT $1",
		/* EXECUTE pganalyze_explain_analyze (*/ []null.String{null.StringFrom("1); SELECT ('data'")}, /* ) */
		[]string{},
		`[
  {
    "Plan": {
      "Node Type": "Result",
      "Parallel Aware": false,
      "Async Capable": false,
      "Startup Cost": XXXXX,
      "Total Cost": XXXXX,
      "Plan Rows": 1,
      "Plan Width": 32,
      "Actual Startup Time": XXXXX,
      "Actual Total Time": XXXXX,
      "Actual Rows": 1,
      "Actual Loops": 1,
      "Output": ["'1); SELECT (''data'''::text"],
      "Shared Hit Blocks": 0,
      "Shared Read Blocks": 0,
      "Shared Dirtied Blocks": 0,
      "Shared Written Blocks": 0,
      "Local Hit Blocks": 0,
      "Local Read Blocks": 0,
      "Local Dirtied Blocks": 0,
      "Local Written Blocks": 0,
      "Temp Read Blocks": 0,
      "Temp Written Blocks": 0
    },
    "Planning": {
      "Shared Hit Blocks": 0,
      "Shared Read Blocks": 0,
      "Shared Dirtied Blocks": 0,
      "Shared Written Blocks": 0,
      "Local Hit Blocks": 0,
      "Local Read Blocks": 0,
      "Local Dirtied Blocks": 0,
      "Local Written Blocks": 0,
      "Temp Read Blocks": 0,
      "Temp Written Blocks": 0
    },
    "Planning Time": XXXXX,
    "Triggers": [
    ],
    "Execution Time": XXXXX
  }
]`,
		"",
	},
	{
		"SELECT $1",
		[]null.String{null.StringFrom("dummy")},
		/* PREPARE pganalyze_explain_analyze (*/ []string{"text) AS SELECT 'data'; PREPARE dummy (text"}, /*) AS [query] */
		"",
		`pq: type "text) AS SELECT 'data'; PREPARE dummy (text" does not exist`,
	},
}

func TestExplainAnalyzeForQueryRun(t *testing.T) {
	db := setupTest(t)
	defer db.Close()

	for _, pair := range queryRunTests {
		var errStr string
		output, err := postgres.RunExplainAnalyzeForQueryRun(context.Background(), db, pair.query, pair.params, pair.paramTypes, "")
		if err != nil {
			errStr = fmt.Sprintf("%s", err)
		}

		// Avoid differences in test runs by masking timing stats
		re := regexp.MustCompile(`("(?:Planning Time|Execution Time|Startup Cost|Total Cost|Actual Startup Time|Actual Total Time)":) [\d.]+`)
		output = re.ReplaceAllString(output, "$1 XXXXX")

		if output != pair.expectedOutput {
			t.Errorf("Incorrect output for query '%s' (via collector code):\n got: %s\n expected: %s", pair.query, output, pair.expectedOutput)
		}

		if errStr != pair.expectedError {
			t.Errorf("Incorrect error for query '%s' (via collector code):\n got: %s\n expected: %s", pair.query, errStr, pair.expectedError)
		}
	}
}

func setupTest(t *testing.T) *sql.DB {
	testDatabaseUrl := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseUrl == "" {
		t.Skipf("Skipping test requiring database connection since TEST_DATABASE_URL is not set")
	}
	db, err := makeConnection(testDatabaseUrl)
	if err != nil {
		t.Fatalf("Could not connect to test database: %s", err)
	}

	err = setupHelperAndRole(db)
	if err != nil {
		t.Fatalf("Could not set up helper: %s", err)
	}

	_, err = db.Exec("GRANT pg_read_all_data TO pganalyze_explain")
	if err != nil {
		t.Fatalf("Could not grant permissions: %s", err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id int)")
	if err != nil {
		t.Fatalf("Could not create test table: %s", err)
	}

	// We're granting the write permissions here to verify the function runs as a read-only transaction
	_, err = db.Exec("GRANT ALL ON test TO pganalyze_explain")
	if err != nil {
		t.Fatalf("Could not GRANT on test table: %s", err)
	}

	return db
}

func makeConnection(testDatabaseUrl string) (*sql.DB, error) {
	db, err := sql.Open("postgres", testDatabaseUrl)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(30 * time.Second)

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	// Emit notices to logs to help with function debugging
	_, err = db.Exec("SET log_min_messages = NOTICE")
	if err != nil {
		db.Close()
		return nil, err
	}

	// Don't generate queryid, to avoid making output different across clusters
	_, err = db.Exec("SET compute_query_id = off")
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func setupHelperAndRole(db *sql.DB) (err error) {
	// Clean up previous helper and role if it exists
	_, err = db.Exec("DROP FUNCTION IF EXISTS pganalyze.explain_analyze(text, text[], text[], text[])")
	if err != nil {
		return
	}

	db.Exec("DROP OWNED BY pganalyze_explain")
	_, err = db.Exec("DROP ROLE IF EXISTS pganalyze_explain")
	if err != nil {
		return
	}

	_, err = db.Exec("CREATE ROLE pganalyze_explain")
	if err != nil {
		return
	}

	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS pganalyze")
	if err != nil {
		return
	}

	_, err = db.Exec("GRANT CREATE ON SCHEMA pganalyze TO pganalyze_explain")
	if err != nil {
		return
	}

	_, err = db.Exec("SET ROLE pganalyze_explain")
	if err != nil {
		return
	}

	_, err = db.Exec(util.ExplainAnalyzeHelper)
	if err != nil {
		return
	}

	_, err = db.Exec("RESET ROLE")
	if err != nil {
		return
	}

	_, err = db.Exec("REVOKE CREATE ON SCHEMA pganalyze FROM pganalyze_explain")
	if err != nil {
		return
	}

	return
}
