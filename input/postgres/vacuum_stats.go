package postgres

import (
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
	FROM pg_class c
	JOIN pg_namespace n ON (c.relnamespace = n.oid)
	JOIN pg_stat_user_tables s ON (s.relid = c.oid)
 WHERE c.relkind IN ('r', 'm') AND c.relpersistence IN ('p', 'u')
`

const globalVacuumSettingsSQL string = `
SELECT name, setting
	FROM pg_settings
 WHERE name LIKE 'autovacuum%'`

func GetVacuumStats(logger *util.Logger, db *sql.DB) (report state.PostgresVacuumStats, err error) {
	configRows, err := db.Query(QueryMarkerSQL + globalVacuumSettingsSQL)
	if err != nil {
		return
	}

	defer configRows.Close()

	for configRows.Next() {
		var name string
		var value string

		configRows.Scan(&name, &value)

		switch name {
		case "autovacuum":
			report.AutovacuumEnabled = value == "on"
		case "autovacuum_max_workers":
			val, _ := strconv.Atoi(value)
			report.AutovacuumMaxWorkers = int32(val)
		case "autovacuum_naptime":
			val, _ := strconv.Atoi(value)
			report.AutovacuumNaptimeSeconds = int32(val)
		case "autovacuum_vacuum_threshold":
			val, _ := strconv.Atoi(value)
			report.AutovacuumVacuumThreshold = int32(val)
		case "autovacuum_analyze_threshold":
			val, _ := strconv.Atoi(value)
			report.AutovacuumAnalyzeThreshold = int32(val)
		case "autovacuum_vacuum_scale_factor":
			val, _ := strconv.ParseFloat(value, 64)
			report.AutovacuumVacuumScaleFactor = val
		case "autovacuum_analyze_scale_factor":
			val, _ := strconv.ParseFloat(value, 64)
			report.AutovacuumAnalyzeScaleFactor = val
		case "autovacuum_freeze_max_age":
			val, _ := strconv.Atoi(value)
			report.AutovacuumFreezeMaxAge = int32(val)
		case "autovacuum_multixact_freeze_max_age":
			val, _ := strconv.Atoi(value)
			report.AutovacuumMultixactFreezeMaxAge = int32(val)
		case "autovacuum_vacuum_cost_delay":
			val, _ := strconv.Atoi(value)
			report.AutovacuumVacuumCostDelay = int32(val)
		case "autovacuum_vacuum_cost_limit":
			val, _ := strconv.Atoi(value)
			report.AutovacuumVacuumCostLimit = int32(val)
		}
	}

	rows, err := db.Query(QueryMarkerSQL + tableVacuumSQL)
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var entry state.PostgresVacuumStatsEntry
		var relopts string

		rows.Scan(&entry.SchemaName, &entry.RelationName, &entry.LiveRowCount,
			&entry.DeadRowCount, &entry.Relfrozenxid, &entry.Relminmxid,
			&entry.LastManualVacuumRun, &entry.LastAutoVacuumRun,
			&entry.LastManualAnalyzeRun, &entry.LastAutoAnalyzeRun,
			&relopts)

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
					val, _ := strconv.Atoi(parts[1])
					entry.AutovacuumVacuumThreshold = int32(val)
				case "autovacuum_analyze_threshold":
					val, _ := strconv.Atoi(parts[1])
					entry.AutovacuumAnalyzeThreshold = int32(val)
				case "autovacuum_vacuum_scale_factor":
					val, _ := strconv.ParseFloat(parts[1], 64)
					entry.AutovacuumVacuumScaleFactor = val
				case "autovacuum_analyze_scale_factor":
					val, _ := strconv.ParseFloat(parts[1], 64)
					entry.AutovacuumAnalyzeScaleFactor = val
				case "autovacuum_freeze_max_age":
					val, _ := strconv.Atoi(parts[1])
					entry.AutovacuumFreezeMaxAge = int32(val)
				case "autovacuum_multixact_freeze_max_age":
					val, _ := strconv.Atoi(parts[1])
					entry.AutovacuumMultixactFreezeMaxAge = int32(val)
				case "autovacuum_vacuum_cost_delay":
					val, _ := strconv.Atoi(parts[1])
					entry.AutovacuumVacuumCostDelay = int32(val)
				case "autovacuum_vacuum_cost_limit":
					val, _ := strconv.Atoi(parts[1])
					entry.AutovacuumVacuumCostLimit = int32(val)
				case "fillfactor":
					val, _ := strconv.Atoi(parts[1])
					entry.Fillfactor = int32(val)
				}
			}
		}

		report.Relations = append(report.Relations, entry)
	}

	report.DatabaseName, err = CurrentDatabaseName(db)
	if err != nil {
		return
	}

	return
}
