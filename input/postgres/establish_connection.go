package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
)

func EstablishConnection(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts, databaseName string) (connection *sql.DB, err error) {
	connection, err = connectToDb(ctx, server.Config, logger, opts, databaseName)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && (server.Config.DbSslMode == "prefer" || server.Config.DbSslMode == "") {
			server.Config.DbSslModePreferFailed = true
			connection, err = connectToDb(ctx, server.Config, logger, opts, databaseName)
		}
	}

	if err != nil {
		return
	}

	err = validateConnectionCount(ctx, connection, logger, server.Config.MaxCollectorConnections, opts)
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

func connectToDb(ctx context.Context, config config.ServerConfig, logger *util.Logger, opts state.CollectionOpts, databaseName string) (*sql.DB, error) {
	var dbPasswordOverride string
	var hostOverride string
	var sslmodeOverride string
	var db *sql.DB
	driverName := "postgres"

	if config.DbUseIamAuth {
		if config.SystemType == "amazon_rds" {
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
		} else if config.SystemType == "google_cloudsql" {
			if config.GcpCloudSQLInstanceID != "" {
				hostOverride = strings.Join([]string{config.GcpProjectID, config.GcpRegion, config.GcpCloudSQLInstanceID}, ":")
			} else {
				hostOverride = fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", config.GcpProjectID, config.GcpRegion, config.GcpAlloyDBClusterID, config.GcpAlloyDBInstanceID)
			}
			// When using cloud-sql-go-connector, this needs to be set as disable
			// https://github.com/GoogleCloudPlatform/cloud-sql-go-connector/issues/889
			sslmodeOverride = "disable"
			if config.GcpCloudSQLInstanceID != "" {
				if config.GcpUsePublicIP {
					driverName = "cloudsql-postgres-public"
				} else {
					driverName = "cloudsql-postgres"
				}
			} else if config.GcpAlloyDBClusterID != "" || config.GcpAlloyDBInstanceID != "" {
				if config.GcpUsePublicIP {
					driverName = "alloydb-postgres-public"
				} else {
					driverName = "alloydb-postgres"
				}
			} else {
				return nil, errors.New("To use IAM auth with either Google Cloud SQL or AlloyDB, you must specify project ID, region, and then either the instance ID (CloudSQL) or cluster ID and instance ID (AlloyDB) in the configuration")
			}
		} else {
			return nil, errors.New("IAM auth is only supported for Amazon RDS, Aurora, Google Cloud SQL, and Google AlloyDB - turn off IAM auth setting to use password-based authentication")
		}
	}

	connectString, err := config.GetPqOpenString(databaseName, dbPasswordOverride, hostOverride, sslmodeOverride)
	if err != nil {
		return nil, err
	}
	connectString += " application_name=" + opts.CollectorApplicationName

	db, err = sql.Open(driverName, connectString)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func validateConnectionCount(ctx context.Context, connection *sql.DB, logger *util.Logger, maxCollectorConnections int, opts state.CollectionOpts) error {
	var connectionCount int

	err := connection.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.count(*) FROM pg_catalog.pg_stat_activity WHERE application_name = '"+opts.CollectorApplicationName+"'").Scan(&connectionCount)
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
	statementTimeoutMs := server.Grant.Load().Config.Features.StatementTimeoutMs
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
	queryTextStatementTimeoutMs := server.Grant.Load().Config.Features.StatementTimeoutMsQueryText
	if queryTextStatementTimeoutMs == 0 { // Default value
		queryTextStatementTimeoutMs = 120000
	}

	err := SetStatementTimeout(ctx, connection, queryTextStatementTimeoutMs)
	if err != nil {
		return err
	}

	return nil
}
