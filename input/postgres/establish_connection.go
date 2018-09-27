package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func EstablishConnection(server state.Server, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (connection *sql.DB, err error) {
	connection, err = connectToDb(server.Config, logger, globalCollectionOpts, databaseName)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && (server.Config.DbSslMode == "prefer" || server.Config.DbSslMode == "") {
			server.Config.DbSslModePreferFailed = true
			connection, err = connectToDb(server.Config, logger, globalCollectionOpts, databaseName)
		}
	}

	if err != nil {
		return
	}

	err = validateConnectionCount(connection, logger, server.Config.MaxCollectorConnections, globalCollectionOpts)
	if err != nil {
		return
	}

	setStatementTimeout(connection, logger, server.Grant.Config.Features.StatementTimeoutMs)

	return
}

func connectToDb(config config.ServerConfig, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (*sql.DB, error) {
	connectString := config.GetPqOpenString(databaseName)
	connectString += " application_name=" + globalCollectionOpts.CollectorApplicationName

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

func validateConnectionCount(connection *sql.DB, logger *util.Logger, maxCollectorConnections int, globalCollectionOpts state.CollectionOpts) error {
	var connectionCount int

	connection.QueryRow(QueryMarkerSQL + "SELECT COUNT(*) FROM pg_stat_activity WHERE application_name = '" + globalCollectionOpts.CollectorApplicationName + "'").Scan(&connectionCount)

	if connectionCount > maxCollectorConnections {
		return fmt.Errorf("Too many open monitoring connections (current: %d, maximum allowed: %d), exiting", connectionCount, maxCollectorConnections)
	}

	return nil
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
