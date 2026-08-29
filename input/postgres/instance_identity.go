package postgres

import (
	"context"
	"database/sql"

	"github.com/pganalyze/collector/state"
)

// Both functions are unprivileged built-ins available on all supported versions.
// inet_server_addr() returns NULL over Unix sockets, and host() renders it
// without the netmask that casting an inet to text would include.
const instanceIdentitySQL string = `
SELECT pg_catalog.pg_postmaster_start_time(),
	COALESCE(pg_catalog.host(pg_catalog.inet_server_addr()), '')`

// GetInstanceIdentity - Determines which Postgres instance, and which postmaster
// on it, this connection is talking to.
func GetInstanceIdentity(ctx context.Context, db *sql.DB) (state.PostgresInstanceIdentity, error) {
	var identity state.PostgresInstanceIdentity
	err := db.QueryRowContext(ctx, QueryMarkerSQL+instanceIdentitySQL).Scan(
		&identity.PostmasterStartTime, &identity.ServerAddr,
	)
	if err != nil {
		return state.PostgresInstanceIdentity{}, err
	}
	return identity, nil
}
