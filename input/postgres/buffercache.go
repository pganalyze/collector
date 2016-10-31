package postgres

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const buffercacheSQL string = `WITH buffers AS (
	SELECT COUNT(*) AS block_count, reldatabase, relfilenode
	FROM %s
	GROUP BY 2, 3
)
SELECT block_count * current_setting('block_size')::int, d.datname, nspname, relname, relkind::text
FROM buffers b
JOIN pg_database d ON (d.oid = reldatabase)
LEFT JOIN pg_class c ON (b.relfilenode = pg_relation_filenode(c.oid) AND (b.reldatabase = 0 OR d.datname = current_database()))
LEFT JOIN pg_namespace n ON (n.oid = c.relnamespace)
UNION
SELECT SUM(block_count) * current_setting('block_size')::int, '', NULL, NULL, 'used' FROM buffers
`

const buffercacheHelperSQL string = `
SELECT 1 AS enabled
	FROM pg_proc
	JOIN pg_namespace ON (pronamespace = pg_namespace.oid)
 WHERE nspname = 'pganalyze' AND proname = 'get_buffercache'
`

const sharedBufferSettingSQL string = `SELECT current_setting('shared_buffers')`

func getSharedBufferBytes(db *sql.DB) int64 {
	var bytesStr string

	err := db.QueryRow(QueryMarkerSQL + sharedBufferSettingSQL).Scan(&bytesStr)
	if err != nil {
		return 0
	}

	re := regexp.MustCompile("(\\d+)\\s*(\\w+)")
	parts := re.FindStringSubmatch(bytesStr)

	if len(parts) != 3 {
		return 0
	}

	var multiplier int64
	switch strings.ToLower(parts[2]) {
	case "bytes":
		multiplier = 1
	case "kb":
		multiplier = 1024
	case "mb":
		multiplier = 1024 * 1024
	case "gb":
		multiplier = 1024 * 1024 * 1024
	case "tb":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	bytes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0
	}

	return bytes * multiplier
}

func buffercacheHelperExists(db *sql.DB) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + buffercacheHelperSQL).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

func GetBuffercache(logger *util.Logger, db *sql.DB) (report state.PostgresBuffercache, err error) {
	var sourceTable string

	if buffercacheHelperExists(db) {
		logger.PrintVerbose("Found pganalyze.get_buffercache() stats helper")
		sourceTable = "pganalyze.get_buffercache()"
	} else {
		if !connectedAsSuperUser(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser to run the buffercache report.")
		}
		sourceTable = "pg_buffercache"
	}

	rows, err := db.Query(QueryMarkerSQL + fmt.Sprintf(buffercacheSQL, sourceTable))
	if err != nil {
		if err.(*pq.Error).Code == "42P01" { // undefined_table
			logger.PrintInfo("pg_buffercache relation does not exist, trying to create extension...")

			_, err = db.Exec(QueryMarkerSQL + "CREATE EXTENSION IF NOT EXISTS pg_buffercache")
			if err != nil {
				return
			}

			rows, err = db.Query(QueryMarkerSQL + buffercacheSQL)
			if err != nil {
				return
			}
		} else {
			return
		}
	}

	if err != nil {
		err = fmt.Errorf("Buffercache/Query: %s", err)
		return
	}

	defer rows.Close()

	var usedBytes int64

	for rows.Next() {
		var row state.PostgresBuffercacheEntry

		err = rows.Scan(&row.Bytes, &row.DatabaseName, &row.SchemaName,
			&row.ObjectName, &row.ObjectKind)
		if err != nil {
			err = fmt.Errorf("Buffercache/Scan: %s", err)
			return
		}

		if row.DatabaseName == "" && row.ObjectKind != nil && *row.ObjectKind == "used" {
			usedBytes = row.Bytes
		} else {
			report.Entries = append(report.Entries, row)
		}
	}

	report.TotalBytes = getSharedBufferBytes(db)
	report.FreeBytes = report.TotalBytes - usedBytes

	return
}
