package postgres

import (
    "context"
    "database/sql"
    "fmt"
    pg_query "github.com/pganalyze/pg_query_go/v6"
)

func RunRepackForQueryRun(ctx context.Context, db *sql.DB, query string, marker string) (result string, err error) {
    err = validateRepackQuery(query)
    if err != nil {
        return
    }
    _, err = db.ExecContext(ctx, marker+query)
    return
}

func validateRepackQuery(query string) error {
    parseResult, err := pg_query.Parse(query)
    if err != nil {
        return fmt.Errorf("query is not permitted to run - failed to parse")
    }
    if len(parseResult.Stmts) != 1 {
        return fmt.Errorf("query is not permitted to run - multi-statement query string")
    }
    // TODO: update pg_query to 19
    // TODO: require CONCURRENTLY
    // if parseResult.Stmts[0].Stmt.GetRepackStmt() == nil {
    //     return fmt.Errorf("query is not permitted to run - wrong statement type")
    // }
    return nil
}
