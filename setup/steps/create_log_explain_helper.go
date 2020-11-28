package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var CreateLogExplainHelper = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Create log-based EXPLAIN helper function",
	Check: func(state *s.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(state.CurrentSection)
		if err != nil {
			return false, err
		}
		if !logExplain {
			return true, nil
		}
		monitoredDBs, err := getMonitoredDBs(state)
		if err != nil {
			return false, err
		}

		for _, db := range monitoredDBs {
			dbRunner := state.QueryRunner.InDB(db)
			isValid, err := util.ValidateHelperFunction("explain", dbRunner)
			if !isValid || err != nil {
				return isValid, err
			}
		}
		return true, nil
	},
	Run: func(state *s.SetupState) error {
		var doCreate bool
		if state.Inputs.Scripted {
			if !state.Inputs.CreateExplainHelper.Valid || !state.Inputs.CreateExplainHelper.Bool {
				return errors.New("create_explain_helper flag not set and helper function does not exist or does not match expected signature on all monitored databases")
			}
			doCreate = state.Inputs.CreateExplainHelper.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Create (or update) EXPLAIN helper function in each monitored database (will be saved to Postgres)?",
				Default: false,
			}, &doCreate)
			if err != nil {
				return err
			}
		}

		if !doCreate {
			return nil
		}
		monitoredDBs, err := getMonitoredDBs(state)
		if err != nil {
			return err
		}
		for _, db := range monitoredDBs {
			err := createHelperInDB(state, db)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func createHelperInDB(state *s.SetupState, db string) error {
	dbRunner := state.QueryRunner.InDB(db)
	isValid, err := util.ValidateHelperFunction("explain", dbRunner)
	if err != nil {
		return err
	}
	if isValid {
		return nil
	}
	userKey, err := state.CurrentSection.GetKey("db_username")
	if err != nil {
		return err
	}
	pgaUser := userKey.String()

	return dbRunner.Exec(
		fmt.Sprintf(
			`CREATE SCHEMA IF NOT EXISTS pganalyze; GRANT USAGE ON SCHEMA pganalyze TO %s;`,
			pq.QuoteIdentifier(pgaUser),
		) + `
CREATE OR REPLACE FUNCTION pganalyze.explain(query text, params text[]) RETURNS text AS
$$
DECLARE
	prepared_query text;
	prepared_params text;
	result text;
BEGIN
	SELECT regexp_replace(query, ';+\s*\Z', '') INTO prepared_query;
	IF prepared_query LIKE '%;%' THEN
		RAISE EXCEPTION 'cannot run EXPLAIN when query contains semicolon';
	END IF;

	IF array_length(params, 1) > 0 THEN
		SELECT string_agg(quote_literal(param) || '::unknown', ',') FROM unnest(params) p(param) INTO prepared_params;

		EXECUTE 'PREPARE pganalyze_explain AS ' || prepared_query;
		BEGIN
			EXECUTE 'EXPLAIN (VERBOSE, FORMAT JSON) EXECUTE pganalyze_explain(' || prepared_params || ')' INTO STRICT result;
		EXCEPTION WHEN OTHERS THEN
			DEALLOCATE pganalyze_explain;
			RAISE;
		END;
		DEALLOCATE pganalyze_explain;
	ELSE
		EXECUTE 'EXPLAIN (VERBOSE, FORMAT JSON) ' || prepared_query INTO STRICT result;
	END IF;

	RETURN result;
END
$$ LANGUAGE plpgsql VOLATILE SECURITY DEFINER;`)
}

func getMonitoredDBs(state *s.SetupState) ([]string, error) {
	key, err := state.CurrentSection.GetKey("db_name")
	if err != nil {
		return nil, err
	}
	dbs := key.Strings(",")
	if len(dbs) == 0 || dbs[0] == "" {
		return nil, errors.New("no databases found under db_name")
	}
	includesAll := dbs[len(dbs)-1] == "*"
	if !includesAll {
		return dbs, nil
	}

	// Expand the "*" entry here
	dbs = dbs[:len(dbs)-1]
	rows, err := state.QueryRunner.Query("SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate")
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		db := row.GetString(0)
		if !util.Includes(dbs, db) {
			dbs = append(dbs, db)
		}
	}
	return dbs, nil
}
