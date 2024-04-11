package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
)

func EstablishConnection(ctx context.Context, server *state.Server, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (connection *sql.DB, err error) {
	connection, err = connectToDb(ctx, server.Config, logger, globalCollectionOpts, databaseName)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && (server.Config.DbSslMode == "prefer" || server.Config.DbSslMode == "") {
			server.Config.DbSslModePreferFailed = true
			connection, err = connectToDb(ctx, server.Config, logger, globalCollectionOpts, databaseName)
		}
	}

	if err != nil {
		return
	}

	err = validateConnectionCount(ctx, connection, logger, server.Config.MaxCollectorConnections, globalCollectionOpts)
	if err != nil {
		connection.Close()
		return
	}

	err = SetDefaultStatementTimeout(ctx, connection, logger, server)
	if err != nil {
		connection.Close()
		return
	}

	return
}

func connectToDb(ctx context.Context, config config.ServerConfig, logger *util.Logger, globalCollectionOpts state.CollectionOpts, databaseName string) (*sql.DB, error) {
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
			fmt.Sprintf("%s:%d", config.GetDbHost(), config.GetDbPortOrDefault()),
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

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func validateConnectionCount(ctx context.Context, connection *sql.DB, logger *util.Logger, maxCollectorConnections int, globalCollectionOpts state.CollectionOpts) error {
	var connectionCount int

	err := connection.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.count(*) FROM pg_catalog.pg_stat_activity WHERE application_name = '"+globalCollectionOpts.CollectorApplicationName+"'").Scan(&connectionCount)
	if err != nil {
		return err
	}

	if connectionCount > maxCollectorConnections {
		return fmt.Errorf("Too many open monitoring connections (current: %d, maximum allowed: %d), exiting", connectionCount, maxCollectorConnections)
	}

	return nil
}

func SetStatementTimeout(ctx context.Context, connection *sql.DB, statementTimeoutMs int32) error {
	_, err := connection.ExecContext(ctx, fmt.Sprintf("%sSET statement_timeout = %d", QueryMarkerSQL, statementTimeoutMs))
	if err != nil {
		return err
	}

	return nil
}

func SetDefaultStatementTimeout(ctx context.Context, connection *sql.DB, logger *util.Logger, server *state.Server) error {
	statementTimeoutMs := server.Grant.Config.Features.StatementTimeoutMs
	if statementTimeoutMs == 0 { // Default value
		statementTimeoutMs = 30000
	}

	// Assume anything below 100ms to be set in error - its not reasonable to have our queries run faster than that
	if statementTimeoutMs < 100 {
		logger.PrintVerbose("Ignoring invalid statement timeout of %dms (set it to at least 100ms)", statementTimeoutMs)
		return nil
	}

	err := SetStatementTimeout(ctx, connection, statementTimeoutMs)
	if err != nil {
		return err
	}

	return nil
}

func SetQueryTextStatementTimeout(ctx context.Context, connection *sql.DB, logger *util.Logger, server *state.Server) error {
	queryTextStatementTimeoutMs := server.Grant.Config.Features.StatementTimeoutMsQueryText
	if queryTextStatementTimeoutMs == 0 { // Default value
		queryTextStatementTimeoutMs = 120000
	}

	err := SetStatementTimeout(ctx, connection, queryTextStatementTimeoutMs)
	if err != nil {
		return err
	}

	return nil
}
