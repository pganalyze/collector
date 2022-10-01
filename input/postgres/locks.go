package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const blockingPidsSQL string = `
SELECT
	pid as blocked_pid,
	pg_blocking_pids(pid) as blocking_pids
FROM unnest(array[%s]::int[]) as pid`

func GetLocks(logger *util.Logger, db *sql.DB, backends []state.PostgresBackend) ([]state.PostgresLock, error) {
	var blockedPids []string
	for _, backend := range backends {
		// potential TODO: only use the ones that are waiting for more than the certain time
		if backend.Waiting.Bool {
			blockedPids = append(blockedPids, strconv.Itoa(int(backend.Pid)))
		}
	}

	blockingInfo, err := getBlockingInfo(logger, db, blockedPids)
	if err != nil {
		return nil, err
	}

	var locks []state.PostgresLock

	for blocked, blockings := range blockingInfo {
		locks = append(locks, state.PostgresLock{BlockedPid: blocked, BlockingPids: blockings})
	}

	return locks, nil
}

func getBlockingInfo(logger *util.Logger, db *sql.DB, blockedPids []string) (map[int32][]int64, error) {
	// potential TODO: use cache and if blockedPids are the same as previous, return the cache
	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(blockingPidsSQL, strings.Join(blockedPids, ",")))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	blocking := make(map[int32][]int64)

	for rows.Next() {
		var blockedPid int32
		var blockingPids []int64

		err := rows.Scan(&blockedPid, pq.Array(&blockingPids))
		if err != nil {
			return nil, err
		}

		blocking[blockedPid] = blockingPids
	}

	return blocking, nil
}
