package steps

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/query"
	s "github.com/pganalyze/collector/setup/state"
)

var EstablishSuperuserConnection = &s.Step{
	Description: "Ensure Postgres superuser connection",
	Check: func(state *s.SetupState) (bool, error) {
		if state.QueryRunner == nil {
			return false, nil
		}
		err := state.QueryRunner.PingSuper()
		return err == nil, err
	},
	Run: func(state *s.SetupState) error {
		localPgs, err := discoverLocalPostgres()
		if err != nil {
			return err
		}
		var selectedPg LocalPostgres
		if state.Inputs.Scripted {
			if !state.Inputs.PGSetupConnPort.Valid {
				return errors.New("no port specified for setup Postgres connection")
			}
			for _, pg := range localPgs {
				if int(state.Inputs.PGSetupConnPort.Int64) == pg.Port &&
					(!state.Inputs.PGSetupConnSocketDir.Valid ||
						state.Inputs.PGSetupConnSocketDir.String == pg.SocketDir) {
					selectedPg = pg
					break
				}
			}
			if selectedPg.Port == 0 {
				var socketDirStr string
				if state.Inputs.PGSetupConnSocketDir.Valid {
					socketDirStr = " in " + state.Inputs.PGSetupConnSocketDir.String
				}

				return fmt.Errorf(
					"no Postgres server found listening on %d%s",
					state.Inputs.PGSetupConnPort.Int64,
					socketDirStr,
				)
			}
		} else {
			if len(localPgs) == 0 {
				return errors.New("failed to find a running local Postgres install")
			} else if len(localPgs) == 1 {
				selectedPg = localPgs[0]
			} else {
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
		if state.Inputs.Scripted {
			if !state.Inputs.PGSetupConnUser.Valid {
				return errors.New("no user specified for setup Postgres connection")
			}
			pgSuperuser = state.Inputs.PGSetupConnUser.String
		} else {
			err = survey.AskOne(&survey.Select{
				Message: "Select Postgres superuser to connect as for configuration purposes",
				Help:    "We will create a separate, restricted monitoring user for the collector later",
				Options: []string{"postgres", "another user..."},
			}, &pgSuperuser)
			if err != nil {
				return err
			}
			if pgSuperuser != "postgres" {
				err = survey.AskOne(&survey.Input{
					Message: "Enter Postgres superuser to connect as for configuration purposes",
					Help:    "We will create a separate, restricted monitoring user for the collector later",
				}, &pgSuperuser, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			}
		}

		state.QueryRunner = query.NewRunner(pgSuperuser, selectedPg.SocketDir, selectedPg.Port)

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
	// technically this should be a filepath.Join, but Unix-domain sockets do not work
	// on windows anyway
	globPattern := fmt.Sprintf("%s/.s.PGSQL.*", dir)
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

func discoverLocalPostgres() ([]LocalPostgres, error) {
	// TODO: find tcp sockets if no unix sockets?
	// TODO: confirm these are live by checking pids?
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
