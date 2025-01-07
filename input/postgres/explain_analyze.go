package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	parseResult, err := pg_query.Parse(query)
	if err != nil {
		return fmt.Errorf("query is not permitted to run - failed to parse")
	}
	if len(parseResult.Stmts) != 1 {
		return fmt.Errorf("query is not permitted to run - multi-statement query string")
	}
	for _, rawStmt := range parseResult.Stmts {
		stmt := rawStmt.Stmt.Node
		switch stmt.(type) {
		case *pg_query.Node_SelectStmt:
			// Allowed, continue
			// Note that we permit wCTEs here (for now), and instead rely on the read-only transaction to block them
		case *pg_query.Node_InsertStmt, *pg_query.Node_UpdateStmt, *pg_query.Node_DeleteStmt:
			return fmt.Errorf("query is not permitted to run - DML statement")
		default:
			return fmt.Errorf("query is not permitted to run - utility statement")
		}
	}
	err = validateBlockedFunctions(parseResult)
	if err != nil {
		return err
	}

	return nil
}

var blockedFunctions = []string{
	// Blocked because these functions allow exfiltrating data to external servers
	"dblink",
	"dblink_connect",
	"dblink_exec",
	// Blocked because these functions allow executing arbitrary SQL as input (which can workaround other checks)
	"xpath_table",
}

func validateBlockedFunctions(msg proto.Message) error {
	return protorange.Range(msg.ProtoReflect(), func(p protopath.Values) error {
		last := p.Index(-1)
		m, ok := last.Value.Interface().(protoreflect.Message)
		if ok && m.Descriptor().Name() == "FuncCall" {
			f := m.Interface().(*pg_query.FuncCall)
			nameNode := f.Funcname[len(f.Funcname)-1]
			name := nameNode.GetString_().Sval
			if slices.Contains(blockedFunctions, name) {
				return fmt.Errorf("query is not permitted to run - function not allowed: %s", name)
			}
		}
		return nil
	})
}
