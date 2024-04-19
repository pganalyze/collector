package runner

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

var helpers = []string{
	// Column stats
	`CREATE OR REPLACE FUNCTION pganalyze.get_column_stats() RETURNS SETOF pg_stats AS
$$
  /* pganalyze-collector */ SELECT schemaname, tablename, attname, inherited, null_frac, avg_width,
    n_distinct, NULL::anyarray, most_common_freqs, NULL::anyarray, correlation, NULL::anyarray,
    most_common_elem_freqs, elem_count_histogram
  FROM pg_catalog.pg_stats;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;`,

	// Extended stats
	`CREATE OR REPLACE FUNCTION pganalyze.get_relation_stats_ext() RETURNS TABLE(
  statistics_schemaname text, statistics_name text,
  inherited boolean, n_distinct pg_ndistinct, dependencies pg_dependencies,
  most_common_val_nulls boolean[], most_common_freqs float8[], most_common_base_freqs float8[]
) AS
$$
  /* pganalyze-collector */ SELECT statistics_schemaname::text, statistics_name::text,
  (row_to_json(se.*)::jsonb ->> 'inherited')::boolean AS inherited, n_distinct, dependencies,
  most_common_val_nulls, most_common_freqs, most_common_base_freqs
  FROM pg_catalog.pg_stats_ext se;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;`}

func GenerateHelperSql(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (string, error) {
	db, err := postgres.EstablishConnection(ctx, server, logger, globalCollectionOpts, "")
	if err != nil {
		return "", err
	}
	defer db.Close()

	version, err := postgres.GetPostgresVersion(ctx, logger, db)
	if err != nil {
		return "", fmt.Errorf("error collecting Postgres version: %s", err)
	}

	databases, _, err := postgres.GetDatabases(ctx, logger, db, version)
	if err != nil {
		return "", fmt.Errorf("error collecting pg_databases: %s", err)
	}

	output := ""
	for _, dbName := range postgres.GetDatabasesToCollect(server, databases) {
		output += fmt.Sprintf("\\c %s\n", pq.QuoteIdentifier(dbName))
		output += "CREATE SCHEMA IF NOT EXISTS pganalyze;\n"
		output += fmt.Sprintf("GRANT USAGE ON SCHEMA pganalyze TO %s;\n", server.Config.GetDbUsername())
		for _, helper := range helpers {
			output += helper + "\n"
		}
		output += "\n"
	}

	return output, nil
}
