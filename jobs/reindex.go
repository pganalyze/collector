package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type ReindexParameters struct {
	DatabaseName string `json:"database_name"`
	SchemaName string `json:"schema_name"`
	IndexName string `json:"index_name"`
}

type ReindexResult struct {
	OldIndexSizeBytes int64 `json:"old_index_size_bytes"`
	NewIndexSizeBytes int64 `json:"new_index_size_bytes"`
}

const indexSizeSql = `
SELECT pg_catalog.pg_relation_size($1::regclass)
`

const reindexSql = `
REINDEX INDEX %s.%s
`

const checkTablespaceUsageSql = `
SELECT has_tablespace_privilege(
	current_user,
	COALESCE(
		NULLIF(
			(SELECT reltablespace FROM pg_class WHERE relnamespace = $1::regnamespace AND relname = $2),
			0),
		(SELECT dattablespace FROM pg_database WHERE datname = current_database())
		),
	'CREATE')
`

const checkTableownerSql = `
SELECT relowner = current_user::regrole FROM pg_class WHERE relnamespace = $1::regnamespace AND relname = $2
`

func initReindexJob(ctx context.Context, server *state.Server, prefixedLogger *util.Logger, globalCollectionOpts state.CollectionOpts, paramsIn []byte) (*sql.DB, *ReindexParameters, error) {
	var params *ReindexParameters
	err := json.Unmarshal(paramsIn, &params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal parameters: %s", err)
	}

	db, err := postgres.EstablishConnection(ctx, server, prefixedLogger, globalCollectionOpts, params.DatabaseName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %s", err)
	}

	return db, params, nil
}

func getIndexSize(ctx context.Context, db *sql.DB, params *ReindexParameters) (int64, error) {
	var indexSize int64
	err := db.QueryRowContext(ctx, postgres.QueryMarkerSQL+indexSizeSql, fmt.Sprintf("%s.%s", params.SchemaName, params.IndexName)).Scan(&indexSize)
	if err != nil {
		return 0, err
	}
	return indexSize, nil
}

func CheckSupportForReindexJob(ctx context.Context, server *state.Server, prefixedLogger *util.Logger, globalCollectionOpts state.CollectionOpts, paramsIn []byte) error {
	db, params, err := initReindexJob(ctx, server, prefixedLogger, globalCollectionOpts, paramsIn)
	if err != nil {
		return err
	}
	defer db.Close()

	var hasTablespaceUsage bool
	err = db.QueryRowContext(ctx, postgres.QueryMarkerSQL+checkTablespaceUsageSql, params.SchemaName, params.IndexName).Scan(&hasTablespaceUsage)
	if err != nil {
		return fmt.Errorf("REINDEX not supported, failed to check tablespace usage permission: %s", err)
	}
	if !hasTablespaceUsage {
		return fmt.Errorf("REINDEX not supported, missing CREATE permissions on tablespace")
	}

	var isTableOwner bool
	err = db.QueryRowContext(ctx, postgres.QueryMarkerSQL+checkTableownerSql, params.SchemaName, params.IndexName).Scan(&isTableOwner)
	if err != nil {
		return fmt.Errorf("REINDEX not supported, failed to check table owner: %s", err)
	}

	if !(isTableOwner || postgres.ConnectedAsSuperUser(ctx, db, server.Config.SystemType)) {
		return fmt.Errorf("REINDEX not supported, user is not table owner and missing superuser permissions")
	}

	return nil
}

func RunReindexJob(ctx context.Context, server *state.Server, prefixedLogger *util.Logger, globalCollectionOpts state.CollectionOpts, paramsIn []byte) ([]byte, error) {
	var result ReindexResult

	db, params, err := initReindexJob(ctx, server, prefixedLogger, globalCollectionOpts, paramsIn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	result.OldIndexSizeBytes, err = getIndexSize(ctx, db, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get old index size: %s", err)
	}

	// TODO: Check if there is a lock held and return early (or use lock_timeout?)

	// TODO: Support helper function
	// TODO: Is there a SQL injection risk here when not using the helper function?
	// TODO: If this fails (assuming we don't crash), can we drop an invalid index, if one was created?
	_, err = db.ExecContext(ctx, postgres.QueryMarkerSQL+fmt.Sprintf(reindexSql, params.SchemaName, params.IndexName))
	if err != nil {
		return nil, fmt.Errorf("failed to run REINDEX: %s", err)
	}

	result.NewIndexSizeBytes, err = getIndexSize(ctx, db, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get new index size: %s", err)
	}

	resultJson, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result JSON: %s", err)
	}

	return resultJson, nil
}
