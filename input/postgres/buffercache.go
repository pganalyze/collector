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

const buffercacheSQL string = `
WITH buffers AS (
	SELECT pg_catalog.count(*) AS block_count, reldatabase, relfilenode
	  FROM %s
	 GROUP BY 2, 3
)
SELECT block_count * pg_catalog.current_setting('block_size')::int, d.datname, nspname, relname, relkind::text
  FROM buffers b
  JOIN pg_catalog.pg_database d ON (d.oid = reldatabase)
       LEFT JOIN pg_catalog.pg_class c ON (b.relfilenode = pg_catalog.pg_relation_filenode(c.oid) AND (b.reldatabase = 0 OR d.datname = pg_catalog.current_database()))
       LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
 WHERE ($1 = '' OR (coalesce(n.nspname, '') || '.' || coalesce(c.relname, '')) !~* $1)
UNION
SELECT pg_catalog.sum(block_count) * pg_catalog.current_setting('block_size')::int, '', NULL, NULL, 'used'
  FROM buffers
`

const sharedBufferSettingSQL string = `SELECT pg_catalog.current_setting('shared_buffers')`

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

func GetBuffercache(logger *util.Logger, db *sql.DB, systemType, ignoreRegexp string) (report state.PostgresBuffercache, err error) {
	var sourceTable string

	if statsHelperExists(db, "get_buffercache") {
		logger.PrintVerbose("Found pganalyze.get_buffercache() stats helper")
		sourceTable = "pganalyze.get_buffercache()"
	} else {
		if !connectedAsSuperUser(db, systemType) && !connectedAsMonitoringRole(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser to run the buffercache report.")
		}
		sourceTable = "public.pg_buffercache"
	}

	query := QueryMarkerSQL + fmt.Sprintf(buffercacheSQL, sourceTable)
	rows, err := db.Query(query, ignoreRegexp)
	if err != nil {
		if err.(*pq.Error).Code == "42P01" { // undefined_table
			logger.PrintInfo("pg_buffercache relation does not exist, trying to create extension...")

			_, err = db.Exec(QueryMarkerSQL + "CREATE EXTENSION IF NOT EXISTS pg_buffercache SCHEMA public")
			if err != nil {
				return
			}

			rows, err = db.Query(query, ignoreRegexp)
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

	for idx, row := range report.Entries {
		if row.SchemaName != nil && *row.SchemaName == "pg_toast" && row.ObjectName != nil {
			toastTable := *row.ObjectName
			if row.ObjectKind != nil && *row.ObjectKind == "i" {
				toastTable = strings.Replace(toastTable, "_index", "", 1)
			}
			schemaName, relationName, err := resolveToastTable(db, toastTable)
			if err != nil {
				logger.PrintVerbose("Failed to resolve TOAST table \"%s\": %s", toastTable, err)
			} else if schemaName != "" && relationName != "" {
				row.SchemaName = &schemaName
				row.ObjectName = &relationName
				row.Toast = true
				report.Entries[idx] = row
			}
		}
	}

	report.TotalBytes = getSharedBufferBytes(db)
	report.FreeBytes = report.TotalBytes - usedBytes

	return
}
