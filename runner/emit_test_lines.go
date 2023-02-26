package runner

import (
	"context"
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func EmitTestLogMsg(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) error {
	db, err := postgres.EstablishConnection(ctx, server, logger, globalCollectionOpts, "")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, postgres.QueryMarkerSQL+fmt.Sprintf("DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: %s';\nEND$$;", server.Config.SectionName))
	return err
}

func EmitTestExplain(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) error {
	db, err := postgres.EstablishConnection(ctx, server, logger, globalCollectionOpts, "")
	if err != nil {
		return err
	}
	defer db.Close()
	// Emit a query that's slow enough to trigger an explain if a log_min_duration is
	// configured (either through auto_explain or the standard one)
	//
	// Note that we intentionally don't use the pganalyze collector query marker here,
	// since we want the EXPLAIN plan to show up in the pganalyze user interface
	_, err = db.ExecContext(ctx, `WITH naptime(value) AS (
SELECT
	COALESCE(pg_catalog.max(setting::float), 0) / 1000 * 1.2
FROM
	pg_settings
WHERE
	name IN ('log_min_duration_statement', 'auto_explain.log_min_duration')
)
SELECT pg_catalog.pg_sleep(value) FROM naptime WHERE value > 0`)
	return err
}
