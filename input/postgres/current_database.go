package postgres

import (
	"context"
	"database/sql"

	"github.com/pganalyze/collector/state"
)

// CurrentDatabaseOid - Find OID of the database we're currently connected to
func CurrentDatabaseOid(ctx context.Context, db *sql.DB) (result state.Oid, err error) {
	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT oid FROM pg_catalog.pg_database WHERE datname = pg_catalog.current_database()").Scan(&result)
	return
}

// CurrentDatabaseName - Get name of the database we're currently connected to
func CurrentDatabaseName(ctx context.Context, db *sql.DB) (result string, err error) {
	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.current_database()").Scan(&result)
	return
}
