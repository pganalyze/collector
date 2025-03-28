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
		output.WriteString(util.GetColumnStatsHelper + "\n")
		output.WriteString(util.GetRelationStatsExtHelper + "\n")
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
