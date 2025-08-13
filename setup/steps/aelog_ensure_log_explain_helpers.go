package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureLogExplainHelpers = &state.Step{
	Kind:        state.AutomatedExplainStep,
	ID:          "aelog_ensure_log_explain_helpers",
	Description: "Ensure EXPLAIN helper functions for log-based EXPLAIN exist in all monitored Postgres databases",
	Check: func(s *state.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(s.CurrentSection)
		if err != nil {
			return false, err
		}
		if !logExplain {
			return true, nil
		}
		monitoredDBs, err := getMonitoredDBs(s)
		if err != nil {
			return false, err
		}

		for _, db := range monitoredDBs {
			dbRunner := s.QueryRunner.InDB(db)
			isValid, err := util.ValidateHelperFunction(util.ExplainHelper, dbRunner)
			if !isValid || err != nil {
				return isValid, err
			}
		}
		return true, nil
	},
	Run: func(s *state.SetupState) error {
		var doCreate bool
		if s.Inputs.Scripted {
			if !s.Inputs.EnsureLogExplainHelpers.Valid || !s.Inputs.EnsureLogExplainHelpers.Bool {
				return errors.New("create_explain_helper flag not set and helper function does not exist or does not match expected signature on all monitored databases")
			}
			doCreate = s.Inputs.EnsureLogExplainHelpers.Bool
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
		monitoredDBs, err := getMonitoredDBs(s)
		if err != nil {
			return err
		}
		for _, db := range monitoredDBs {
			err := createHelperInDB(s, db)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func createHelperInDB(s *state.SetupState, db string) error {
	dbRunner := s.QueryRunner.InDB(db)
	isValid, err := util.ValidateHelperFunction(util.ExplainHelper, dbRunner)
	if err != nil {
		return err
	}
	if isValid {
		return nil
	}
	userKey, err := s.CurrentSection.GetKey("db_username")
	if err != nil {
		return err
	}
	pgaUser := userKey.String()

	return dbRunner.Exec(
		fmt.Sprintf(
			`CREATE SCHEMA IF NOT EXISTS pganalyze; GRANT USAGE ON SCHEMA pganalyze TO %s;`,
			pq.QuoteIdentifier(pgaUser),
		) + util.ExplainHelper.GetDefinition(),
	)
}

func getMonitoredDBs(s *state.SetupState) ([]string, error) {
	key, err := s.CurrentSection.GetKey("db_name")
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
	rows, err := s.QueryRunner.Query("SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate")
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
