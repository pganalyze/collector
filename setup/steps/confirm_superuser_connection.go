package steps

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/state"
)

var ConfirmSuperuserConnection = &state.Step{
	ID:          "confirm_superuser_connection",
	Description: "Confirm the Postgres superuser connection to use only for this guided setup session",
	Check: func(s *state.SetupState) (bool, error) {
		if s.QueryRunner == nil {
			return false, nil
		}
		err := s.QueryRunner.PingSuper()
		return err == nil, err
	},
	Run: func(s *state.SetupState) error {
		localPgs, err := discoverLocalPgFromUnixSockets()
		if err != nil {
			return err
		}
		var selectedPg LocalPostgres
		if len(localPgs) == 0 {
			return errors.New("failed to find a running local Postgres install")
		} else if len(localPgs) > 1 {
			return errors.New("found multiple local Postgres installs; this is not supported for guided setup")
		} else {
			selectedPg = localPgs[0]
		}
		if s.Inputs.Scripted {
			if selectedPg.Port != 0 {
				// skip finding the server if there's only one, but validate it matches config, if present
				if (s.Inputs.PGSetupConnPort.Valid && int(s.Inputs.PGSetupConnPort.Int64) != selectedPg.Port) ||
					(s.Inputs.PGSetupConnSocketDir.Valid && s.Inputs.PGSetupConnSocketDir.String != selectedPg.SocketDir) {
					// just clear the selection and depend on error handling below
					selectedPg = LocalPostgres{}
				}
			} else {
				if !s.Inputs.PGSetupConnPort.Valid {
					return errors.New("no port specified for setup Postgres connection")
				}
				for _, pg := range localPgs {
					if int(s.Inputs.PGSetupConnPort.Int64) == pg.Port &&
						(!s.Inputs.PGSetupConnSocketDir.Valid ||
							s.Inputs.PGSetupConnSocketDir.String == pg.SocketDir) {
						selectedPg = pg
						break
					}
				}
			}
			if selectedPg.Port == 0 {
				var portStr string
				if s.Inputs.PGSetupConnPort.Valid {
					portStr = " on " + strconv.Itoa(int(s.Inputs.PGSetupConnPort.Int64))
				}
				var socketDirStr string
				if s.Inputs.PGSetupConnSocketDir.Valid {
					socketDirStr = " in " + s.Inputs.PGSetupConnSocketDir.String
				}

				return fmt.Errorf("no Postgres server found listening%s%s", portStr, socketDirStr)
			}
		} else {
			if selectedPg.Port == 0 {
				var opts []string
				for _, localPg := range localPgs {
					opts = append(opts, fmt.Sprintf("port %d in socket dir %s", localPg.Port, localPg.SocketDir))
				}
				var selectedIdx int
				err := survey.AskOne(&survey.Select{
					Message: "Found several Postgres installations; please select one",
					Options: opts,
				}, &selectedIdx)
				if err != nil {
					return err
				}
				selectedPg = localPgs[selectedIdx]
			}
		}

		var pgSuperuser string
		if s.Inputs.Scripted {
			if !s.Inputs.PGSetupConnUser.Valid {
				return errors.New("no user specified for setup Postgres connection")
			}
			pgSuperuser = s.Inputs.PGSetupConnUser.String
		} else {
			err = survey.AskOne(&survey.Select{
				Message: "Select Postgres superuser to connect as for initial setup:",
				Help:    "We will create a separate, restricted monitoring user for the collector later",
				Options: []string{"postgres", "another user..."},
			}, &pgSuperuser)
			if err != nil {
				return err
			}
			if pgSuperuser != "postgres" {
				err = survey.AskOne(&survey.Input{
					Message: "Enter Postgres superuser to connect as for initial setup:",
					Help:    "We will create a separate, restricted monitoring user for the collector later",
				}, &pgSuperuser, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			}
		}

		s.QueryRunner = query.NewRunner(pgSuperuser, selectedPg.SocketDir, selectedPg.Port)

		return nil
	},
}

type LocalPostgres struct {
	SocketDir string
	LocalAddr string
	Port      int
}

var pgsqlDomainSocketPortRe = regexp.MustCompile("\\d+$")

func getSocketDirMatches(dir string) ([]LocalPostgres, error) {
	var result []LocalPostgres
	globPattern := filepath.Join(dir, ".s.PGSQL.*")
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		portStr := pgsqlDomainSocketPortRe.FindString(match)
		if portStr == "" {
			continue
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
		result = append(result, LocalPostgres{SocketDir: dir, Port: port})
	}
	return result, nil
}

func discoverLocalPgFromUnixSockets() ([]LocalPostgres, error) {
	varRunMatches, err := getSocketDirMatches("/var/run/postgresql")
	if err != nil {
		return nil, err
	}
	tmpMatches, err := getSocketDirMatches("/tmp")
	if err != nil {
		return nil, err
	}
	var result []LocalPostgres
	return append(append(result, varRunMatches...), tmpMatches...), nil
}
