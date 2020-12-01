package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/guregu/null"
	flag "github.com/ogier/pflag"

	survey "github.com/AlecAivazis/survey/v2"

	"github.com/pganalyze/collector/setup/log"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/steps"
)

const defaultConfigFile = "/etc/pganalyze-collector.conf"

func main() {
	steps := []*s.Step{
		steps.CheckPlatform,
		steps.DoLoadConfig,
		steps.AskSuperuserConnection,
		steps.CheckPostgresVersion,
		steps.CheckReplicationStatus,
		steps.CheckRestartNeeded,
		steps.SpecifyAPIKey,
		steps.SpecifyDatabases,
		steps.SpecifyMonitoringUser,
		steps.EnsureMonitoringUser,
		steps.SpecifyMonitoringUserPasswd,
		steps.EnsureMonitoringUserPassword,
		steps.EnsureMonitoringUserPermissions,
		steps.EnsurePganalyzeSchema,
		steps.ConfirmPgssAvailable,
		steps.EnsurePgssExtInstalled,
		steps.EnablePgssInSpl,

		steps.AskSetUpLogInsights,
		steps.EnsureSupportedLogErrorVerbosity,
		steps.EnsureSupportedLogDuration,
		steps.EnsureSupportedLogStatement,
		steps.ConfigureLogMinDurationStatement,
		steps.EnsureSupportedLogLinePrefix,
		steps.SpecifyDbLogLocation,

		steps.AskSetUpAutoExplain,
		steps.AskUseLogBasedExplain,
		steps.EnsureLogExplainHelpers,
		steps.ConfirmAutoExplainAvailable,
		steps.EnsureAutoExplainInSpl,

		steps.ConfirmReloadCollector,
		steps.ConfirmRestartPostgres,
		steps.EnsureRecommendedAutoExplainSettings,
		steps.ConfirmTestCommand,
	}

	var setupState state.SetupState
	var quiet bool
	var logFile string
	var inputsFile string
	var recommended bool
	var apiKey string
	var apiBaseURL string
	var dbName string
	flag.StringVar(&setupState.ConfigFilename, "config", defaultConfigFile, "specify alternative path for config file")
	flag.StringVar(&apiKey, "api-key", "", "pganalyze API key")
	flag.StringVar(&apiBaseURL, "api-base-url", "", "pganalyze API base URL")
	flag.StringVar(&dbName, "db-name", "", "database name to monitor")
	flag.BoolVar(&quiet, "quiet", false, "omit verbose logging output")
	flag.StringVar(&logFile, "log", "", "save output to log file (always includes verbose output)")
	flag.StringVar(&inputsFile, "inputs", "", "do not prompt for user inputs and use JSON file describing answers to all setup prompts")
	flag.BoolVar(&recommended, "recommended", false, "do not prompt for user inputs and use recommended values (the --inputs flag can override individual settings)")
	flag.Parse()

	logger := log.NewLogger()
	if logFile == "" {
		if !quiet {
			logger.VerboseOutput = os.Stdout
		}
	} else {
		log, err := os.Create(logFile)
		if err != nil {
			fmt.Printf("ERROR: could not open log file %s for writes: %s\n", logFile, err)
			os.Exit(1)
		}
		defer log.Close()
		outputBoth := io.MultiWriter(os.Stdout, log)
		logger.StandardOutput = outputBoth
		if quiet {
			logger.VerboseOutput = log
		} else {
			logger.VerboseOutput = outputBoth
		}
	}
	setupState.Logger = &logger

	var inputs state.SetupInputs
	if recommended {
		inputs = s.RecommendedInputs
	}
	if inputsFile != "" {
		inputsReader, err := os.Open(inputsFile)
		if err != nil {
			setupState.Log("ERROR: could not open inputs file %s: %s", logFile, err)
			os.Exit(1)
		}
		inputsBytes, err := ioutil.ReadAll(inputsReader)
		if err != nil {
			setupState.Log("ERROR: could not open inputs file %s: %s", logFile, err)
			os.Exit(1)
		}
		err = json.Unmarshal(inputsBytes, &inputs)
		if err != nil {
			setupState.Log("ERROR: could not parse inputs file %s: %s", logFile, err)
			os.Exit(1)
		}
		inputs.Scripted = true
		err = os.Stdin.Close()
		if err != nil {
			setupState.Log("ERROR: could not close stdin for scripted input: %s", err)
			os.Exit(1)
		}
	}
	if apiKey != "" {
		inputs.Settings.APIKey = null.StringFrom(apiKey)
	}
	if apiBaseURL != "" {
		inputs.Settings.APIBaseURL = null.StringFrom(apiBaseURL)
	}
	if dbName != "" {
		inputs.Settings.DBName = null.StringFrom(dbName)
	}

	setupState.Inputs = &inputs

	id := os.Geteuid()
	if id > 0 {
		setupState.Log(`ERROR: The pganalyze collector guided setup must be run as root (or with sudo, if available)

It will provide details on the process and prompt you before making any changes to the
collector config file or your database. If you prefer, you can instead follow
the manual collector install instructions: https://pganalyze.com/docs/install`)
		os.Exit(1)
	}

	setupState.Log(`Welcome to the pganalyze collector guided setup!

IMPORTANT: Please note that this setup only works when monitoring a self-managed system,
and installing the collector directly on your database server. For other setup types,
please check the installation instructions https://pganalyze.com/docs/install .

We will go through a series of steps to set up the collector to monitor your
Postgres server. We will not make any changes to Postgres or your system
without confirmation.

Before we do anything else, we will check whether a Postgres restart is necessary
to enable any features. This will happen before making any changes to your system.
You can abort the process if a restart would be required but is not feasible.

After that, at a high level, we will:

 1. Configure database access and, if necessary, create the pganalyze database user with monitoring-only access
 2. Update the collector configuration file
 3. Set up the pg_stat_statements extension in your database for query performance monitoring
 4. (Optional) Change log-related configuration settings to enable the pganalyze Log Insights feature
 5. (Optional) Set up EXPLAIN plan collection to enable the pganalyze Automated EXPLAIN feature

At each step, we'll check if any changes are necessary, and if so, prompt you to
provide input or confirm any required changes.

Changes to Postgres configuration settings will be done with the ALTER SYSTEM command.
If you later need to refine any of these, make sure to use ALTER SYSTEM or ALTER SYSTEM RESET,
since otherwise, the ALTER SYSTEM changes will override any direct config file edits. Learn
more at https://www.postgresql.org/docs/current/sql-altersystem.html .

You can stop at any time by pressing Ctrl+C.

If you stop before completing setup, you can resume by running the guided setup
again. We can pick up where you left off.`)
	setupState.Log("")
	if !setupState.Inputs.Scripted {
		var doSetup bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Continue with setup?",
			Default: false,
		}, &doSetup)
		if err != nil {
			setupState.Log("  automated setup failed: %s", err)
		}
		if !doSetup {
			setupState.Log("Exiting...")
			os.Exit(0)
		}
	}

	for _, step := range steps {
		skipLogInsights := setupState.Inputs.SkipLogInsights.Valid && setupState.Inputs.SkipLogInsights.Bool
		skipAutomatedExplain := setupState.Inputs.SkipLogInsights.Valid && setupState.Inputs.SkipLogInsights.Bool
		if step.Kind == state.LogInsightsStep && skipLogInsights {
			continue
		}
		if step.Kind == state.AutomatedExplainStep && (skipLogInsights || skipAutomatedExplain) {
			continue
		}

		err := doStep(&setupState, step)
		if err != nil {
			if setupState.NeedsReload {
				setupState.Log(`
WARNING: Exiting with pending changes to collector config.

Please run pganalyze-collector --reload to apply these changes.`)
			}
			os.Exit(1)
		}
	}
	setupState.Log(`
Collector setup complete!

For next steps, go back to the pganalyze app in your web browser.`)
}

func doStep(setupState *s.SetupState, step *s.Step) error {
	if step.Check == nil {
		panic("step missing completion check")
	}
	setupState.Logger.StartStep(step.Description)
	defer setupState.Logger.EndStep()
	done, err := step.Check(setupState)
	if err != nil {
		setupState.Log("✗ step check failed: %s", err)
		return err
	}
	if done {
		setupState.Verbose("✓ no changes needed")
		return nil
	}
	if step.Run == nil {
		// panic because we should always define a Run func if a check does not
		// pass but there is no fatal error
		panic("check failed and no resolution defined")
	}
	setupState.Verbose("? attempting resolution")

	err = step.Run(setupState)
	if err != nil {
		setupState.Log("✗ step failed: %s", err)
		return err
	}

	setupState.Verbose("  re-checking...")
	done, err = step.Check(setupState)
	if err != nil {
		setupState.Log("✗ step check failed: %s", err)
		return err
	}
	if !done {
		err := errors.New("check still failed after running resolution; please try again")
		setupState.Log("✗ %s", err)
		return err
	}
	setupState.Verbose("✓ step completed")
	return nil
}
