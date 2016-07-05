package runner

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func establishConnection(server state.Server, logger *util.Logger, globalCollectionOpts state.CollectionOpts) (connection *sql.DB, err error) {
	connection, err = connectToDb(server.Config, logger, globalCollectionOpts)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && server.RequestedSslMode == "prefer" {
			server.Config.DbSslMode = "disable"
			connection, err = connectToDb(server.Config, logger, globalCollectionOpts)
		}
	}

	if err != nil {
		return
	}

	validateConnectionCount(connection, logger, globalCollectionOpts)
	setStatementTimeout(connection, logger, globalCollectionOpts)

	return
}

func connectToDb(config config.ServerConfig, logger *util.Logger, globalCollectionOpts state.CollectionOpts) (*sql.DB, error) {
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

func validateConnectionCount(connection *sql.DB, logger *util.Logger, globalCollectionOpts state.CollectionOpts) {
	var connectionCount int

	connection.QueryRow(postgres.QueryMarkerSQL + "SELECT COUNT(*) FROM pg_stat_activity WHERE application_name = '" + globalCollectionOpts.CollectorApplicationName + "'").Scan(&connectionCount)

	if connectionCount > 1 {
		logger.PrintError("Too many open monitoring connections (%d), exiting", connectionCount)
		panic("Too many open monitoring connections")
	}

	return
}

func setStatementTimeout(connection *sql.DB, logger *util.Logger, globalCollectionOpts state.CollectionOpts) {
	// Assume anything below 100ms to be set in error - its not reasonable to have out queries run faster than that
	if globalCollectionOpts.StatementTimeoutMs < 100 {
		logger.PrintVerbose("Ignoring invalid statement timeout of %dms (set it to at least 100ms)", globalCollectionOpts.StatementTimeoutMs)
		return
	}

	connection.Exec(fmt.Sprintf("%sSET statement_timeout = %d", postgres.QueryMarkerSQL, globalCollectionOpts.StatementTimeoutMs))

	return
}
