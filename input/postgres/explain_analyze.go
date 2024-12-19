package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/util"
)

func RunExplainAnalyzeForQueryRun(ctx context.Context, db *sql.DB, query string, parameters []null.String, parameterTypes []string, marker string) (result string, err error) {
	err = validateQuery(query)
	if err != nil {
		return
	}

	// Warm up caches without collecting timing info (slightly faster)
	_, err = runExplainAnalyze(ctx, db, query, parameters, parameterTypes, []string{"ANALYZE", "TIMING OFF"}, marker)
	if err != nil {
		if !strings.Contains(err.Error(), "statement timeout") {
			return
		}

		// Run again if it was a timeout error, to make sure we got the caches warmed up all the way
		_, err = runExplainAnalyze(ctx, db, query, parameters, parameterTypes, []string{"ANALYZE", "TIMING OFF"}, marker)
		if err != nil {
			if !strings.Contains(err.Error(), "statement timeout") {
				return
			}

			// If it timed out again, capture a non-ANALYZE EXPLAIN instead
			return runExplainAnalyze(ctx, db, query, parameters, parameterTypes, []string{}, marker)
		}
	}

	// Run EXPLAIN ANALYZE once more to get a warm cache result (this is the one we return)
	return runExplainAnalyze(ctx, db, query, parameters, parameterTypes, []string{"ANALYZE", "BUFFERS"}, marker)
}

func runExplainAnalyze(ctx context.Context, db *sql.DB, query string, parameters []null.String, parameterTypes []string, analyzeFlags []string, marker string) (explainOutput string, err error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, marker+"SELECT pganalyze.explain_analyze($1, $2, $3, $4)", marker+query, pq.Array(parameters), pq.Array(parameterTypes), pq.Array(analyzeFlags)).Scan(&explainOutput)

	return
}

func validateQuery(query string) error {
	var isUtil []bool
	// To be on the safe side never EXPLAIN a statement that can't be parsed,
	// or multiple statements in one (leading to accidental execution)
	isUtil, err := util.IsUtilityStmt(query)
	if err != nil || len(isUtil) != 1 || isUtil[0] {
		err = fmt.Errorf("query is not permitted to run (multi-statement or utility command?)")
		return err
	}

	// TODO: Consider adding additional checks here (e.g. blocking known bad function calls)

	return nil
}
