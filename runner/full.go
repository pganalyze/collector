package runner

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/snapshot"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func collectStatistics(db state.Database, collectionOpts state.CollectionOpts, logger *util.Logger) (s state.State, err error) {
	var explainInputs []state.PostgresExplainInput

	postgresVersion, err := postgres.GetPostgresVersion(logger, db.Connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	/*stats.Postgres = &snapshot.SnapshotPostgres{}
	stats.Postgres.Version = &postgresVersion*/

	if postgresVersion.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", postgresVersion.Short)
		return
	}

	s.Backends, err = postgres.GetBackends(logger, db.Connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	s.Statements, err = postgres.GetStatements(logger, db.Connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_statements")
		return
	}

	if collectionOpts.CollectPostgresRelations {
		s.Relations, err = postgres.GetRelations(db.Connection, postgresVersion, collectionOpts.CollectPostgresBloat)
		if err != nil {
			logger.PrintError("Error collecting schema information")
			return
		}
	}

	if collectionOpts.CollectPostgresSettings {
		s.Settings, err = postgres.GetSettings(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		s.Functions, err = postgres.GetFunctions(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return
		}
	}

	if collectionOpts.CollectSystemInformation {
		systemState := system.GetSystemState(db.Config, logger)
		s.System = &systemState
	}

	if collectionOpts.CollectLogs {
		s.Logs, explainInputs = system.GetLogLines(db.Config)

		if collectionOpts.CollectExplain {
			s.Explains = postgres.RunExplain(db.Connection, explainInputs)
		}
	}

	return
}

func diffState(logger *util.Logger, prevState state.State, newState state.State) (diffState state.DiffState) {
	//if postgresVersion.Numeric >= state.PostgresVersion94 {

	// Iterate through all statements, and diff them based on the previous state
	for key, statement := range newState.Statements {
		var diffedStatement state.DiffedPostgresStatement

		prevStatement, exists := prevState.Statements[key]
		if exists {
			diffedStatement = statement.DiffSince(prevStatement)
		} else if len(prevState.Statements) > 0 { // New statement since the last run
			diffedStatement = statement.DiffSince(state.PostgresStatement{})
		}

		if diffedStatement.Calls > 0 {
			diffState.Statements = append(diffState.Statements, diffedStatement)
		}
	}

	return
}

func performOutput(db state.Database, collectionOpts state.CollectionOpts, logger *util.Logger, newState state.State, diffState state.DiffState) (err error) {
	var queryInformations []*snapshot.QueryInformation

	logger.PrintInfo("Diff: %+v", diffState.Statements)

	for _, statement := range diffState.Statements {
		var queryInformation snapshot.QueryInformation

		queryInformation.NormalizedQuery = statement.NormalizedQuery

		queryInformations = append(queryInformations, &queryInformation)
	}

	// FIXME: Need to transform state into snapshot
	statsProto, err := proto.Marshal(&snapshot.Snapshot{})
	if err != nil {
		logger.PrintError("Error marshaling statistics")
		return
	}

	if !collectionOpts.SubmitCollectedData {
		statsReRead := &snapshot.Snapshot{}
		if err = proto.Unmarshal(statsProto, statsReRead); err != nil {
			log.Fatalln("Failed to re-read stats:", err)
		}

		var out bytes.Buffer
		statsJSON, _ := json.Marshal(statsReRead)
		json.Indent(&out, statsJSON, "", "\t")
		logger.PrintInfo("Dry run - data that would have been sent will be output on stdout:\n")
		fmt.Print(out.String())
		return
	}

	var compressedJSON bytes.Buffer
	w := zlib.NewWriter(&compressedJSON)
	w.Write(statsProto)
	w.Close()

	requestURL := db.Config.APIBaseURL + "/v1/snapshots"

	if collectionOpts.TestRun {
		requestURL = db.Config.APIBaseURL + "/v1/snapshots/test"
	}

	data := url.Values{
		"data":            {compressedJSON.String()},
		"data_compressor": {"zlib"},
		"api_key":         {db.Config.APIKey},
		"submitter":       {"pganalyze-collector 0.9.0rc7"},
		"no_reset":        {"true"},
		"query_source":    {"pg_stat_statements"},
		"collected_at":    {fmt.Sprintf("%d", time.Now().Unix())},
	}

	encodedData := data.Encode()

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(encodedData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	logger.PrintVerbose("Successfully prepared request - size of request body: %.4f MB", float64(len(encodedData))/1024.0/1024.0)

	resp, err := http.DefaultClient.Do(req)
	// TODO: We could consider re-running on error (e.g. if it was a temporary server issue)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Error when submitting: %s\n", body)
		return
	}

	if len(body) > 0 {
		logger.PrintInfo("%s", body)
	} else {
		logger.PrintInfo("Submitted snapshot successfully")
	}

	return
}

func processDatabase(db state.Database, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.State, error) {
	newState, err := collectStatistics(db, globalCollectionOpts, logger)
	if err != nil {
		return newState, err
	}

	diffState := diffState(logger, db.PrevState, newState)

	performOutput(db, globalCollectionOpts, logger, newState, diffState)

	return newState, nil
}

func CollectAllDatabases(databases []state.Database, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for idx, db := range databases {
		var err error

		prefixedLogger := logger.WithPrefix(db.Config.SectionName)

		db.Connection, err = establishConnection(db, logger, globalCollectionOpts)
		if err != nil {
			prefixedLogger.PrintError("Error: Failed to connect to database: %s", err)
			return
		}

		newState, err := processDatabase(db, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Error: Could not process database: %s", err)
		} else {
			databases[idx].PrevState = newState
		}

		// This is the easiest way to avoid opening multiple connections to different databases on the same instance
		db.Connection.Close()
		db.Connection = nil
	}
}

func validateConnectionCount(connection *sql.DB, logger *util.Logger, globalCollectionOpts state.CollectionOpts) {
	var connectionCount int

	connection.QueryRow(postgres.QueryMarkerSQL + "SELECT COUNT(*) FROM pg_stat_activity WHERE application_name = '" + globalCollectionOpts.CollectorApplicationName + "'").Scan(&connectionCount)

	if connectionCount > 1 {
		logger.PrintError("Too many open monitoring connections (%d), exiting", connectionCount)
		panic("Too many open monitoring connections")
	}

	return
}

func connectToDb(config config.DatabaseConfig, logger *util.Logger, globalCollectionOpts state.CollectionOpts) (*sql.DB, error) {
	connectString := config.GetPqOpenString()

	if strings.HasPrefix(connectString, "postgres://") || strings.HasPrefix(connectString, "postgresql://") {
		if strings.Contains(connectString, "?") {
			connectString += "&"
		} else {
			connectString += "?"
		}
	} else {
		connectString += " "
	}
	connectString += "application_name=" + globalCollectionOpts.CollectorApplicationName

	// logger.PrintVerbose("sql.Open(\"postgres\", \"%s\")", connectString)

	db, err := sql.Open("postgres", connectString)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(30 * time.Second)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func establishConnection(db state.Database, logger *util.Logger, globalCollectionOpts state.CollectionOpts) (connection *sql.DB, err error) {
	connection, err = connectToDb(db.Config, logger, globalCollectionOpts)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && db.RequestedSslMode == "prefer" {
			db.Config.DbSslMode = "disable"
			connection, err = connectToDb(db.Config, logger, globalCollectionOpts)
		}
	}

	if err != nil {
		return
	}

	validateConnectionCount(connection, logger, globalCollectionOpts)

	return
}
