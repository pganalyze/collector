package logs

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func AnalyzeLogLines(logLinesIn []state.LogLine) (logLinesOut []state.LogLine, samples []state.PostgresQuerySample) {
	// Split log lines by backend to ensure we have the right context
	backendLogLines := make(map[int32][]state.LogLine)

	for _, logLine := range logLinesIn {
		backendLogLines[logLine.BackendPid] = append(backendLogLines[logLine.BackendPid], logLine)
	}

	for _, logLines := range backendLogLines {
		backendLogLinesOut, backendSamples := AnalyzeBackendLogLines(logLines)
		for _, logLine := range backendLogLinesOut {
			logLinesOut = append(logLinesOut, logLine)
		}
		for _, sample := range backendSamples {
			samples = append(samples, sample)
		}
	}

	return
}

func AnalyzeBackendLogLines(logLines []state.LogLine) (logLinesOut []state.LogLine, samples []state.PostgresQuerySample) {
	var parts []string

	additionalLines := 0

	for idx, logLine := range logLines {
		if additionalLines > 0 {
			logLinesOut = append(logLinesOut, logLine)
			additionalLines--
			continue
		}

		// Look up to 3 lines in the future to find context for this line
		var detailLine state.LogLine

		lowerBound := int(math.Min(float64(len(logLines)), float64(idx+1)))
		upperBound := int(math.Min(float64(len(logLines)), float64(idx+5)))
		for idx, futureLine := range logLines[lowerBound:upperBound] {
			if futureLine.LogLevel == pganalyze_collector.LogLineInformation_STATEMENT || futureLine.LogLevel == pganalyze_collector.LogLineInformation_DETAIL ||
				futureLine.LogLevel == pganalyze_collector.LogLineInformation_HINT || futureLine.LogLevel == pganalyze_collector.LogLineInformation_CONTEXT ||
				futureLine.LogLevel == pganalyze_collector.LogLineInformation_QUERY {
				if futureLine.LogLevel == pganalyze_collector.LogLineInformation_STATEMENT && !strings.HasSuffix(futureLine.Content, "[Your log message was truncated]") {
					logLine.Query = futureLine.Content
				}
				if futureLine.LogLevel == pganalyze_collector.LogLineInformation_DETAIL {
					detailLine = futureLine
				}
				logLines[lowerBound+idx].ParentUUID = logLine.UUID
				additionalLines++
			} else {
				break
			}
		}

		// Connects/Disconnects
		if strings.HasPrefix(logLine.Content, "connection received: ") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_RECEIVED
		}
		if strings.HasPrefix(logLine.Content, "connection authorized: ") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_AUTHORIZED
		}
		if strings.HasPrefix(logLine.Content, "pg_hba.conf rejects connection ") || strings.HasPrefix(logLine.Content, "password authentication failed for user") || strings.HasPrefix(logLine.Content, "no pg_hba.conf entry for") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
		}
		if regexp.MustCompile(`^database ".+?" is not currently accepting connections`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
		}
		if regexp.MustCompile(`^role ".+?" is not permitted to log in`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
		}
		if strings.HasPrefix(logLine.Content, "disconnection: ") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_DISCONNECTED
			parts = regexp.MustCompile(`^disconnection: session time: (\d+):(\d+):([\d\.]+)`).FindStringSubmatch(logLine.Content)
			if len(parts) == 4 {
				timeSecs, _ := strconv.ParseFloat(parts[3], 64)
				timeMinutes, _ := strconv.ParseFloat(parts[2], 64)
				timeHours, _ := strconv.ParseFloat(parts[1], 64)
				logLine.Details = map[string]interface{}{"session_time_secs": timeSecs + timeMinutes*60 + timeHours*3600}
			}
		}
		if strings.HasPrefix(logLine.Content, "incomplete startup packet") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_CLIENT_FAILED_TO_CONNECT
		}
		if strings.HasPrefix(logLine.Content, "could not receive data from client") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		}
		if strings.HasPrefix(logLine.Content, "could not send data to client") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		}
		if strings.HasPrefix(logLine.Content, "connection to client lost") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		}
		if strings.HasPrefix(logLine.Content, "terminating connection because protocol synchronization was lost") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		}
		if strings.HasPrefix(logLine.Content, "unexpected EOF on client connection with an open transaction") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST_OPEN_TX
		} else if strings.HasPrefix(logLine.Content, "unexpected EOF on client connection") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		}
		if strings.HasPrefix(logLine.Content, "remaining connection slots are reserved") {
			logLine.Classification = pganalyze_collector.LogLineInformation_OUT_OF_CONNECTIONS
		}
		if strings.HasPrefix(logLine.Content, "too many connections for role") {
			logLine.Classification = pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_ROLE
		}
		if strings.HasPrefix(logLine.Content, "terminating connection due to administrator command") {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_TERMINATED
		}

		// Checkpointer
		parts = regexp.MustCompile(`^(checkpoint|restartpoint) starting: (.+)`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			if parts[1] == "checkpoint" {
				logLine.Classification = pganalyze_collector.LogLineInformation_CHECKPOINT_STARTING
			} else if parts[1] == "restartpoint" {
				logLine.Classification = pganalyze_collector.LogLineInformation_RESTARTPOINT_STARTING
			}

			logLine.Details = map[string]interface{}{"reason": parts[2]}
		}
		parts = regexp.MustCompile(`^(checkpoint|restartpoint) complete: wrote (\d+) buffers \(([\d\.]+)%\); ` +
			`(\d+) transaction log file\(s\) added, (\d+) removed, (\d+) recycled; ` +
			`write=([\d\.]+) s, sync=([\d\.]+) s, total=([\d\.]+) s; ` +
			`sync files=(\d+), longest=([\d\.]+) s, average=([\d\.]+) s` +
			`(; distance=(\d+) kB, estimate=(\d+) kB)?`).FindStringSubmatch(logLine.Content)
		if len(parts) == 16 {
			if parts[1] == "checkpoint" {
				logLine.Classification = pganalyze_collector.LogLineInformation_CHECKPOINT_COMPLETE
			} else if parts[1] == "restartpoint" {
				logLine.Classification = pganalyze_collector.LogLineInformation_RESTARTPOINT_COMPLETE
			}

			bufsWritten, _ := strconv.ParseInt(parts[2], 10, 64)
			bufsWrittenPct, _ := strconv.ParseFloat(parts[3], 64)
			segsAdded, _ := strconv.ParseInt(parts[4], 10, 64)
			segsRemoved, _ := strconv.ParseInt(parts[5], 10, 64)
			segsRecycled, _ := strconv.ParseInt(parts[6], 10, 64)
			writeSecs, _ := strconv.ParseFloat(parts[7], 64)
			syncSecs, _ := strconv.ParseFloat(parts[8], 64)
			totalSecs, _ := strconv.ParseFloat(parts[9], 64)
			syncRels, _ := strconv.ParseInt(parts[10], 10, 64)
			longestSecs, _ := strconv.ParseFloat(parts[11], 64)
			averageSecs, _ := strconv.ParseFloat(parts[12], 64)
			logLine.Details = map[string]interface{}{
				"bufs_written": bufsWritten, "segs_added": segsAdded,
				"segs_removed": segsRemoved, "segs_recycled": segsRecycled,
				"sync_rels":        syncRels,
				"bufs_written_pct": bufsWrittenPct, "write_secs": writeSecs,
				"sync_secs": syncSecs, "total_secs": totalSecs,
				"longest_secs": longestSecs, "average_secs": averageSecs,
			}

			// Postgres 9.5 and newer
			if parts[14] != "" {
				distanceKb, _ := strconv.ParseInt(parts[14], 10, 64)
				logLine.Details["distance_kb"] = distanceKb
			}
			if parts[15] != "" {
				estimateKb, _ := strconv.ParseInt(parts[15], 10, 64)
				logLine.Details["estimate_kb"] = estimateKb
			}
		}
		parts = regexp.MustCompile(`^checkpoints are occurring too frequently \((\d+) seconds? apart\)`).FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECKPOINT_TOO_FREQUENT
			elapsedSecs, _ := strconv.ParseInt(parts[1], 10, 64)
			logLine.Details = map[string]interface{}{
				"elapsed_secs": elapsedSecs,
			}
		}
		if strings.HasPrefix(logLine.Content, "recovery restart point at") {
			logLine.Classification = pganalyze_collector.LogLineInformation_RESTARTPOINT_AT
		}

		// WAL/Archiving
		if strings.HasPrefix(logLine.Content, "invalid record length") {
			logLine.Classification = pganalyze_collector.LogLineInformation_WAL_INVALID_RECORD_LENGTH
		}
		if strings.HasPrefix(logLine.Content, "redo ") {
			logLine.Classification = pganalyze_collector.LogLineInformation_WAL_REDO
		}

		// Lock waits
		parts = regexp.MustCompile(`^process \d+ acquired (\w+Lock) on (\w+) [\(\)\d,]+( of \w+ \d+)* after ([\d\.]+) ms`).FindStringSubmatch(logLine.Content)
		if len(parts) == 5 {
			logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_ACQUIRED
			afterMs, _ := strconv.ParseFloat(parts[4], 64)
			logLine.Details = map[string]interface{}{
				"lock_mode": parts[1],
				"lock_type": parts[2],
				"after_ms":  afterMs,
			}
		}
		parts = regexp.MustCompile(`^process \d+ (still waiting|avoided deadlock|detected deadlock while waiting) for (\w+) on (\w+) (?:.+?) after ([\d\.]+) ms`).FindStringSubmatch(logLine.Content)
		if len(parts) == 5 {
			if parts[1] == "still waiting" {
				logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_WAITING
			} else if parts[1] == "avoided deadlock" {
				logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_AVOIDED
			} else if parts[1] == "detected deadlock while waiting" {
				logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_DETECTED
			}
			lockType := parts[3]
			// Match lock types to names in pg_locks.locktype
			if lockType == "extension" {
				lockType = "extend"
			} else if lockType == "transaction" {
				lockType = "transactionid"
			} else if lockType == "virtual" {
				lockType = "virtualxid"
			}
			afterMs, _ := strconv.ParseFloat(parts[4], 64)
			logLine.Details = map[string]interface{}{"lock_mode": parts[2], "lock_type": lockType, "after_ms": afterMs}
			if additionalLines > 0 && logLines[lowerBound].LogLevel == pganalyze_collector.LogLineInformation_DETAIL {
				parts = regexp.MustCompile(`^Process(?:es)? holding the lock: ([\d, ]+). Wait queue: ([\d, ]+)`).FindStringSubmatch(logLines[lowerBound].Content)
				if len(parts) == 3 {
					lockHolders := []int64{}
					for _, s := range strings.Split(parts[1], ", ") {
						i, _ := strconv.ParseInt(s, 10, 64)
						lockHolders = append(lockHolders, i)
					}
					lockWaiters := []int64{}
					for _, s := range strings.Split(parts[2], ", ") {
						i, _ := strconv.ParseInt(s, 10, 64)
						lockWaiters = append(lockWaiters, i)
					}
					logLine.Details["lock_holders"] = lockHolders
					logLine.Details["lock_waiters"] = lockWaiters
				}
			}
		}
		if strings.HasPrefix(logLine.Content, "deadlock detected") {
			logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_DETECTED
		}
		if strings.HasPrefix(logLine.Content, "canceling statement due to lock timeout") {
			logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_TIMEOUT
		}

		// Statement duration (log_min_duration_statement output)
		if strings.HasPrefix(logLine.Content, "duration: ") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_DURATION
			if strings.HasSuffix(strings.TrimSpace(logLine.Content), "[Your log message was truncated]") {
				logLine.Details = map[string]interface{}{"truncated": true}
			} else {
				parts = regexp.MustCompile(`(?ms)^duration: ([\d\.]+) ms([^:]+):(.+)`).FindStringSubmatch(logLine.Content)

				if len(parts) == 4 {
					logLine.Query = strings.TrimSpace(parts[3])

					if !strings.Contains(parts[2], "bind") && !strings.Contains(parts[2], "parse") {
						runtime, _ := strconv.ParseFloat(parts[1], 64)
						logLine.Details = map[string]interface{}{"duration_ms": runtime}
						sample := state.PostgresQuerySample{
							OccurredAt:  logLine.OccurredAt,
							Username:    logLine.Username,
							Database:    logLine.Database,
							Query:       logLine.Query,
							LogLineUUID: logLine.UUID,
							RuntimeMs:   runtime,
						}
						if strings.HasPrefix(detailLine.Content, "parameters: ") {
							parameterParts := regexp.MustCompile(`\$\d+ = '([^']*)',?\s*`).FindAllStringSubmatch(detailLine.Content, -1)
							for _, part := range parameterParts {
								if len(part) == 2 {
									sample.Parameters = append(sample.Parameters, string(part[1]))
								}
							}
						}
						samples = append(samples, sample)
					}
				}
			}
		}

		// Statement cancellation (except lock timeout)
		if strings.HasPrefix(logLine.Content, "canceling statement due to statement timeout") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_TIMEOUT
		}
		if strings.HasPrefix(logLine.Content, "canceling statement due to user request") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_USER
		}

		// Autovacuum
		if strings.HasPrefix(logLine.Content, "canceling autovacuum task") {
			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_CANCEL
		}
		parts = regexp.MustCompile(`^database (with OID (\d+)|"(.+?)") must be vacuumed within (\d+) transactions`).FindStringSubmatch(logLine.Content)
		if len(parts) == 5 {
			logLine.Classification = pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_WARNING
			remainingXids, _ := strconv.ParseInt(parts[4], 10, 64)
			logLine.Details = map[string]interface{}{"remaining_xids": remainingXids}
			if parts[2] != "" {
				databaseOid, _ := strconv.ParseInt(parts[2], 10, 64)
				logLine.Details["database_oid"] = databaseOid
			}
			if parts[3] != "" {
				logLine.Details["database_name"] = parts[3]
			}
		}
		parts = regexp.MustCompile(`^database is not accepting commands to avoid wraparound data loss in database (with OID (\d+)|"(.+?)")`).FindStringSubmatch(logLine.Content)
		if len(parts) == 4 {
			logLine.Classification = pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_ERROR
			if parts[2] != "" {
				databaseOid, _ := strconv.ParseInt(parts[2], 10, 64)
				logLine.Details = map[string]interface{}{"database_oid": databaseOid}
			}
			if parts[3] != "" {
				logLine.Details = map[string]interface{}{"database_name": parts[3]}
			}
		}
		if strings.HasPrefix(logLine.Content, "autovacuum launcher started") {
			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_STARTED
		}
		if strings.HasPrefix(logLine.Content, "autovacuum launcher shutting down") {
			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_SHUTTING_DOWN
		}
		parts = regexp.MustCompile(`^automatic vacuum of table "(.+?)": index scans: (\d+)\s*` +
			`pages: (\d+) removed, (\d+) remain(?:, (\d+) skipped due to pins)?(?:, (\d+) skipped frozen)?\s*` +
			`tuples: (\d+) removed, (\d+) remain, (\d+) are dead but not yet removable\s*` +
			`buffer usage: (\d+) hits, (\d+) misses, (\d+) dirtied\s*` +
			`avg read rate: ([\d.]+) MB/s, avg write rate: ([\d.]+) MB/s\s*` +
			`system usage: CPU ([\d.]+)s/([\d.]+)u sec elapsed ([\d.]+) sec`).FindStringSubmatch(logLine.Content)
		if len(parts) == 18 {
			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED
			// FIXME: Associate relation (parts[1])

			numIndexScans, _ := strconv.ParseInt(parts[2], 10, 64)
			pagesRemoved, _ := strconv.ParseInt(parts[3], 10, 64)
			relPages, _ := strconv.ParseInt(parts[4], 10, 64)
			tuplesDeleted, _ := strconv.ParseInt(parts[7], 10, 64)
			newRelTuples, _ := strconv.ParseInt(parts[8], 10, 64)
			newDeadTuples, _ := strconv.ParseInt(parts[9], 10, 64)
			vacuumPageHit, _ := strconv.ParseInt(parts[10], 10, 64)
			vacuumPageMiss, _ := strconv.ParseInt(parts[11], 10, 64)
			vacuumPageDirty, _ := strconv.ParseInt(parts[12], 10, 64)
			readRateMb, _ := strconv.ParseFloat(parts[13], 64)
			writeRateMb, _ := strconv.ParseFloat(parts[14], 64)
			rusageKernelMode, _ := strconv.ParseFloat(parts[15], 64)
			rusageUserMode, _ := strconv.ParseFloat(parts[16], 64)
			rusageElapsed, _ := strconv.ParseFloat(parts[17], 64)
			logLine.Details = map[string]interface{}{
				"num_index_scans": numIndexScans, "pages_removed": pagesRemoved,
				"rel_pages": relPages, "tuples_deleted": tuplesDeleted,
				"new_rel_tuples": newRelTuples, "new_dead_tuples": newDeadTuples,
				"vacuum_page_hit": vacuumPageHit, "vacuum_page_miss": vacuumPageMiss,
				"vacuum_page_dirty": vacuumPageDirty, "read_rate_mb": readRateMb,
				"write_rate_mb": writeRateMb, "rusage_kernel": rusageKernelMode,
				"rusage_user": rusageUserMode, "elapsed_secs": rusageElapsed,
			}
			if parts[5] != "" {
				pinskippedPages, _ := strconv.ParseInt(parts[5], 10, 64)
				logLine.Details["pinskipped_pages"] = pinskippedPages
			}
			if parts[6] != "" {
				frozenskippedPages, _ := strconv.ParseInt(parts[6], 10, 64)
				logLine.Details["frozenskipped_pages"] = frozenskippedPages
			}
		}
		parts = regexp.MustCompile(`^automatic analyze of table "(.+?)" system usage: CPU ([\d.]+)s/([\d.]+)u sec elapsed ([\d.]+) sec`).FindStringSubmatch(logLine.Content)
		if len(parts) == 5 {
			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOANALYZE_COMPLETED
			// FIXME: Associate relation (parts[1])
			rusageKernelMode, _ := strconv.ParseFloat(parts[2], 64)
			rusageUserMode, _ := strconv.ParseFloat(parts[3], 64)
			rusageElapsed, _ := strconv.ParseFloat(parts[4], 64)
			logLine.Details = map[string]interface{}{
				"rusage_kernel": rusageKernelMode, "rusage_user": rusageUserMode,
				"elapsed_secs": rusageElapsed,
			}
		}

		// Server events
		if regexp.MustCompile(`^server process \(PID \d+\) was terminated by signal (6|11)`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_CRASHED
		}
		if strings.HasPrefix(logLine.Content, "terminating any other active server processes") || strings.HasPrefix(logLine.Content, "terminating connection because of crash of another server process") || strings.HasPrefix(logLine.Content, "all server processes terminated; reinitializing") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_CRASHED
		}
		if strings.HasPrefix(logLine.Content, "database system was shut down") || strings.HasPrefix(logLine.Content, "database system is ready") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		}
		if strings.HasPrefix(logLine.Content, "MultiXact member wraparound protections are now enabled") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		}
		if strings.HasPrefix(logLine.Content, "entering standby mode") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		}
		if strings.HasPrefix(logLine.Content, "database system was interrupted") || strings.HasPrefix(logLine.Content, "database system was not properly shut down") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING
		}
		if regexp.MustCompile(`^received \w+ shutdown request`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN
		}
		if strings.HasPrefix(logLine.Content, "aborting any active transactions") || strings.HasPrefix(logLine.Content, "shutting down") || strings.HasPrefix(logLine.Content, "database system is shut down") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN
		}
		if strings.HasPrefix(logLine.Content, "out of memory") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY
		}
		if regexp.MustCompile(`^server process \(PID \d+\) was terminated by signal 9`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY
		}
		if strings.HasPrefix(logLine.Content, "page verification failed") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM
		}
		parts = regexp.MustCompile(`^invalid page in block (\d+) of relation (.+)`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM
			blockNo, _ := strconv.ParseInt(parts[1], 10, 64)
			logLine.Details = map[string]interface{}{"block": blockNo, "file": parts[2]}
		}
		parts = regexp.MustCompile(`^temporary file: path "(.+?)", size (\d+)`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_TEMP_FILE_CREATED
			size, _ := strconv.ParseInt(parts[2], 10, 64)
			logLine.Details = map[string]interface{}{"size": size, "file": parts[1]}
		}
		if strings.HasPrefix(logLine.Content, "could not open usermap file") ||
			strings.HasPrefix(logLine.Content, "invalid byte sequence for encoding") ||
			strings.HasPrefix(logLine.Content, "could not link file") ||
			strings.HasPrefix(logLine.Content, "unexpected pageaddr") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_MISC
		}
		if strings.HasPrefix(logLine.Content, "received SIGHUP, reloading configuration files") {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_RELOAD
		}
		if regexp.MustCompile(`^parameter ".+?" (changed|cannot be changed)`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_RELOAD
		}
		if regexp.MustCompile(`^configuration file ".+?" contains errors`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_RELOAD
		}

		// Standby
		if strings.HasPrefix(logLine.Content, "restored log file") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_RESTORED_WAL_FROM_ARCHIVE
		}
		if strings.HasPrefix(logLine.Content, "started streaming WAL") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STARTED_STREAMING
		}
		if strings.HasPrefix(logLine.Content, "could not receive data from WAL stream") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STREAMING_INTERRUPTED
		}
		if strings.HasPrefix(logLine.Content, "terminating walreceiver process") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STOPPED_STREAMING
		}
		if strings.HasPrefix(logLine.Content, "consistent recovery state reached at") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_CONSISTENT_RECOVERY_STATE
		}
		if strings.HasPrefix(logLine.Content, "canceling statement due to conflict with recovery") {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STATEMENT_CANCELED
		}
		if regexp.MustCompile(`^according to history file, WAL location .+? belongs to timeline \d+, but previous recovered WAL file came from timeline \d+`).MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_INVALID_TIMELINE
		}

		// Constraint violations
		parts = regexp.MustCompile(`^duplicate key value violates unique constraint "(.+?)"`).FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_UNIQUE_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
		}
		parts = regexp.MustCompile(`^insert or update on table "(.+?)" violates foreign key constraint "(.+?)"`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name and relation name
		}
		parts = regexp.MustCompile(`^update or delete on table "(.+?)" violates foreign key constraint "(.+?)" on table "(.+?)"`).FindStringSubmatch(logLine.Content)
		if len(parts) == 4 {
			logLine.Classification = pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name and both relation names
		}
		parts = regexp.MustCompile(`^null value in column "(.+?)" violates not-null constraint`).FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_NOT_NULL_CONSTRAINT_VIOLATION
		}
		parts = regexp.MustCompile(`^new row for relation "(.+?)" violates check constraint "(.+?)"`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name and relation name
		}
		parts = regexp.MustCompile(`^check constraint "(.+?)" is violated by some row`).FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
		}
		parts = regexp.MustCompile(`^column "(.+?)" of table "(.+?)" contains values that violate the new constraint`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store relation name
		}
		parts = regexp.MustCompile(`^value for domain (.+?) violates check constraint "(.+?)"`).FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
		}
		parts = regexp.MustCompile(`^conflicting key value violates exclusion constraint "(.+?)"`).FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_EXCLUSION_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
		}

		//if logLine.Classification == pganalyze_collector.LogLineInformation_UNKNOWN_LOG_CLASSIFICATION {
		//	fmt.Printf("%s\n", logLine.Content)
		//}

		logLinesOut = append(logLinesOut, logLine)
	}

	// Ensure no other part of the system accidentally sends log line contents, as
	// they should be considered opaque from here on
	for idx := range logLinesOut {
		logLinesOut[idx].Content = ""
	}

	return
}
