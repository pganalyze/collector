package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
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
		return util.ValidateHelperFunction("explain", state.QueryRunner)
	},
	Run: func(state *s.SetupState) error {
		var doCreate bool
		if state.Inputs.Scripted {
			if !state.Inputs.CreateExplainHelper.Valid || !state.Inputs.CreateHelperFunctions.Bool {
				return errors.New("create_explain_helper flag not set and helper function does not exist or does not match expected signature")
			}
			doCreate = state.Inputs.CreateHelperFunctions.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Create (or update) EXPLAIN helper function (will be saved to Postgres)?",
				Default: false,
			}, &doCreate)
			if err != nil {
				return err
			}
		}

		if !doCreate {
			return nil
		}
		return state.QueryRunner.Exec(`CREATE OR REPLACE FUNCTION pganalyze.explain(query text, params text[]) RETURNS text AS
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
	},
}
