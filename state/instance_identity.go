package state

import (
	"fmt"
	"time"
)

// PostgresInstanceIdentity - Identifies the specific Postgres instance, and the
// specific postmaster on it, that a connection was made to.
//
// Cumulative statistics (pg_stat_statements, pg_stat_user_tables, etc.) live in
// the memory of one postmaster and are not synchronized between instances, so
// diffing them is only meaningful when both sides of the diff came from the
// same instance and the same postmaster.
//
// This matters for clusters where a single hostname resolves to different
// instances over time. On an Aurora reader endpoint, a failover promotes the
// reader to writer and demotes the old writer to reader, so the reader endpoint
// starts pointing at what used to be the writer. Diffing that instance's
// counters against the previous reader's makes the collector attribute entire
// counter lifetimes to a single interval.
type PostgresInstanceIdentity struct {
	// Start time of the postmaster we're connected to; changes when it restarts
	PostmasterStartTime time.Time
	// Address of the instance this connection landed on, from inet_server_addr().
	// Empty for Unix socket connections, where it is not available.
	ServerAddr string
}

// IsZero - Returns whether the identity is unset, i.e. we were unable to
// determine which instance a connection reached.
func (i PostgresInstanceIdentity) IsZero() bool {
	return i.PostmasterStartTime.IsZero() && i.ServerAddr == ""
}

// Matches - Returns whether two identities refer to the same postmaster on the
// same instance.
//
// A zero identity matches anything. We can fail to determine the identity (an
// old Postgres version, a permission problem, a transient error), and in that
// case we prefer to keep diffing as before rather than discard data on the
// basis of information we do not have.
func (i PostgresInstanceIdentity) Matches(other PostgresInstanceIdentity) bool {
	if i.IsZero() || other.IsZero() {
		return true
	}
	// Compare the timestamps with Equal rather than ==, since the driver hands us
	// a separately allocated *time.Location for each row it scans, which makes ==
	// report two readings of the same start time as different
	return i.PostmasterStartTime.Equal(other.PostmasterStartTime) &&
		i.ServerAddr == other.ServerAddr
}

func (i PostgresInstanceIdentity) String() string {
	if i.IsZero() {
		return "unknown instance"
	}
	addr := i.ServerAddr
	if addr == "" {
		addr = "local"
	}
	return fmt.Sprintf("%s started %s", addr, i.PostmasterStartTime.UTC().Format(time.RFC3339))
}
