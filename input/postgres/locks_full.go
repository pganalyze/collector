package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// expand the case for classid/objid/objsubid
// consider changing relation oid to table name

// [example data]
//
//	blocked_pid |    blocked_mode     | blocked_lock_type | blocking_pid |    blocking_mode
//
// -------------+---------------------+-------------------+--------------+---------------------
//
//	47473 | AccessShareLock     | relation 16404    |        65277 | AccessExclusiveLock
//	65277 | AccessExclusiveLock | relation 16404    |        47473 | AccessShareLock
//	47473 | AccessShareLock     | relation 16404    |        47516 | RowExclusiveLock
//	65277 | AccessExclusiveLock | relation 16404    |        47516 | RowExclusiveLock
//
// (4 rows)
const lockInfoSQL string = `
SELECT
	blocked_locks.pid as blocked_pid,
	blocked_locks.mode as blocked_mode,
	CASE
		WHEN blocked_locks.locktype IN ('relation', 'extend') THEN
			'relation ' || blocked_locks.relation
		WHEN blocked_locks.locktype = 'page' THEN
			'relation ' || blocked_locks.relation || ' page ' || blocked_locks.page
		WHEN blocked_locks.locktype = 'tuple' THEN
			'relation ' || blocked_locks.relation || ' page ' || blocked_locks.page || ' tuple ' || blocked_locks.tuple
		WHEN blocked_locks.locktype = 'transactionid' THEN
			'transactionid ' || blocked_locks.transactionid
		WHEN blocked_locks.locktype = 'virtualxid' THEN
			'virtualxid ' || blocked_locks.virtualxid
		ELSE
			blocked_locks.locktype
	END as blocked_lock_type,
	blocking_locks.pid as blocking_pid,
	blocking_locks.mode as blocking_mode
FROM pg_catalog.pg_locks blocked_locks
	LEFT JOIN pg_catalog.pg_locks blocking_locks 
        ON blocking_locks.locktype = blocked_locks.locktype
        AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
        AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
        AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
        AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
        AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
        AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
        AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
        AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
        AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
        AND blocking_locks.pid != blocked_locks.pid
WHERE blocked_locks.pid IN (%s) AND NOT blocked_locks.granted`

func GetLocksFull(logger *util.Logger, db *sql.DB, backends []state.PostgresBackend) ([]state.PostgresLockFull, error) {
	var blockedPids []string
	for _, backend := range backends {
		if backend.Waiting.Bool {
			blockedPids = append(blockedPids, strconv.Itoa(int(backend.Pid)))
		}
	}

	blockingInfo, err := getBlockingInfo(logger, db, blockedPids)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(lockInfoSQL, strings.Join(blockedPids, ",")))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// This could grab inaccurate rows
	var allLocks []state.PostgresLockFull
	for rows.Next() {
		var row state.PostgresLockFull

		err := rows.Scan(&row.BlockedPid, &row.BlockedMode, &row.BlockedLockType,
			&row.BlockingPid, &row.BlockingMode)
		if err != nil {
			return nil, err
		}

		allLocks = append(allLocks, row)
	}

	// Remove falsely created rows
	// (blocked_pid and blocking_pid are not matching to the blockingPids list)
	var locks []state.PostgresLockFull
	for _, lock := range allLocks {
		if containsBlocking(blockingInfo[lock.BlockedPid], lock.BlockingPid) {
			locks = append(locks, lock)
		}
	}

	return locks, nil
}

// containsBlocking checks if the certain blocking PID exists within the slice (list) of blocking PIDs.
// This can be replaced with slices.Contains once it becomes a standard library
func containsBlocking(blockings []int64, blocking int32) bool {
	for _, v := range blockings {
		if int(v) == int(blocking) {
			return true
		}
	}
	return false
}
