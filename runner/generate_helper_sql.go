package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

var statsHelpers = []string{
	// Column stats
	`DROP FUNCTION IF EXISTS pganalyze.get_column_stats;
CREATE FUNCTION pganalyze.get_column_stats() RETURNS TABLE(
  schemaname name, tablename name, attname name, inherited bool, null_frac real, avg_width int, n_distinct real, correlation real
) AS $$
  /* pganalyze-collector */
  SELECT schemaname, tablename, attname, inherited, null_frac, avg_width, n_distinct, correlation
  FROM pg_catalog.pg_stats
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND tablename <> 'pg_subscription';
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;`,

	// Extended stats
	`DROP FUNCTION IF EXISTS pganalyze.get_relation_stats_ext;
CREATE FUNCTION pganalyze.get_relation_stats_ext() RETURNS TABLE(
  statistics_schemaname text, statistics_name text,
  inherited boolean, n_distinct pg_ndistinct, dependencies pg_dependencies,
  most_common_val_nulls boolean[], most_common_freqs float8[], most_common_base_freqs float8[]
) AS
$$
  /* pganalyze-collector */ SELECT statistics_schemaname::text, statistics_name::text,
  (row_to_json(se.*)::jsonb ->> 'inherited')::boolean AS inherited, n_distinct, dependencies,
  most_common_val_nulls, most_common_freqs, most_common_base_freqs
  FROM pg_catalog.pg_stats_ext se
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND tablename <> 'pg_subscription';
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;`}

func GenerateStatsHelperSql(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (string, error) {
	db, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return "", err
	}
	defer db.Close()

	databases, _, err := postgres.GetDatabases(ctx, db)
	if err != nil {
		return "", fmt.Errorf("error collecting pg_databases: %s", err)
	}

	output := strings.Builder{}
	for _, dbName := range postgres.GetDatabasesToCollect(server.Config, databases) {
		output.WriteString(fmt.Sprintf("\\c %s\n", pq.QuoteIdentifier(dbName)))
		output.WriteString("CREATE SCHEMA IF NOT EXISTS pganalyze;\n")
		output.WriteString(fmt.Sprintf("GRANT USAGE ON SCHEMA pganalyze TO %s;\n", server.Config.GetDbUsername()))
		for _, helper := range statsHelpers {
			output.WriteString(helper + "\n")
		}
		output.WriteString("\n")
	}

	return output.String(), nil
}

func GenerateExplainAnalyzeHelperSql(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (string, error) {
	db, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return "", err
	}
	defer db.Close()

	databases, _, err := postgres.GetDatabases(ctx, db)
	if err != nil {
		return "", fmt.Errorf("error collecting pg_databases: %s", err)
	}

	output := strings.Builder{}
	for _, dbName := range postgres.GetDatabasesToCollect(server.Config, databases) {
		output.WriteString(fmt.Sprintf("\\c %s\n", pq.QuoteIdentifier(dbName)))
		output.WriteString("CREATE SCHEMA IF NOT EXISTS pganalyze;\n")
		output.WriteString(fmt.Sprintf("GRANT USAGE ON SCHEMA pganalyze TO %s;\n", pq.QuoteIdentifier(server.Config.GetDbUsername())))
		output.WriteString(fmt.Sprintf("GRANT CREATE ON SCHEMA pganalyze TO %s;\n", pq.QuoteIdentifier(opts.GenerateExplainAnalyzeHelperRole)))
		output.WriteString(fmt.Sprintf("SET ROLE %s;\n", pq.QuoteIdentifier(opts.GenerateExplainAnalyzeHelperRole)))
		output.WriteString(util.ExplainAnalyzeHelper + "\n")
		output.WriteString("RESET ROLE;\n")
		output.WriteString(fmt.Sprintf("REVOKE CREATE ON SCHEMA pganalyze FROM %s;\n", pq.QuoteIdentifier(opts.GenerateExplainAnalyzeHelperRole)))
		output.WriteString("\n")
	}

	return output.String(), nil
}
