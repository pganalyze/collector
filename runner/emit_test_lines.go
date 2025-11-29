package runner

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func EmitTestLogMsg(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) error {
	db, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, postgres.QueryMarkerSQL+fmt.Sprintf("DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: %s';\nEND$$;", server.Config.SectionName))
	return err
}

func EmitTestExplain(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) error {
	db, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return err
	}
	defer db.Close()
	// Emit a query that's slow enough to trigger an explain if a log_min_duration is
	// configured (either through auto_explain or the standard one)
	//
	// Note that we intentionally don't use the pganalyze collector query marker here,
	// since we want the EXPLAIN plan to show up in the pganalyze user interface
	var naptime float64
	var setting string
	if server.Config.EnableLogExplain {
		setting = "log_min_duration_statement"
	} else {
		setting = "auto_explain.log_min_duration"
	}
	err = db.QueryRowContext(ctx, `WITH naptime(value) AS (
SELECT
	COALESCE(pg_catalog.max(setting::float), 0) / 1000 * 1.2
FROM
	pg_settings
WHERE
	name = $1
), nap AS (
	SELECT pg_catalog.pg_sleep(value) FROM naptime WHERE value >= 0
)
SELECT value FROM naptime, nap`, setting).Scan(&naptime)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("could not check current value for setting '%s'", setting)
		}
		return err
	}
	if naptime < 0 {
		return fmt.Errorf("setting '%s' disabled; could not generate EXPLAIN plan", setting)
	}
	return nil
}
