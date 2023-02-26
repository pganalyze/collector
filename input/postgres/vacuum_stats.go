package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const tableVacuumSQL string = `
SELECT n.nspname,
			 c.relname,
			 c.reltuples,
			 s.n_dead_tup,
			 c.relfrozenxid,
			 c.relminmxid,
			 s.last_vacuum,
			 s.last_autovacuum,
			 s.last_analyze,
			 s.last_autoanalyze,
			 c.reloptions
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON (c.relnamespace = n.oid)
	JOIN pg_catalog.pg_stat_user_tables s ON (s.relid = c.oid)
 WHERE c.relkind IN ('r', 'm')
			 AND c.relpersistence IN ('p', 'u')
			 AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
`

const globalVacuumSettingsSQL string = `
SELECT name, setting
	FROM pg_catalog.pg_settings
 WHERE name LIKE 'autovacuum%'`

func GetVacuumStats(ctx context.Context, logger *util.Logger, db *sql.DB, ignoreRegexp string) (report state.PostgresVacuumStats, err error) {
	configRows, err := db.QueryContext(ctx, QueryMarkerSQL+globalVacuumSettingsSQL, ignoreRegexp)
	if err != nil {
		return
	}

	defer configRows.Close()

	for configRows.Next() {
		var name string
		var value string

		err = configRows.Scan(&name, &value)
		if err != nil {
			return
		}

		switch name {
		case "autovacuum":
			report.AutovacuumEnabled = value == "on"
		case "autovacuum_max_workers":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumMaxWorkers = int32(val)
		case "autovacuum_naptime":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumNaptimeSeconds = int32(val)
		case "autovacuum_vacuum_threshold":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumVacuumThreshold = int32(val)
		case "autovacuum_analyze_threshold":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumAnalyzeThreshold = int32(val)
		case "autovacuum_vacuum_scale_factor":
			val, _ := strconv.ParseFloat(value, 64)
			report.AutovacuumVacuumScaleFactor = val
		case "autovacuum_analyze_scale_factor":
			val, _ := strconv.ParseFloat(value, 64)
			report.AutovacuumAnalyzeScaleFactor = val
		case "autovacuum_freeze_max_age":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumFreezeMaxAge = int32(val)
		case "autovacuum_multixact_freeze_max_age":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumMultixactFreezeMaxAge = int32(val)
		case "autovacuum_vacuum_cost_delay":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumVacuumCostDelay = int32(val)
		case "autovacuum_vacuum_cost_limit":
			val, _ := strconv.ParseInt(value, 10, 32)
			report.AutovacuumVacuumCostLimit = int32(val)
		}
	}

	if err = configRows.Err(); err != nil {
		return
	}

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+tableVacuumSQL)
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var entry state.PostgresVacuumStatsEntry
		var relopts string

		err = rows.Scan(&entry.SchemaName, &entry.RelationName, &entry.LiveRowCount,
			&entry.DeadRowCount, &entry.Relfrozenxid, &entry.Relminmxid,
			&entry.LastManualVacuumRun, &entry.LastAutoVacuumRun,
			&entry.LastManualAnalyzeRun, &entry.LastAutoAnalyzeRun,
			&relopts)
		if err != nil {
			return
		}

		entry.AutovacuumEnabled = report.AutovacuumEnabled
		entry.AutovacuumVacuumThreshold = report.AutovacuumVacuumThreshold
		entry.AutovacuumVacuumScaleFactor = report.AutovacuumVacuumScaleFactor
		entry.AutovacuumFreezeMaxAge = report.AutovacuumFreezeMaxAge
		entry.AutovacuumMultixactFreezeMaxAge = report.AutovacuumMultixactFreezeMaxAge
		entry.AutovacuumVacuumCostDelay = report.AutovacuumVacuumCostDelay
		entry.AutovacuumVacuumCostLimit = report.AutovacuumVacuumCostLimit
		entry.Fillfactor = 100

		if len(relopts) >= 2 {
			for _, relopt := range strings.Split(relopts[1:len(relopts)-1], ",") {
				parts := strings.SplitN(relopt, "=", 2)
				switch parts[0] {
				case "autovacuum_enabled":
					entry.AutovacuumEnabled = parts[1] == "on"
				case "autovacuum_vacuum_threshold":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.AutovacuumVacuumThreshold = int32(val)
				case "autovacuum_analyze_threshold":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.AutovacuumAnalyzeThreshold = int32(val)
				case "autovacuum_vacuum_scale_factor":
					val, _ := strconv.ParseFloat(parts[1], 64)
					entry.AutovacuumVacuumScaleFactor = val
				case "autovacuum_analyze_scale_factor":
					val, _ := strconv.ParseFloat(parts[1], 64)
					entry.AutovacuumAnalyzeScaleFactor = val
				case "autovacuum_freeze_max_age":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.AutovacuumFreezeMaxAge = int32(val)
				case "autovacuum_multixact_freeze_max_age":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.AutovacuumMultixactFreezeMaxAge = int32(val)
				case "autovacuum_vacuum_cost_delay":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.AutovacuumVacuumCostDelay = int32(val)
				case "autovacuum_vacuum_cost_limit":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.AutovacuumVacuumCostLimit = int32(val)
				case "fillfactor":
					val, _ := strconv.ParseInt(parts[1], 10, 32)
					entry.Fillfactor = int32(val)
				}
			}
		}

		report.Relations = append(report.Relations, entry)
	}

	if err = rows.Err(); err != nil {
		return
	}

	report.DatabaseName, err = CurrentDatabaseName(ctx, db)
	if err != nil {
		return
	}

	return
}
