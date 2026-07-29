package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func EstablishConnection(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts, databaseName string) (connection *sql.DB, err error) {
	connection, err = connectToDb(ctx, server, logger, opts, databaseName)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && (server.Config.DbSslMode == "prefer" || server.Config.DbSslMode == "") {
			server.Config.DbSslModePreferFailed = true
			connection, err = connectToDb(ctx, server, logger, opts, databaseName)
		}
	}

	if err != nil {
		// The pinned address may be the reason we couldn't connect (the instance
		// went away, or was replaced), so let the next attempt resolve again. Only
		// network-level errors say anything about the address: a bad password or a
		// missing database must not drop the pin, since that could move later
		// connections onto a different instance.
		if addressRelatedError(err) {
			server.ResolvedHost.Invalidate()
		}
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

func connectToDb(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts, databaseName string) (*sql.DB, error) {
	var db *sql.DB
	var iamParams iamConnectionParams
	var err error

	config := server.Config

	driverName := "postgres"
	if config.DbUseIamAuth {
		driverName, iamParams, err = getIamConnectionParams(ctx, config)
		if err != nil {
			return nil, err
		}
	}

	connectString, err := config.GetPqOpenString(databaseName, iamParams.passwordOverride, iamParams.hostOverride, iamParams.sslmodeOverride)
	if err != nil {
		return nil, err
	}
	connectString += " application_name=" + opts.CollectorApplicationName

	db, err = openConnection(server, logger, driverName, connectString)
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

// openConnection - Opens the connection pool, routing its dials through the
// address this server is pinned to when we're connecting over the network
// ourselves. Resolution happens at dial time (see pinnedDialer), so the pool
// follows the pin for as long as it lives.
//
// Only the lib/pq driver is handled here. The Cloud SQL and AlloyDB drivers
// connect through a connector that is given an instance name rather than a
// hostname, so there is no DNS result of ours to share in the first place.
func openConnection(server *state.Server, logger *util.Logger, driverName string, connectString string) (*sql.DB, error) {
	host := server.Config.GetDbHost()
	if driverName != "postgres" || !canPinHost(host) {
		return sql.Open(driverName, connectString)
	}

	connector, err := pq.NewConnector(connectString)
	if err != nil {
		return nil, err
	}
	connector.Dialer(pinnedDialer{resolved: server.ResolvedHost, hostname: host, logger: logger})

	return sql.OpenDB(connector), nil
}

// addressRelatedError - Returns whether a connection error could plausibly be the
// fault of the address we dialed, as opposed to something the server told us over
// a working connection. Network-level failures and timeouts qualify; errors the
// Postgres protocol produced (authentication, a missing database) do not.
func addressRelatedError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
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
