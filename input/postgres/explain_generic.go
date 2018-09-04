package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pganalyze/collector/output/pganalyze_collector"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func GetGenericExplains(logger *util.Logger, db *sql.DB, statements state.PostgresStatementMap) (result state.PostgresStatementExplainMap, resultErr error) {
	var err error

	result = make(state.PostgresStatementExplainMap)

	for key, value := range statements {
		queryText := value.NormalizedQuery

		// Crude safety measure to avoid multi-statement texts in old Postgres versions
		if strings.Contains(queryText, ";") {
			continue
		}

		// TODO: We need to replace ? with $n for Postgres versions older than 10

		_, err = db.Exec(QueryMarkerSQL + "PREPARE pganalyze_explain AS " + queryText)
		if err != nil {
			logger.PrintVerbose("Failed generic EXPLAIN due to PREPARE failure: %s", err)
		} else {
			rows, err := db.Query(QueryMarkerSQL + "SELECT typname FROM pg_type t JOIN (SELECT unnest(parameter_types)::oid AS oid FROM pg_prepared_statements WHERE name = 'pganalyze_explain') x ON (t.oid = x.oid)")
			if err != nil {
				logger.PrintVerbose("Failed to get type names: %s", err)
			} else {
				defer rows.Close()
				placeholderNum := 1
				for rows.Next() {
					var typname string
					err = rows.Scan(&typname)
					if err != nil {
						resultErr = fmt.Errorf("Unexpected row scan error: %s", err)
						return
					}
					queryText = strings.Replace(queryText, fmt.Sprintf("$%d", placeholderNum), fmt.Sprintf("((SELECT null::%s)::%s)", typname, typname), 1)
					placeholderNum++
				}
			}
			_, err = db.Exec(QueryMarkerSQL + "DEALLOCATE pganalyze_explain")
			if err != nil {
				logger.PrintVerbose("Failed to deallocate prepared statement used for explain", err)
				return
			}

			var explainOutput string
			err = db.QueryRow(QueryMarkerSQL + "EXPLAIN (FORMAT JSON) " + queryText).Scan(&explainOutput)
			if err != nil {
				logger.PrintVerbose("Failed to EXPLAIN: %s", err)
			} else {
				result[key] = state.PostgresStatementExplain{
					ExplainOutput: explainOutput,
					ExplainError:  "",
					ExplainFormat: pganalyze_collector.QueryExplainInformation_JSON_EXPLAIN_FORMAT,
					ExplainSource: pganalyze_collector.QueryExplainInformation_GENERIC_EXPLAIN_SOURCE,
				}
			}
		}
	}

	return
}
