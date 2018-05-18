package logs

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

var ContentAutoExplainRegexp = regexp.MustCompile(`^duration: ([\d\.]+) ms\s+ plan:([\s\S]+)`)
var ContentDurationRegexp = regexp.MustCompile(`(?ms)^duration: ([\d\.]+) ms([^:]+):(.+)`)
var ContentDurationDetailsRegexp = regexp.MustCompile(`\$\d+ = '([^']*)',?\s*`)
var ContentAutoVacuumRegexp = regexp.MustCompile(`^automatic vacuum of table "(.+?)": index scans: (\d+)\s*` +
	`pages: (\d+) removed, (\d+) remain(?:, (\d+) skipped due to pins)?(?:, (\d+) skipped frozen)?\s*` +
	`tuples: (\d+) removed, (\d+) remain, (\d+) are dead but not yet removable(?:, oldest xmin: (\d+))?\s*` +
	`buffer usage: (\d+) hits, (\d+) misses, (\d+) dirtied\s*` +
	`avg read rate: ([\d.]+) MB/s, avg write rate: ([\d.]+) MB/s\s*` +
	`system usage: CPU(?:(?: ([\d.]+)s/([\d.]+)u sec elapsed ([\d.]+) sec)|(?:: user: ([\d.]+) s, system: ([\d.]+) s, elapsed: ([\d.]+) s))`)
var ContentAutoAnalyzeRegexp = regexp.MustCompile(`^automatic analyze of table "(.+?)" system usage: CPU(?:(?: ([\d.]+)s/([\d.]+)u sec elapsed ([\d.]+) sec)|(?:: user: ([\d.]+) s, system: ([\d.]+) s, elapsed: ([\d.]+) s))`)
var ContentCheckpointStartingRegexp = regexp.MustCompile(`^(checkpoint|restartpoint) starting: (.+)`)
var ContentCheckpointCompleteRegexp = regexp.MustCompile(`^(checkpoint|restartpoint) complete: wrote (\d+) buffers \(([\d\.]+)%\); ` +
	`(\d+) (?:transaction log|WAL) file\(s\) added, (\d+) removed, (\d+) recycled; ` +
	`write=([\d\.]+) s, sync=([\d\.]+) s, total=([\d\.]+) s; ` +
	`sync files=(\d+), longest=([\d\.]+) s, average=([\d\.]+) s` +
	`(; distance=(\d+) kB, estimate=(\d+) kB)?`)
var ContentDisconnectionRegexp = regexp.MustCompile(`^disconnection: session time: (\d+):(\d+):([\d\.]+)`)
var ContentRoleNotAllowedLoginRegexp = regexp.MustCompile(`^role ".+?" is not permitted to log in`)
var ContentDatabaseNotAcceptingConnectionsRegexp = regexp.MustCompile(`^database ".+?" is not currently accepting connections`)
var ContentCheckpointsTooFrequentRegexp = regexp.MustCompile(`^checkpoints are occurring too frequently \((\d+) seconds? apart\)`)
var ContentRedoLastTxRegexp = regexp.MustCompile(`^last completed transaction was at log time (.+)`)
var ContentArchiveCommandFailedRegexp = regexp.MustCompile(`^archive command (?:failed with exit code (\d+)|was terminated by signal (\d+))`)
var ContentArchiveCommandFailedDetailsRegexp = regexp.MustCompile(`^The failed archive command was: (.+)`)
var ContentLockAcquiredRegexp = regexp.MustCompile(`^process \d+ acquired (\w+Lock) on (\w+)(?: [\(\)\d,]+)?( of \w+ \d+)* after ([\d\.]+) ms`)
var ContentLockWaitRegexp = regexp.MustCompile(`^process \d+ (still waiting|avoided deadlock|detected deadlock while waiting) for (\w+) on (\w+) (?:.+?) after ([\d\.]+) ms`)
var ContentLockWaitDetailsRegexp = regexp.MustCompile(`^Process(?:es)? holding the lock: ([\d, ]+). Wait queue: ([\d, ]+)`)
var ContentDeadlockDetailsRegexp = regexp.MustCompile(`(?m)^Process (\d+)`)
var ContentWraparoundWarningRegexp = regexp.MustCompile(`^database (with OID (\d+)|"(.+?)") must be vacuumed within (\d+) transactions`)
var ContentWraparoundErrorRegexp = regexp.MustCompile(`^database is not accepting commands to avoid wraparound data loss in database (with OID (\d+)|"(.+?)")`)
var ContentServerCrashedRegexp = regexp.MustCompile(`^server process \(PID (\d+)\) was terminated by signal (6|11)`)
var ContentServerOutOfMemoryRegexp = regexp.MustCompile(`^server process \(PID (\d+)\) was terminated by signal (9)`)
var ContentTemporaryFileRegexp = regexp.MustCompile(`^temporary file: path "(.+?)", size (\d+)`)
var ContentReceivedShutdownRequestRegexp = regexp.MustCompile(`^received \w+ shutdown request`)
var ContentInvalidChecksumRegexp = regexp.MustCompile(`^invalid page in block (\d+) of relation (.+)`)
var ContentParameterCannotBeChangedRegexp = regexp.MustCompile(`^parameter ".+?" (changed|cannot be changed)`)
var ContentConfigFileContainsErrorsRegexp = regexp.MustCompile(`^configuration file ".+?" contains errors`)
var ContentWorkerProcessExitedRegexp = regexp.MustCompile(`^worker process: (.+?) \(PID (\d+)\) (?:exited with exit code (\d+)|was terminated by signal (\d+))`)
var ContentRegexpInvalidTimelineRegexp = regexp.MustCompile(`^according to history file, WAL location .+? belongs to timeline \d+, but previous recovered WAL file came from timeline \d+`)
var ContentUniqueConstraintViolationRegexp = regexp.MustCompile(`^duplicate key value violates unique constraint "(.+?)"`)
var ContentForeignKeyConstraintViolation1Regexp = regexp.MustCompile(`^insert or update on table "(.+?)" violates foreign key constraint "(.+?)"`)
var ContentForeignKeyConstraintViolation2Regexp = regexp.MustCompile(`^update or delete on table "(.+?)" violates foreign key constraint "(.+?)" on table "(.+?)"`)
var ContentNullConstraintViolationRegexp = regexp.MustCompile(`^null value in column "(.+?)" violates not-null constraint`)
var ContentCheckConstraintViolation1Regexp = regexp.MustCompile(`^new row for relation "(.+?)" violates check constraint "(.+?)"`)
var ContentCheckConstraintViolation2Regexp = regexp.MustCompile(`^check constraint "(.+?)" is violated by some row`)
var ContentCheckConstraintViolation3Regexp = regexp.MustCompile(`^column "(.+?)" of table "(.+?)" contains values that violate the new constraint`)
var ContentCheckConstraintViolation4Regexp = regexp.MustCompile(`^value for domain (.+?) violates check constraint "(.+?)"`)
var ContentExclusionConstraintViolationRegexp = regexp.MustCompile(`^conflicting key value violates exclusion constraint "(.+?)"`)
var ContentColumnMissingFromGroupByRegexp = regexp.MustCompile(`^column "(.+?)" must appear in the GROUP BY clause or be used in an aggregate function`)
var ContentColumnDoesNotExistRegexp = regexp.MustCompile(`^column "(.+?)" does not exist`)
var ContentColumnDoesNotExistOnTableRegexp = regexp.MustCompile(`^column "(.+?)" on "(.+?)" does not exist`)
var ContentColumnReferenceAmbiguous = regexp.MustCompile(`^column reference "(.+?)" is ambiguous`)
var ContentRelationDoesNotExist = regexp.MustCompile(`^relation "(.+?)" does not exist`)
var ContentFunctionDoesNotExist = regexp.MustCompile(`^function (.+?) does not exist`)
var ContentColumnCannotBeCastRegexp = regexp.MustCompile(`^column "(.+?)" cannot be cast to type "(.+?)"`)
var ContentCannotDropRegexp = regexp.MustCompile(`^cannot drop (.+?) because other objects depend on it`)
var ContentStatementLogRegexp = regexp.MustCompile(`^statement: (.*)`)
var ContentStatementLogExecuteRegexp = regexp.MustCompile(`^execute (.+?): (.*)`)
var ContentPgaCollectorIdentifyRegexp = regexp.MustCompile(`^pganalyze-collector-identify: (.*)`)

type autoExplainJsonPlanDetails struct {
	QueryText string                 `json:"Query Text"`
	Plan      map[string]interface{} `json:"Plan"`
}

var autoExplainTextPlanDetailsRegexp = regexp.MustCompile(`^Query Text: (.+)\s+([\s\S]+)`)

var parallelWorkerProcessTextRegexp = regexp.MustCompile(`^parallel worker for PID (\d+)`)

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

func classifyAndSetDetails(logLine state.LogLine, detailLine state.LogLine, samples []state.PostgresQuerySample) (state.LogLine, []state.PostgresQuerySample) {
	var parts []string

	// Connects/Disconnects
	if strings.HasPrefix(logLine.Content, "connection received: ") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_RECEIVED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "connection authorized: ") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_AUTHORIZED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "pg_hba.conf rejects connection ") || strings.HasPrefix(logLine.Content, "password authentication failed for user") || strings.HasPrefix(logLine.Content, "no pg_hba.conf entry for") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "Ident authentication failed for user") || strings.HasPrefix(logLine.Content, "could not connect to Ident server") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "database") {
		if ContentDatabaseNotAcceptingConnectionsRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "role") {
		if ContentRoleNotAllowedLoginRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_REJECTED
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "disconnection: ") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_DISCONNECTED
		parts = ContentDisconnectionRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 4 {
			timeSecs, _ := strconv.ParseFloat(parts[3], 64)
			timeMinutes, _ := strconv.ParseFloat(parts[2], 64)
			timeHours, _ := strconv.ParseFloat(parts[1], 64)
			logLine.Details = map[string]interface{}{"session_time_secs": timeSecs + timeMinutes*60 + timeHours*3600}
		}
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "incomplete startup packet") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_CLIENT_FAILED_TO_CONNECT
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "could not receive data from client") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "could not send data to client") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "connection to client lost") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "terminating connection because protocol synchronization was lost") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "unexpected EOF on client connection with an open transaction") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST_OPEN_TX
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "unexpected EOF on client connection") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_LOST
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "terminating connection due to administrator command") {
		logLine.Classification = pganalyze_collector.LogLineInformation_CONNECTION_TERMINATED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "remaining connection slots are reserved") {
		logLine.Classification = pganalyze_collector.LogLineInformation_OUT_OF_CONNECTIONS
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "too many connections for role") {
		logLine.Classification = pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_ROLE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "too many connections for database") {
		logLine.Classification = pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_DATABASE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "could not accept SSL connection: ") {
		logLine.Classification = pganalyze_collector.LogLineInformation_COULD_NOT_ACCEPT_SSL_CONNECTION
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "unsupported frontend protocol") {
		logLine.Classification = pganalyze_collector.LogLineInformation_PROTOCOL_ERROR_UNSUPPORTED_VERSION
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "incomplete message from client") {
		logLine.Classification = pganalyze_collector.LogLineInformation_PROTOCOL_ERROR_INCOMPLETE_MESSAGE
		return logLine, samples
	}

	// Checkpointer
	if strings.HasPrefix(logLine.Content, "checkpoint") || strings.HasPrefix(logLine.Content, "restartpoint") {
		parts = ContentCheckpointStartingRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			if parts[1] == "checkpoint" {
				logLine.Classification = pganalyze_collector.LogLineInformation_CHECKPOINT_STARTING
			} else if parts[1] == "restartpoint" {
				logLine.Classification = pganalyze_collector.LogLineInformation_RESTARTPOINT_STARTING
			}

			logLine.Details = map[string]interface{}{"reason": parts[2]}
			return logLine, samples
		}

		parts = ContentCheckpointCompleteRegexp.FindStringSubmatch(logLine.Content)
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
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "checkpoints") {
		parts = ContentCheckpointsTooFrequentRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECKPOINT_TOO_FREQUENT
			elapsedSecs, _ := strconv.ParseFloat(parts[1], 64)
			logLine.Details = map[string]interface{}{
				"elapsed_secs": elapsedSecs,
			}
		}
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "recovery restart point at") {
		logLine.Classification = pganalyze_collector.LogLineInformation_RESTARTPOINT_AT
		return logLine, samples
	}

	// WAL/Archiving
	if strings.HasPrefix(logLine.Content, "invalid record length") {
		logLine.Classification = pganalyze_collector.LogLineInformation_WAL_INVALID_RECORD_LENGTH
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "redo ") {
		logLine.Classification = pganalyze_collector.LogLineInformation_WAL_REDO
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "last completed transaction was at log time ") {
		parts = ContentRedoLastTxRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_WAL_REDO
			logLine.Details = map[string]interface{}{
				"last_transaction": parts[1],
			}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "archive command") {
		parts = ContentArchiveCommandFailedRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_WAL_ARCHIVE_COMMAND_FAILED
			logLine.Details = map[string]interface{}{}
			if parts[1] != "" {
				exitCode, _ := strconv.ParseInt(parts[1], 10, 32)
				logLine.Details["exit_code"] = exitCode
			}
			if parts[2] != "" {
				signal, _ := strconv.ParseInt(parts[2], 10, 32)
				logLine.Details["signal"] = signal
			}
			if detailLine.Content != "" {
				parts = ContentArchiveCommandFailedDetailsRegexp.FindStringSubmatch(detailLine.Content)
				if len(parts) == 2 {
					logLine.Details["archive_command"] = parts[1]
				}
			}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "archiver process") {
		logLine.Classification = pganalyze_collector.LogLineInformation_WAL_ARCHIVE_COMMAND_FAILED
		return logLine, samples
	}

	// Lock waits
	if strings.HasPrefix(logLine.Content, "process") {
		parts = ContentLockAcquiredRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 5 {
			logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_ACQUIRED
			afterMs, _ := strconv.ParseFloat(parts[4], 64)
			logLine.Details = map[string]interface{}{
				"lock_mode": parts[1],
				"lock_type": parts[2],
				"after_ms":  afterMs,
			}
			return logLine, samples
		}

		parts = ContentLockWaitRegexp.FindStringSubmatch(logLine.Content)
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
			if detailLine.Content != "" {
				parts = ContentLockWaitDetailsRegexp.FindStringSubmatch(detailLine.Content)
				if len(parts) == 3 {
					logLine.RelatedPids = []int32{}
					lockHolders := []int64{}
					for _, s := range strings.Split(parts[1], ", ") {
						i, _ := strconv.ParseInt(s, 10, 64)
						lockHolders = append(lockHolders, i)
						logLine.RelatedPids = append(logLine.RelatedPids, int32(i))
					}
					lockWaiters := []int64{}
					for _, s := range strings.Split(parts[2], ", ") {
						i, _ := strconv.ParseInt(s, 10, 64)
						lockWaiters = append(lockWaiters, i)
						logLine.RelatedPids = append(logLine.RelatedPids, int32(i))
					}
					logLine.Details["lock_holders"] = lockHolders
					logLine.Details["lock_waiters"] = lockWaiters
				}
			}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "deadlock detected") {
		logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_DETECTED
		logLine.RelatedPids = []int32{}
		allParts := ContentDeadlockDetailsRegexp.FindAllStringSubmatch(detailLine.Content, -1)
		for _, parts = range allParts {
			pid, _ := strconv.ParseInt(parts[1], 10, 32)
			logLine.RelatedPids = append(logLine.RelatedPids, int32(pid))
		}
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "canceling statement due to lock timeout") {
		logLine.Classification = pganalyze_collector.LogLineInformation_LOCK_TIMEOUT
		return logLine, samples
	}
	// Statement duration (log_min_duration_statement output) and auto_explain
	if strings.HasPrefix(logLine.Content, "duration: ") {
		// auto_explain needs to come before statement duration since its a subset of that regexp
		parts = ContentAutoExplainRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN
			runtime, _ := strconv.ParseFloat(parts[1], 64)
			logLine.Details = map[string]interface{}{"duration_ms": runtime}

			explainText := strings.TrimSpace(parts[2])
			if strings.HasPrefix(explainText, "{") { // json format
				var planDetails autoExplainJsonPlanDetails
				if strings.HasSuffix(explainText, "[Your log message was truncated]") {
					logLine.Details["truncated"] = true
				} else if err := json.Unmarshal([]byte(explainText), &planDetails); err != nil {
					logLine.Details["unparsed_explain_text"] = explainText
				} else {
					logLine.Query = strings.TrimSpace(planDetails.QueryText)
					explainJson, err := json.Marshal(planDetails.Plan)
					if err != nil {
						logLine.Details["unparsed_explain_text"] = explainText
					} else {
						sample := state.PostgresQuerySample{
							OccurredAt:    logLine.OccurredAt,
							Username:      logLine.Username,
							Database:      logLine.Database,
							Query:         logLine.Query,
							LogLineUUID:   logLine.UUID,
							RuntimeMs:     runtime,
							HasExplain:    true,
							ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
							ExplainFormat: pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT,
							// Reformat JSON so its the same as when using EXPLAIN (FORMAT JSON)
							ExplainOutput: "[{\"Plan\":" + string(explainJson) + "}]",
						}
						samples = append(samples, sample)
					}
				}
			} else if strings.HasPrefix(explainText, "Query Text:") { // text format
				explainParts := autoExplainTextPlanDetailsRegexp.FindStringSubmatch(explainText)

				if len(explainParts) == 3 {
					logLine.Query = strings.TrimSpace(explainParts[1])
					sample := state.PostgresQuerySample{
						OccurredAt:    logLine.OccurredAt,
						Username:      logLine.Username,
						Database:      logLine.Database,
						Query:         logLine.Query,
						LogLineUUID:   logLine.UUID,
						RuntimeMs:     runtime,
						HasExplain:    true,
						ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
						ExplainFormat: pganalyze_collector.QuerySample_TEXT_EXPLAIN_FORMAT,
						ExplainOutput: explainParts[2],
					}
					samples = append(samples, sample)
				} else {
					logLine.Details["unparsed_explain_text"] = explainText
				}
			}

			return logLine, samples
		}

		logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_DURATION
		if strings.HasSuffix(strings.TrimSpace(logLine.Content), "[Your log message was truncated]") {
			logLine.Details = map[string]interface{}{"truncated": true}
		} else {
			parts = ContentDurationRegexp.FindStringSubmatch(logLine.Content)

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
						parameterParts := ContentDurationDetailsRegexp.FindAllStringSubmatch(detailLine.Content, -1)
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

		return logLine, samples
	}

	// Statement log (log_statement output)
	if strings.HasPrefix(logLine.Content, "statement: ") {
		parts = ContentStatementLogRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_LOG
			logLine.Query = strings.TrimSpace(parts[1])
		}
	}
	if strings.HasPrefix(logLine.Content, "execute") {
		parts = ContentStatementLogExecuteRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_LOG
			logLine.Query = strings.TrimSpace(parts[2])
		}
	}

	// Statement cancellation (except lock timeout)
	if strings.HasPrefix(logLine.Content, "canceling statement due to statement timeout") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_TIMEOUT
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "canceling statement due to user request") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_USER
		return logLine, samples
	}

	// Autovacuum
	if strings.HasPrefix(logLine.Content, "canceling autovacuum task") {
		logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_CANCEL
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "database") {
		parts = ContentWraparoundWarningRegexp.FindStringSubmatch(logLine.Content)
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
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "database is not accepting commands to avoid wraparound") {
		parts = ContentWraparoundErrorRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 4 {
			logLine.Classification = pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_ERROR
			if parts[2] != "" {
				databaseOid, _ := strconv.ParseInt(parts[2], 10, 64)
				logLine.Details = map[string]interface{}{"database_oid": databaseOid}
			}
			if parts[3] != "" {
				logLine.Details = map[string]interface{}{"database_name": parts[3]}
			}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "autovacuum launcher started") {
		logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_STARTED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "autovacuum launcher shutting down") || strings.HasPrefix(logLine.Content, "terminating autovacuum process due to administrator command") {
		logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_SHUTTING_DOWN
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "automatic vacuum of table") {
		parts = ContentAutoVacuumRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 22 {
			var kernelPart, userPart, elapsedPart string

			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED
			// FIXME: Associate relation (parts[1])

			numIndexScans, _ := strconv.ParseInt(parts[2], 10, 64)
			pagesRemoved, _ := strconv.ParseInt(parts[3], 10, 64)
			relPages, _ := strconv.ParseInt(parts[4], 10, 64)
			tuplesDeleted, _ := strconv.ParseInt(parts[7], 10, 64)
			newRelTuples, _ := strconv.ParseInt(parts[8], 10, 64)
			newDeadTuples, _ := strconv.ParseInt(parts[9], 10, 64)
			vacuumPageHit, _ := strconv.ParseInt(parts[11], 10, 64)
			vacuumPageMiss, _ := strconv.ParseInt(parts[12], 10, 64)
			vacuumPageDirty, _ := strconv.ParseInt(parts[13], 10, 64)
			readRateMb, _ := strconv.ParseFloat(parts[14], 64)
			writeRateMb, _ := strconv.ParseFloat(parts[15], 64)

			if parts[16] != "" {
				kernelPart = parts[16]
				userPart = parts[17]
				elapsedPart = parts[18]
			} else {
				userPart = parts[19]
				kernelPart = parts[20]
				elapsedPart = parts[21]
			}
			rusageKernelMode, _ := strconv.ParseFloat(kernelPart, 64)
			rusageUserMode, _ := strconv.ParseFloat(userPart, 64)
			rusageElapsed, _ := strconv.ParseFloat(elapsedPart, 64)

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
			if parts[10] != "" {
				oldestXmin, _ := strconv.ParseInt(parts[10], 10, 64)
				logLine.Details["oldest_xmin"] = oldestXmin
			}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "automatic analyze of table") {
		parts = ContentAutoAnalyzeRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 8 {
			var kernelPart, userPart, elapsedPart string
			logLine.Classification = pganalyze_collector.LogLineInformation_AUTOANALYZE_COMPLETED
			// FIXME: Associate relation (parts[1])
			if parts[2] != "" {
				kernelPart = parts[2]
				userPart = parts[3]
				elapsedPart = parts[4]
			} else {
				userPart = parts[5]
				kernelPart = parts[6]
				elapsedPart = parts[7]
			}
			rusageKernelMode, _ := strconv.ParseFloat(kernelPart, 64)
			rusageUserMode, _ := strconv.ParseFloat(userPart, 64)
			rusageElapsed, _ := strconv.ParseFloat(elapsedPart, 64)
			logLine.Details = map[string]interface{}{
				"rusage_kernel": rusageKernelMode, "rusage_user": rusageUserMode,
				"elapsed_secs": rusageElapsed,
			}
			return logLine, samples
		}
	}

	// Server events
	if strings.HasPrefix(logLine.Content, "server process") {
		parts = ContentServerCrashedRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_CRASHED
			processPid, _ := strconv.ParseInt(parts[1], 10, 32)
			signal, _ := strconv.ParseInt(parts[2], 10, 32)
			logLine.Details = map[string]interface{}{
				"process_type": "server process",
				"process_pid":  processPid,
				"signal":       signal,
			}
			logLine.RelatedPids = []int32{int32(processPid)}
			return logLine, samples
		}
		parts = ContentServerOutOfMemoryRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY
			processPid, _ := strconv.ParseInt(parts[1], 10, 32)
			signal, _ := strconv.ParseInt(parts[2], 10, 32)
			logLine.Details = map[string]interface{}{
				"process_type": "server process",
				"process_pid":  processPid,
				"signal":       signal,
			}
			logLine.RelatedPids = []int32{int32(processPid)}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "terminating any other active server processes") || strings.HasPrefix(logLine.Content, "terminating connection because of crash of another server process") || strings.HasPrefix(logLine.Content, "all server processes terminated; reinitializing") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_CRASHED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "database system was shut down") || strings.HasPrefix(logLine.Content, "database system is ready") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "MultiXact member wraparound protections are now enabled") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "entering standby mode") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "redirecting log output to logging collector process") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "ending log output to stderr") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "database system was interrupted") || strings.HasPrefix(logLine.Content, "database system was not properly shut down") || strings.HasPrefix(logLine.Content, "database system shutdown was interrupted") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "received") {
		if ContentReceivedShutdownRequestRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "aborting any active transactions") || strings.HasPrefix(logLine.Content, "shutting down") || strings.HasPrefix(logLine.Content, "the database system is shutting down") || strings.HasPrefix(logLine.Content, "database system is shut down") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "out of memory") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "page verification failed") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "invalid page in block") {
		parts = ContentInvalidChecksumRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM
			blockNo, _ := strconv.ParseInt(parts[1], 10, 64)
			logLine.Details = map[string]interface{}{"block": blockNo, "file": parts[2]}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "temporary file") {
		parts = ContentTemporaryFileRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_TEMP_FILE_CREATED
			size, _ := strconv.ParseInt(parts[2], 10, 64)
			logLine.Details = map[string]interface{}{"size": size, "file": parts[1]}
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "could not open usermap file") ||
		strings.HasPrefix(logLine.Content, "could not link file") ||
		strings.HasPrefix(logLine.Content, "unexpected pageaddr") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_MISC
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "received SIGHUP, reloading configuration files") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_RELOAD
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "parameter") {
		if ContentParameterCannotBeChangedRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_RELOAD
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "configuration file") {
		if ContentConfigFileContainsErrorsRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_RELOAD
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "worker process: ") {
		parts = ContentWorkerProcessExitedRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 5 {
			logLine.Classification = pganalyze_collector.LogLineInformation_SERVER_PROCESS_EXITED
			processPid, _ := strconv.ParseInt(parts[2], 10, 32)
			logLine.RelatedPids = []int32{int32(processPid)}
			logLine.Details = map[string]interface{}{
				"process_type": parts[1],
				"process_pid":  processPid,
			}

			if parts[3] != "" {
				exitCode, _ := strconv.ParseInt(parts[3], 10, 32)
				logLine.Details["exit_code"] = exitCode
			}
			if parts[4] != "" {
				signal, _ := strconv.ParseInt(parts[4], 10, 32)
				logLine.Details["signal"] = signal
			}

			if strings.HasPrefix(parts[1], "parallel worker for PID") {
				textParts := parallelWorkerProcessTextRegexp.FindStringSubmatch(parts[1])
				if len(textParts) == 2 {
					parentPid, _ := strconv.ParseInt(textParts[1], 10, 32)
					logLine.Details["process_type"] = "parallel worker"
					logLine.Details["parent_pid"] = parentPid
					logLine.RelatedPids = append(logLine.RelatedPids, int32(parentPid))
				}
			}
			return logLine, samples
		}
	}

	// Standby
	if strings.HasPrefix(logLine.Content, "restored log file") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_RESTORED_WAL_FROM_ARCHIVE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "started streaming WAL") || strings.HasPrefix(logLine.Content, "restarted WAL streaming") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STARTED_STREAMING
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "could not receive data from WAL stream") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STREAMING_INTERRUPTED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "terminating walreceiver process") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STOPPED_STREAMING
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "consistent recovery state reached at") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_CONSISTENT_RECOVERY_STATE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "canceling statement due to conflict with recovery") {
		logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_STATEMENT_CANCELED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "according to history file, WAL location") {
		if ContentRegexpInvalidTimelineRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_STANDBY_INVALID_TIMELINE
			return logLine, samples
		}
	}

	// Constraint violations
	if strings.HasPrefix(logLine.Content, "duplicate key value violates unique constraint") {
		parts = ContentUniqueConstraintViolationRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_UNIQUE_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "insert or update on table") {
		parts = ContentForeignKeyConstraintViolation1Regexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name and relation name
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "update or delete on table") {
		parts = ContentForeignKeyConstraintViolation2Regexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 4 {
			logLine.Classification = pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name and both relation names
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "null value in column") {
		parts = ContentNullConstraintViolationRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_NOT_NULL_CONSTRAINT_VIOLATION
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "new row for relation") {
		parts = ContentCheckConstraintViolation1Regexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name and relation name
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "check constraint") {
		parts = ContentCheckConstraintViolation2Regexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "column") {
		parts = ContentCheckConstraintViolation3Regexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store relation name
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "value for domain") {
		parts = ContentCheckConstraintViolation4Regexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 3 {
			logLine.Classification = pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "conflicting key value violates exclusion constraint") {
		parts = ContentExclusionConstraintViolationRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_EXCLUSION_CONSTRAINT_VIOLATION
			// FIXME: Store constraint name
			return logLine, samples
		}
	}

	// Application errors
	if strings.HasPrefix(logLine.Content, "syntax error") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SYNTAX_ERROR
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "invalid input syntax for") {
		logLine.Classification = pganalyze_collector.LogLineInformation_INVALID_INPUT_SYNTAX
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "value too long for type") {
		logLine.Classification = pganalyze_collector.LogLineInformation_VALUE_TOO_LONG_FOR_TYPE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "invalid value") {
		logLine.Classification = pganalyze_collector.LogLineInformation_INVALID_VALUE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "malformed array literal") {
		logLine.Classification = pganalyze_collector.LogLineInformation_MALFORMED_ARRAY_LITERAL
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "subquery in FROM must have an alias") {
		logLine.Classification = pganalyze_collector.LogLineInformation_SUBQUERY_MISSING_ALIAS
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "INSERT has more expressions than target columns") {
		logLine.Classification = pganalyze_collector.LogLineInformation_INSERT_TARGET_COLUMN_MISMATCH
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "op ANY/ALL (array) requires array on right side") {
		logLine.Classification = pganalyze_collector.LogLineInformation_ANY_ALL_REQUIRES_ARRAY
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "column") {
		if ContentColumnMissingFromGroupByRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_COLUMN_MISSING_FROM_GROUP_BY
			return logLine, samples
		}
		if ContentColumnDoesNotExistRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_COLUMN_DOES_NOT_EXIST
			return logLine, samples
		}
		if ContentColumnDoesNotExistOnTableRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_COLUMN_DOES_NOT_EXIST
			return logLine, samples
		}
		if ContentColumnReferenceAmbiguous.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_COLUMN_REFERENCE_AMBIGUOUS
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "relation") {
		if ContentRelationDoesNotExist.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_RELATION_DOES_NOT_EXIST
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "function") {
		if ContentFunctionDoesNotExist.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_FUNCTION_DOES_NOT_EXIST
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "operator does not exist") {
		logLine.Classification = pganalyze_collector.LogLineInformation_OPERATOR_DOES_NOT_EXIST
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "permission denied") {
		logLine.Classification = pganalyze_collector.LogLineInformation_PERMISSION_DENIED
		// FIXME: Store relation name when this is "permission denied for relation [relation name]"
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "current transaction is aborted, commands ignored until end of transaction block") {
		logLine.Classification = pganalyze_collector.LogLineInformation_TRANSACTION_IS_ABORTED
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "there is no unique or exclusion constraint matching the ON CONFLICT specification") {
		logLine.Classification = pganalyze_collector.LogLineInformation_ON_CONFLICT_NO_CONSTRAINT_MATCH
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "ON CONFLICT DO UPDATE command cannot affect row a second time") {
		logLine.Classification = pganalyze_collector.LogLineInformation_ON_CONFLICT_ROW_AFFECTED_TWICE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "column") {
		if ContentColumnCannotBeCastRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_COLUMN_CANNOT_BE_CAST
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "division by zero") {
		logLine.Classification = pganalyze_collector.LogLineInformation_DIVISION_BY_ZERO
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "cannot drop") {
		if ContentCannotDropRegexp.MatchString(logLine.Content) {
			logLine.Classification = pganalyze_collector.LogLineInformation_CANNOT_DROP
			return logLine, samples
		}
	}
	if strings.HasPrefix(logLine.Content, "integer out of range") {
		logLine.Classification = pganalyze_collector.LogLineInformation_INTEGER_OUT_OF_RANGE
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "invalid regular expression: ") {
		logLine.Classification = pganalyze_collector.LogLineInformation_INVALID_REGEXP
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "there is no parameter $") {
		logLine.Classification = pganalyze_collector.LogLineInformation_PARAM_MISSING
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "no such savepoint") {
		logLine.Classification = pganalyze_collector.LogLineInformation_NO_SUCH_SAVEPOINT
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "unterminated quoted string at or near") {
		logLine.Classification = pganalyze_collector.LogLineInformation_UNTERMINATED_QUOTED_STRING
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "unterminated quoted identifier at or near") {
		logLine.Classification = pganalyze_collector.LogLineInformation_UNTERMINATED_QUOTED_IDENTIFIER
		return logLine, samples
	}
	if strings.HasPrefix(logLine.Content, "invalid byte sequence for encoding") {
		logLine.Classification = pganalyze_collector.LogLineInformation_INVALID_BYTE_SEQUENCE
		return logLine, samples
	}
	// pganalyze-collector-identify
	if strings.HasPrefix(logLine.Content, "pganalyze-collector-identify: ") {
		parts = ContentPgaCollectorIdentifyRegexp.FindStringSubmatch(logLine.Content)
		if len(parts) == 2 {
			logLine.Classification = pganalyze_collector.LogLineInformation_PGA_COLLECTOR_IDENTIFY
			logLine.Details = map[string]interface{}{
				"config_section": strings.TrimSpace(parts[1]),
			}
		}
		return logLine, samples
	}

	return logLine, samples
}

func AnalyzeBackendLogLines(logLines []state.LogLine) (logLinesOut []state.LogLine, samples []state.PostgresQuerySample) {
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

		logLine, samples = classifyAndSetDetails(logLine, detailLine, samples)

		logLinesOut = append(logLinesOut, logLine)
	}

	// Ensure no other part of the system accidentally sends log line contents, as
	// they should be considered opaque from here on
	for idx := range logLinesOut {
		logLinesOut[idx].Content = ""
	}

	return
}
