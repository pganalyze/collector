package util

import (
	pg_query "github.com/lfittl/pg_query_go"
	pg_query_nodes "github.com/lfittl/pg_query_go/nodes"
)

// IsUtilityStmt determines whether each statement in the query text is a
// utility statement or a standard SELECT/INSERT/UPDATE/DELETE statement.
func IsUtilityStmt(query string) ([]bool, error) {
	var result []bool
	parsetree, err := pg_query.Parse(query)
	if err != nil {
		return nil, err
	}
	for _, rawStmt := range parsetree.Statements {
		stmt := rawStmt.(pg_query_nodes.RawStmt).Stmt
		var isUtility bool
		switch stmt.(type) {
		case pg_query_nodes.SelectStmt, pg_query_nodes.InsertStmt, pg_query_nodes.UpdateStmt, pg_query_nodes.DeleteStmt:
			isUtility = false
		default:
			isUtility = true
		}
		result = append(result, isUtility)
	}
	return result, nil
}
