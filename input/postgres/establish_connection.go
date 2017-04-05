package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func EstablishConnection(server state.Server, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (connection *sql.DB, err error) {
	connection, err = connectToDb(server.Config, logger, globalCollectionOpts, databaseName)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && server.RequestedSslMode == "prefer" {
			server.Config.DbSslMode = "disable"
			connection, err = connectToDb(server.Config, logger, globalCollectionOpts, databaseName)
		}
	}

	if err != nil {
		return
	}

	validateConnectionCount(connection, logger, globalCollectionOpts)
	setStatementTimeout(connection, logger, server.Grant.Config.Features.StatementTimeoutMs)

	return
}

func connectToDb(config config.ServerConfig, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (*sql.DB, error) {
	connectString := config.GetPqOpenString(databaseName)

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

	connection.QueryRow(QueryMarkerSQL + "SELECT COUNT(*) FROM pg_stat_activity WHERE application_name = '" + globalCollectionOpts.CollectorApplicationName + "'").Scan(&connectionCount)

	if connectionCount > 5 {
		logger.PrintError("Too many open monitoring connections (current: %d, maximum allowed: 5), exiting", connectionCount)
		panic("Too many open monitoring connections")
	}

	return
}

func setStatementTimeout(connection *sql.DB, logger *util.Logger, statementTimeoutMs int32) {
	if statementTimeoutMs == 0 { // Default value
		statementTimeoutMs = 30000
	}

	// Assume anything below 100ms to be set in error - its not reasonable to have our queries run faster than that
	if statementTimeoutMs < 100 {
		logger.PrintVerbose("Ignoring invalid statement timeout of %dms (set it to at least 100ms)", statementTimeoutMs)
		return
	}

	connection.Exec(fmt.Sprintf("%sSET statement_timeout = %d", QueryMarkerSQL, statementTimeoutMs))

	return
}
