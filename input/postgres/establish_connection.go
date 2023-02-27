package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
)

func EstablishConnection(server *state.Server, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (connection *sql.DB, err error) {
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
		connection.Close()
		return
	}

	SetDefaultStatementTimeout(connection, logger, server)

	return
}

func connectToDb(config config.ServerConfig, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (*sql.DB, error) {
	var dbPasswordOverride string

	if config.DbUseIamAuth {
		if config.SystemType != "amazon_rds" {
			return nil, fmt.Errorf("IAM auth is only supported for Amazon RDS and Aurora - turn off IAM auth setting to use password-based authentication")
		}
		sess, err := awsutil.GetAwsSession(config)
		if err != nil {
			return nil, err
		}
		if dbToken, err := rdsutils.BuildAuthToken(
			fmt.Sprintf("%s:%d", config.GetDbHost(), config.GetDbPort()),
			config.AwsRegion,
			config.GetDbUsername(),
			sess.Config.Credentials,
		); err != nil {
			return nil, err
		} else {
			dbPasswordOverride = dbToken
		}
	}

	connectString, err := config.GetPqOpenString(databaseName, dbPasswordOverride)
	if err != nil {
		return nil, err
	}
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
		db.Close()
		return nil, err
	}

	return db, nil
}

func validateConnectionCount(connection *sql.DB, logger *util.Logger, maxCollectorConnections int, globalCollectionOpts state.CollectionOpts) error {
	var connectionCount int

	connection.QueryRow(QueryMarkerSQL + "SELECT pg_catalog.count(*) FROM pg_catalog.pg_stat_activity WHERE application_name = '" + globalCollectionOpts.CollectorApplicationName + "'").Scan(&connectionCount)

	if connectionCount > maxCollectorConnections {
		return fmt.Errorf("Too many open monitoring connections (current: %d, maximum allowed: %d), exiting", connectionCount, maxCollectorConnections)
	}

	return nil
}

func SetStatementTimeout(connection *sql.DB, statementTimeoutMs int32) {
	connection.Exec(fmt.Sprintf("%sSET statement_timeout = %d", QueryMarkerSQL, statementTimeoutMs))

	return
}

func SetDefaultStatementTimeout(connection *sql.DB, logger *util.Logger, server *state.Server) {
	statementTimeoutMs := server.Grant.Config.Features.StatementTimeoutMs
	if statementTimeoutMs == 0 { // Default value
		statementTimeoutMs = 30000
	}

	// Assume anything below 100ms to be set in error - its not reasonable to have our queries run faster than that
	if statementTimeoutMs < 100 {
		logger.PrintVerbose("Ignoring invalid statement timeout of %dms (set it to at least 100ms)", statementTimeoutMs)
		return
	}

	SetStatementTimeout(connection, statementTimeoutMs)

	return
}

func SetQueryTextStatementTimeout(connection *sql.DB, logger *util.Logger, server *state.Server) {
	queryTextStatementTimeoutMs := server.Grant.Config.Features.StatementTimeoutMsQueryText
	if queryTextStatementTimeoutMs == 0 { // Default value
		queryTextStatementTimeoutMs = 120000
	}

	SetStatementTimeout(connection, queryTextStatementTimeoutMs)

	return
}
