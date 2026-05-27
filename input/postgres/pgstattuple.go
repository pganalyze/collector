package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "slices"
    pg_query "github.com/pganalyze/pg_query_go/v6"
    "google.golang.org/protobuf/proto"
)

func RunPgstattupleForQueryRun(ctx context.Context, db *sql.DB, query string, marker string) (result string, err error) {
    err = validatePgstattupleQuery(query)
    if err != nil {
        return
    }
    err = db.QueryRowContext(ctx, marker+query).Scan(&result)
    return
}

func validatePgstattupleQuery(query string) error {
    parseResult, err := pg_query.Parse(query)
    if err != nil {
        return fmt.Errorf("query is not permitted to run - failed to parse")
    }
    if len(parseResult.Stmts) != 1 {
        return fmt.Errorf("query is not permitted to run - multi-statement query string")
    }
    if parseResult.Stmts[0].Stmt.GetSelectStmt() == nil {
        return fmt.Errorf("query is not permitted to run - wrong statement type")
    }
    err = walkParseTree(parseResult, func(nodeType string, node proto.Message) error {
        if nodeType != "FuncCall" {
            return nil
        }

        // TODO: reject anything else, like selecting from a table

        f := node.(*pg_query.FuncCall)
        // The funcname field can be optionally schema qualified, so we take the last item in the list of names
        nameNode := f.Funcname[len(f.Funcname)-1]
        name := nameNode.GetString_().Sval

        if !slices.Contains(pgStattupleFunctions, name) {
            return fmt.Errorf("query is not permitted to run - function not allowed: %s", name)
        }
        return nil
    })
    if err != nil {
        return err
    }
    return nil
}

var pgStattupleFunctions = []string{
    "pgstattuple",
    "pgstatindex",
    "pgstatginindex",
    "pgstathashindex",
    "pgstattuple_approx",
    "pg_relpages",
}
