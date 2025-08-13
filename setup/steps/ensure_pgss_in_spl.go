package steps

import (
	"errors"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnablePgssInSpl = &state.Step{
	ID:          "ensure_pgss_in_spl",
	Description: "Ensure the pg_stat_statements extension is included in the shared_preload_libraries setting in Postgres",
	Check: func(s *state.SetupState) (bool, error) {
		spl, err := util.GetPendingSharedPreloadLibraries(s.QueryRunner)
		if err != nil {
			return false, err
		}

		return strings.Contains(spl, "pg_stat_statements"), nil
	},
	Run: func(s *state.SetupState) error {
		var doAdd bool
		if s.Inputs.Scripted {
			if !s.Inputs.EnsurePgStatStatementsLoaded.Valid || !s.Inputs.EnsurePgStatStatementsLoaded.Bool {
				return errors.New("enable_pg_stat_statements flag not set but pg_stat_statements not in shared_preload_libraries")
			}
			doAdd = s.Inputs.EnsurePgStatStatementsLoaded.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Add pg_stat_statements to shared_preload_libraries (will be saved to Postgres--requires restart in a later step)?",
				Default: false,
				Help:    "Postgres will have to be restarted in a later step to apply this configuration change; learn more about shared_preload_libraries here: https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SHARED-PRELOAD-LIBRARIES",
			}, &doAdd)
			if err != nil {
				return err
			}
		}
		if !doAdd {
			return nil
		}

		existingSpl, err := util.GetPendingSharedPreloadLibraries(s.QueryRunner)
		if err != nil {
			return err
		}

		var newSpl string
		if existingSpl == "" {
			newSpl = "pg_stat_statements"
		} else {
			newSpl = existingSpl + ",pg_stat_statements"
		}
		return util.ApplyConfigSetting("shared_preload_libraries", newSpl, s.QueryRunner)
	},
}
