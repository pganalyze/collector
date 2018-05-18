package logs_test

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

type testpair struct {
	logLinesIn  []state.LogLine
	logLinesOut []state.LogLine
	samplesOut  []state.PostgresQuerySample
}

var tests = []testpair{
	// Statement duration
	{
		[]state.LogLine{{
			Content: "duration: 3205.800 ms execute a2: SELECT \"servers\".* FROM \"servers\" WHERE \"servers\".\"id\" = 1 LIMIT 2",
		}},
		[]state.LogLine{{
			Query:          "SELECT \"servers\".* FROM \"servers\" WHERE \"servers\".\"id\" = 1 LIMIT 2",
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			Details:        map[string]interface{}{"duration_ms": 3205.8},
		}},
		[]state.PostgresQuerySample{{
			Query:     "SELECT \"servers\".* FROM \"servers\" WHERE \"servers\".\"id\" = 1 LIMIT 2",
			RuntimeMs: 3205.8,
		}},
	},
	{
		[]state.LogLine{{
			Content:  "duration: 4079.697 ms execute <unnamed>: \nSELECT * FROM x WHERE y = $1 LIMIT $2",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "parameters: $1 = 'long string', $2 = '1'",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			Query:          "SELECT * FROM x WHERE y = $1 LIMIT $2",
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details:        map[string]interface{}{"duration_ms": 4079.697},
		}, {
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.PostgresQuerySample{{
			Query:      "SELECT * FROM x WHERE y = $1 LIMIT $2",
			RuntimeMs:  4079.697,
			Parameters: []string{"long string", "1"},
		}},
	},
	{
		[]state.LogLine{{
			Content: "duration: 3205.800 ms execute a2: SELECT ...[Your log message was truncated]",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			Details:        map[string]interface{}{"truncated": true},
		}},
		nil,
	},
	// Statement log
	{
		[]state.LogLine{{
			Content:  "execute <unnamed>: SELECT $1, $2",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "parameters: $1 = '1', $2 = 't'",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_LOG,
			UUID:           uuid.UUID{1},
			Query:          "SELECT $1, $2",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "statement: EXECUTE x(1);",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "prepare: PREPARE x AS SELECT $1;",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_LOG,
			UUID:           uuid.UUID{1},
			Query:          "EXECUTE x(1);",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	// Connects/Disconnects
	{
		[]state.LogLine{{
			Content: "connection received: host=172.30.0.165 port=56902",
		}, {
			Content: "connection authorized: user=myuser database=mydb SSL enabled (protocol=TLSv1.2, cipher=ECDHE-RSA-AES256-GCM-SHA384, compression=off)",
		}, {
			Content: "pg_hba.conf rejects connection for host \"172.1.0.1\", user \"myuser\", database \"mydb\", SSL on",
		}, {
			Content:  "no pg_hba.conf entry for host \"8.8.8.8\", user \"postgres\", database \"postgres\", SSL off",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content:  "password authentication failed for user \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Connection matched pg_hba.conf line 4: \"hostssl postgres        postgres        0.0.0.0/0               md5\"",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "database \"template0\" is not currently accepting connections",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content:  "role \"abc\" is not permitted to log in",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content: "could not connect to Ident server at address \"127.0.0.1\", port 113: Connection refused",
		}, {
			Content:  "Ident authentication failed for user \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content: "disconnection: session time: 1:53:01.198 user=myuser database=mydb host=172.30.0.165 port=56902",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_RECEIVED,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_AUTHORIZED,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_DISCONNECTED,
			Details:        map[string]interface{}{"session_time_secs": 6781.198},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "incomplete startup packet",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_CLIENT_FAILED_TO_CONNECT,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "could not receive data from client: Connection reset by peer",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_LOST,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "could not send data to client: Broken pipe",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_LOST,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "terminating connection because protocol synchronization was lost",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_LOST,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "connection to client lost",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_LOST,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "unexpected EOF on client connection",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_LOST,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "terminating connection due to administrator command",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_TERMINATED,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unexpected EOF on client connection with an open transaction",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_LOST_OPEN_TX,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "remaining connection slots are reserved for non-replication superuser connections",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_OUT_OF_CONNECTIONS,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "too many connections for role \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_ROLE,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "too many connections for database \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_DATABASE,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "could not accept SSL connection: EOF detected",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "could not accept SSL connection: Connection reset by peer",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_COULD_NOT_ACCEPT_SSL_CONNECTION,
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_COULD_NOT_ACCEPT_SSL_CONNECTION,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unsupported frontend protocol 65363.12345: server supports 1.0 to 3.0",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_PROTOCOL_ERROR_UNSUPPORTED_VERSION,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "incomplete message from client",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "COPY \"abc\" (\"x\") FROM STDIN BINARY",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}, {
			Content:  "COPY abc, line 1234, column x",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_PROTOCOL_ERROR_INCOMPLETE_MESSAGE,
			Query:          "COPY \"abc\" (\"x\") FROM STDIN BINARY",
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	// Checkpoints
	{
		[]state.LogLine{{
			Content: "checkpoint starting: xlog",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CHECKPOINT_STARTING,
			Details:        map[string]interface{}{"reason": "xlog"},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "checkpoint complete: wrote 111906 buffers (10.9%); 0 WAL file(s) added, 22 removed, 29 recycled; write=215.895 s, sync=0.014 s, total=216.130 s; sync files=94, longest=0.014 s, average=0.000 s; distance=850730 kB, estimate=910977 kB",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CHECKPOINT_COMPLETE,
			Details: map[string]interface{}{
				"bufs_written_pct": 10.9, "write_secs": 215.895, "sync_secs": 0.014,
				"total_secs": 216.130, "longest_secs": 0.014, "average_secs": 0.0,
				"bufs_written": 111906, "segs_added": 0, "segs_removed": 22, "segs_recycled": 29,
				"sync_rels": 94, "distance_kb": 850730, "estimate_kb": 910977,
			},
		}},
		nil,
	},
	{ // Pre 10 syntax (WAL instead of transaction files)
		[]state.LogLine{{
			Content: "checkpoint complete: wrote 111906 buffers (10.9%); 0 transaction log file(s) added, 22 removed, 29 recycled; write=215.895 s, sync=0.014 s, total=216.130 s; sync files=94, longest=0.014 s, average=0.000 s; distance=850730 kB, estimate=910977 kB",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CHECKPOINT_COMPLETE,
			Details: map[string]interface{}{
				"bufs_written_pct": 10.9, "write_secs": 215.895, "sync_secs": 0.014,
				"total_secs": 216.130, "longest_secs": 0.014, "average_secs": 0.0,
				"bufs_written": 111906, "segs_added": 0, "segs_removed": 22, "segs_recycled": 29,
				"sync_rels": 94, "distance_kb": 850730, "estimate_kb": 910977,
			},
		}},
		nil,
	},
	{ // Pre 9.5 syntax (without distance/estimate)
		[]state.LogLine{{
			Content: "checkpoint complete: wrote 15047 buffers (1.4%); 0 transaction log file(s) added, 0 removed, 30 recycled; write=68.980 s, sync=1.542 s, total=70.548 s; sync files=925, longest=0.216 s, average=0.001 s",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CHECKPOINT_COMPLETE,
			Details: map[string]interface{}{
				"bufs_written": 15047, "segs_added": 0, "segs_removed": 0, "segs_recycled": 30,
				"sync_rels":        925,
				"bufs_written_pct": 1.4, "write_secs": 68.98, "sync_secs": 1.542, "total_secs": 70.548,
				"longest_secs": 0.216, "average_secs": 0.001},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "checkpoints are occurring too frequently (18 seconds apart)",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Consider increasing the configuration parameter \"max_wal_size\".",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CHECKPOINT_TOO_FREQUENT,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"elapsed_secs": 18,
			},
			UUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "restartpoint starting: shutdown immediate",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_RESTARTPOINT_STARTING,
			Details:        map[string]interface{}{"reason": "shutdown immediate"},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content: "restartpoint complete: wrote 693 buffers (0.1%); 0 transaction log file(s) added, 0 removed, 5 recycled; write=0.015 s, sync=0.240 s, total=0.288 s; sync files=74, longest=0.024 s, average=0.003 s; distance=81503 kB, estimate=81503 kB",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_RESTARTPOINT_COMPLETE,
			Details: map[string]interface{}{
				"bufs_written_pct": 0.1, "write_secs": 0.015, "sync_secs": 0.240,
				"total_secs": 0.288, "longest_secs": 0.024, "average_secs": 0.003,
				"bufs_written": 693, "segs_added": 0, "segs_removed": 0, "segs_recycled": 5,
				"sync_rels": 74, "distance_kb": 81503, "estimate_kb": 81503,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "recovery restart point at 4E8/9B13FBB0",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "last completed transaction was at log time 2017-05-05 20:17:06.511443+00",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_RESTARTPOINT_AT,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	// WAL/Archiving
	{
		[]state.LogLine{{
			Content: "invalid record length at 4E8/9E0979A8",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_WAL_INVALID_RECORD_LENGTH,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "redo starts at 4E8/9B13FBB0",
		}, {
			Content: "redo is not required",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_WAL_REDO,
		}, {
			Classification: pganalyze_collector.LogLineInformation_WAL_REDO,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "redo starts at 25DFA/2B000098",
		}, {
			Content: "invalid record length at 25DFA/2B000548: wanted 24, got 0",
		}, {
			Content: "redo done at 25DFA/2B000500",
		}, {
			Content: "last completed transaction was at log time 2018-03-12 02:09:12.585354+00",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_WAL_REDO,
		}, {
			Classification: pganalyze_collector.LogLineInformation_WAL_INVALID_RECORD_LENGTH,
		}, {
			Classification: pganalyze_collector.LogLineInformation_WAL_REDO,
		}, {
			Classification: pganalyze_collector.LogLineInformation_WAL_REDO,
			Details: map[string]interface{}{
				"last_transaction": "2018-03-12 02:09:12.585354+00",
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "archive command failed with exit code 1",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "The failed archive command was: /etc/rds/dbbin/pgscripts/rds_wal_archive pg_xlog/0000000100025DFA00000023",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_WAL_ARCHIVE_COMMAND_FAILED,
			UUID:           uuid.UUID{1},
			Details: map[string]interface{}{
				"archive_command": "/etc/rds/dbbin/pgscripts/rds_wal_archive pg_xlog/0000000100025DFA00000023",
				"exit_code":       1,
			},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "archive command was terminated by signal 6: Abort trap",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "The failed archive command was: /usr/local/bin/envdir /usr/local/etc/wal-e.d/env /usr/local/bin/wal-e wal-push pg_xlog/000000040000023B000000CC",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "archiver process (PID 5886) exited with exit code 1",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
			Classification: pganalyze_collector.LogLineInformation_WAL_ARCHIVE_COMMAND_FAILED,
			UUID:           uuid.UUID{1},
			Details: map[string]interface{}{
				"archive_command": "/usr/local/bin/envdir /usr/local/etc/wal-e.d/env /usr/local/bin/wal-e wal-push pg_xlog/000000040000023B000000CC",
				"signal":          6,
			},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_WAL_ARCHIVE_COMMAND_FAILED,
		}},
		nil,
	},
	// Lock waits
	{
		[]state.LogLine{{
			Content:  "process 583 acquired AccessExclusiveLock on relation 185044 of database 16384 after 2175.443 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "ALTER TABLE x ADD COLUMN y text;",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_LOCK_ACQUIRED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Query:          "ALTER TABLE x ADD COLUMN y text;",
			Details: map[string]interface{}{
				"after_ms":  2175.443,
				"lock_mode": "AccessExclusiveLock",
				"lock_type": "relation",
			},
			UUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "process 25307 acquired ExclusiveLock on tuple (106,38) of relation 16421 of database 16385 after 1129279.295 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_LOCK_ACQUIRED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"after_ms":  1129279.295,
				"lock_mode": "ExclusiveLock",
				"lock_type": "tuple",
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "process 21813 acquired ExclusiveLock on extension of relation 419652 of database 16400 after 1003.994 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_LOCK_ACQUIRED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"after_ms":  1003.994,
				"lock_mode": "ExclusiveLock",
				"lock_type": "extension",
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "process 2078 still waiting for ShareLock on transaction 1045207414 after 1000.100 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Process holding the lock: 583. Wait queue: 2078, 456",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "INSERT INTO x (y) VALUES (1)",
			LogLevel: pganalyze_collector.LogLineInformation_QUERY,
		}, {
			Content:  "PL/pgSQL function insert_helper(text) line 5 at EXECUTE statement",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "SELECT insert_helper($1)",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_LOCK_WAITING,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Query:          "SELECT insert_helper($1)",
			Details: map[string]interface{}{
				"lock_holders": []int64{583},
				"lock_waiters": []int64{2078, 456},
				"after_ms":     1000.1,
				"lock_mode":    "ShareLock",
				"lock_type":    "transactionid",
			},
			RelatedPids: []int32{583, 2078, 456},
			UUID:        uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_QUERY,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "canceling statement due to lock timeout",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "while updating tuple (24,41) in relation \"mytable\"",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "UPDATE mytable SET y = 2 WHERE x = 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_LOCK_TIMEOUT,
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Query:          "UPDATE mytable SET y = 2 WHERE x = 1",
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "process 2078 avoided deadlock for AccessExclusiveLock on relation 999 by rearranging queue order after 123.456 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Processes holding the lock: 583, 123. Wait queue: 2078",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_AVOIDED,
			Details: map[string]interface{}{
				"lock_holders": []int64{583, 123},
				"lock_waiters": []int64{2078},
				"after_ms":     123.456,
				"lock_mode":    "AccessExclusiveLock",
				"lock_type":    "relation",
			},
			RelatedPids: []int32{583, 123, 2078},
			UUID:        uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "process 123 detected deadlock while waiting for AccessExclusiveLock on extension of relation 666 of database 123 after 456.000 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_DETECTED,
			Details: map[string]interface{}{
				"lock_mode": "AccessExclusiveLock",
				"lock_type": "extend",
				"after_ms":  456.0,
			},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "deadlock detected",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content: "Process 9788 waits for ShareLock on transaction 1035; blocked by process 91." +
				"\nProcess 91 waits for ShareLock on transaction 1045; blocked by process 98.\n" +
				"\nProcess 98: INSERT INTO x (id, name, email) VALUES (1, 'ABC', 'abc@example.com') ON CONFLICT(email) DO UPDATE SET name = excluded.name, /* truncated */" +
				"\nProcess 91: INSERT INTO x (id, name, email) VALUES (1, 'ABC', 'abc@example.com') ON CONFLICT(email) DO UPDATE SET name = excluded.name, /* truncated */",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "See server log for query details.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "while inserting index tuple (1,42) in relation \"x\"",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "INSERT INTO x (id, name, email) VALUES (1, 'ABC', 'abc@example.com') ON CONFLICT(email) DO UPDATE SET name = excluded.name RETURNING id",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_DETECTED,
			Query:          "INSERT INTO x (id, name, email) VALUES (1, 'ABC', 'abc@example.com') ON CONFLICT(email) DO UPDATE SET name = excluded.name RETURNING id",
			UUID:           uuid.UUID{1},
			RelatedPids:    []int32{9788, 91, 98, 91},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "process 663 still waiting for ShareLock on virtual transaction 2/7 after 1000.123 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_LOCK_WAITING,
			Details: map[string]interface{}{
				"lock_mode": "ShareLock",
				"lock_type": "virtualxid",
				"after_ms":  1000.123,
			},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "process 663 still waiting for ExclusiveLock on advisory lock [233136,1,2,2] after 1000.365 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Process holding the lock: 660. Wait queue: 663.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "SELECT pg_advisory_lock(1, 2);",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_LOCK_WAITING,
			Details: map[string]interface{}{
				"lock_mode":    "ExclusiveLock",
				"lock_type":    "advisory",
				"lock_holders": []int64{660},
				"lock_waiters": []int64{663},
				"after_ms":     1000.365,
			},
			RelatedPids: []int32{660, 663},
			Query:       "SELECT pg_advisory_lock(1, 2);",
			UUID:        uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	// Autovacuum
	{
		[]state.LogLine{{
			Content:  "canceling autovacuum task",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "automatic analyze of table \"dbname.schemaname.tablename\"",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_CANCEL,
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "database \"template1\" must be vacuumed within 938860 transactions",
			LogLevel: pganalyze_collector.LogLineInformation_WARNING,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "To avoid a database shutdown, execute a full-database VACUUM in \"template1\".\nYou might also need to commit or roll back old prepared transactions.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_WARNING,
			Classification: pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_WARNING,
			Details: map[string]interface{}{
				"database_name":  "template1",
				"remaining_xids": 938860,
			},
			UUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "database with OID 10 must be vacuumed within 100 transactions",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_WARNING,
			Details: map[string]interface{}{
				"database_oid":   10,
				"remaining_xids": 100,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "database is not accepting commands to avoid wraparound data loss in database \"mydb\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Stop the postmaster and use a standalone backend to vacuum that database. You might also need to commit or roll back old prepared transactions.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_ERROR,
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Details: map[string]interface{}{
				"database_name": "mydb",
			},
			UUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "database is not accepting commands to avoid wraparound data loss in database with OID 16384",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_TXID_WRAPAROUND_ERROR,
			Details: map[string]interface{}{
				"database_oid": 16384,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "autovacuum launcher started",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_STARTED,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "autovacuum launcher shutting down",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_SHUTTING_DOWN,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "terminating autovacuum process due to administrator command",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_SHUTTING_DOWN,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"mydb.public.vac_test\": index scans: 1" +
				"\n pages: 0 removed, 1 remain, 0 skipped due to pins, 0 skipped frozen" +
				"\n tuples: 3 removed, 6 remain, 0 are dead but not yet removable" +
				"\n buffer usage: 70 hits, 4 misses, 4 dirtied" +
				"\n avg read rate: 62.877 MB/s, avg write rate: 62.877 MB/s" +
				"\n system usage: CPU 0.00s/0.00u sec elapsed 0.00 sec",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"num_index_scans":     1,
				"pages_removed":       0,
				"rel_pages":           1,
				"pinskipped_pages":    0,
				"frozenskipped_pages": 0,
				"tuples_deleted":      3,
				"new_rel_tuples":      6,
				"new_dead_tuples":     0,
				"vacuum_page_hit":     70,
				"vacuum_page_miss":    4,
				"vacuum_page_dirty":   4,
				"read_rate_mb":        62.877,
				"write_rate_mb":       62.877,
				"rusage_kernel":       0,
				"rusage_user":         0,
				"elapsed_secs":        0,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"postgres.public.pgbench_branches\": index scans: 1" +
				"\npages: 0 removed, 12 remain" +
				"\ntuples: 423 removed, 107 remain, 3 are dead but not yet removable" +
				"\nbuffer usage: 52 hits, 1 misses, 1 dirtied" +
				"\navg read rate: 7.455 MB/s, avg write rate: 7.455 MB/s" +
				"\nsystem usage: CPU 0.00s/0.00u sec elapsed 0.00 sec",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"num_index_scans":   1,
				"pages_removed":     0,
				"rel_pages":         12,
				"tuples_deleted":    423,
				"new_rel_tuples":    107,
				"new_dead_tuples":   3,
				"vacuum_page_hit":   52,
				"vacuum_page_miss":  1,
				"vacuum_page_dirty": 1,
				"read_rate_mb":      7.455,
				"write_rate_mb":     7.455,
				"rusage_kernel":     0,
				"rusage_user":       0,
				"elapsed_secs":      0,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"my_db.public.my_dimension\": index scans: 1" +
				"\n  pages: 0 removed, 29457 remain" +
				"\n  tuples: 3454 removed, 429481 remain, 0 are dead but not yet removable" +
				"\n  buffer usage: 64215 hits, 8056 misses, 22588 dirtied" +
				"\n  avg read rate: 1.018 MB/s, avg write rate: 2.855 MB/s" +
				"\n  system usage: CPU 0.10s/0.88u sec elapsed 61.80 seconds",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"num_index_scans":   1,
				"pages_removed":     0,
				"rel_pages":         29457,
				"tuples_deleted":    3454,
				"new_rel_tuples":    429481,
				"new_dead_tuples":   0,
				"vacuum_page_hit":   64215,
				"vacuum_page_miss":  8056,
				"vacuum_page_dirty": 22588,
				"read_rate_mb":      1.018,
				"write_rate_mb":     2.855,
				"rusage_kernel":     0.10,
				"rusage_user":       0.88,
				"elapsed_secs":      61.80,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"mydb.public.mytable\": index scans: 1" +
				" pages: 0 removed, 597092 remain, 0 skipped due to pins" +
				"	tuples: 466347 removed, 17314747 remain, 0 are dead but not yet removable" +
				"	buffer usage: 1854343 hits, 1447635 misses, 272945 dirtied" +
				"	avg read rate: 5.215 MB/s, avg write rate: 0.983 MB/s" +
				"	system usage: CPU 2.86s/16.36u sec elapsed 2168.76 sec",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"num_index_scans":   1,
				"pages_removed":     0,
				"rel_pages":         597092,
				"pinskipped_pages":  0,
				"tuples_deleted":    466347,
				"new_rel_tuples":    17314747,
				"new_dead_tuples":   0,
				"vacuum_page_hit":   1854343,
				"vacuum_page_miss":  1447635,
				"vacuum_page_dirty": 272945,
				"read_rate_mb":      5.215,
				"write_rate_mb":     0.983,
				"rusage_kernel":     2.86,
				"rusage_user":       16.36,
				"elapsed_secs":      2168.76,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"demo_pgbench.public.pgbench_tellers\": index scans: 0" +
				" pages: 0 removed, 839 remain, 0 skipped due to pins, 705 skipped frozen" +
				"	tuples: 1849 removed, 2556 remain, 5 are dead but not yet removable, oldest xmin: 448424944" +
				"	buffer usage: 569 hits, 1 misses, 0 dirtied" +
				"	avg read rate: 0.064 MB/s, avg write rate: 0.000 MB/s" +
				"	system usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.12 s",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"num_index_scans":     0,
				"pages_removed":       0,
				"rel_pages":           839,
				"pinskipped_pages":    0,
				"frozenskipped_pages": 705,
				"tuples_deleted":      1849,
				"new_rel_tuples":      2556,
				"new_dead_tuples":     5,
				"oldest_xmin":         448424944,
				"vacuum_page_hit":     569,
				"vacuum_page_miss":    1,
				"vacuum_page_dirty":   0,
				"read_rate_mb":        0.064,
				"write_rate_mb":       0.000,
				"rusage_kernel":       0.00,
				"rusage_user":         0.00,
				"elapsed_secs":        0.12,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "automatic analyze of table \"postgres.public.pgbench_branches\" system usage: CPU 1.02s/2.08u sec elapsed 108.25 sec",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOANALYZE_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"rusage_kernel": 1.02,
				"rusage_user":   2.08,
				"elapsed_secs":  108.25,
			},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "automatic analyze of table \"demo_pgbench.public.pgbench_history\" system usage: CPU: user: 0.23 s, system: 0.01 s, elapsed: 0.89 s",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOANALYZE_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"rusage_kernel": 0.01,
				"rusage_user":   0.23,
				"elapsed_secs":  0.89,
			},
		}},
		nil,
	},
	// Statement cancellation (other than lock timeout)
	{
		[]state.LogLine{{
			Content:  "canceling statement due to user request",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_USER,
			Query:          "SELECT 1",
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "canceling statement due to statement timeout",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_TIMEOUT,
			Query:          "SELECT 1",
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	// Server events
	{
		[]state.LogLine{{
			Content:  "server process (PID 660) was terminated by signal 6: Aborted",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Failed process was running: SELECT pg_advisory_lock(1, 2);",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "terminating any other active server processes",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "terminating connection because of crash of another server process",
			LogLevel: pganalyze_collector.LogLineInformation_WARNING,
			UUID:     uuid.UUID{2},
		}, {
			Content:  "The postmaster has commanded this server process to roll back the current transaction and exit, because another server process exited abnormally and possibly corrupted shared memory.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "In a moment you should be able to reconnect to the database and repeat your command.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "all server processes terminated; reinitializing",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			UUID:           uuid.UUID{1},
			RelatedPids:    []int32{660},
			Details: map[string]interface{}{
				"process_type": "server process",
				"process_pid":  660,
				"signal":       6,
			},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:       pganalyze_collector.LogLineInformation_WARNING,
			UUID:           uuid.UUID{2},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{2},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{2},
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "redirecting log output to logging collector process",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "ending log output to stderr",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "database system was shut down at 2017-05-03 23:23:37 UTC",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "MultiXact member wraparound protections are now enabled",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "database system is ready to accept connections",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "database system was shut down in recovery at 2017-05-05 20:17:07 UTC",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "entering standby mode",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "database system is ready to accept read only connections",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "database system was interrupted; last known up at 2017-05-07 22:33:02 UTC",
		}, {
			Content: "database system was not properly shut down; automatic recovery in progress",
		}, {
			Content:  "database system shutdown was interrupted; last known up at 2017-05-05 20:17:07 UTC",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "database system was interrupted while in recovery at 2017-05-05 20:17:07 UTC",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "This probably means that some data is corrupted and you will have to use the last backup for recovery.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "database system was interrupted while in recovery at log time 2017-05-05 20:17:07 UTC",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{2},
		}, {
			Content:  "If this has occurred more than once some data might be corrupted and you might need to choose an earlier recovery target.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			UUID:           uuid.UUID{2},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{2},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content: "received smart shutdown request",
		}, {
			Content: "received fast shutdown request",
		}, {
			Content: "aborting any active transactions",
		}, {
			Content: "shutting down",
		}, {
			Content: "the database system is shutting down",
		}, {
			Content: "database system is shut down",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "temporary file: path \"base/pgsql_tmp/pgsql_tmp15967.0\", size 200204288",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "alter table pgbench_accounts add primary key (aid)",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_SERVER_TEMP_FILE_CREATED,
			UUID:           uuid.UUID{1},
			Query:          "alter table pgbench_accounts add primary key (aid)",
			Details: map[string]interface{}{
				"file": "base/pgsql_tmp/pgsql_tmp15967.0",
				"size": 200204288,
			},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content: "could not open usermap file \"/var/lib/pgsql/9.5/data/pg_ident.conf\": No such file or directory",
		}, {
			Content: "could not link file \"pg_xlog/xlogtemp.26115\" to \"pg_xlog/000000010000021B000000C5\": File exists",
		}, {
			Content: "unexpected pageaddr 2D5/12000000 in log segment 00000001000002D500000022, offset 0",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_MISC,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_MISC,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_MISC,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "out of memory",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Failed on request of size 324589128.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "SELECT 123",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY,
			UUID:           uuid.UUID{1},
			Query:          "SELECT 123",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "server process (PID 123) was terminated by signal 9: Killed",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY,
			Details: map[string]interface{}{
				"process_type": "server process",
				"process_pid":  123,
				"signal":       9,
			},
			RelatedPids: []int32{123},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "page verification failed, calculated checksum 20919 but expected 15254",
			LogLevel: pganalyze_collector.LogLineInformation_WARNING,
		}, {
			Content:  "invalid page in block 335458 of relation base/16385/99454",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_WARNING,
			Classification: pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM,
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM,
			UUID:           uuid.UUID{1},
			Query:          "SELECT 1",
			Details: map[string]interface{}{
				"block": 335458,
				"file":  "base/16385/99454",
			},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "received SIGHUP, reloading configuration files",
		}, {
			Content: "parameter \"log_autovacuum_min_duration\" changed to \"0\"",
		}, {
			Content: "parameter \"shared_preload_libraries\" cannot be changed without restarting the server",
		}, {
			Content: "configuration file \"/var/lib/postgresql/data/postgresql.auto.conf\" contains errors; unaffected changes were applied",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_RELOAD,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_RELOAD,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_RELOAD,
		}, {
			Classification: pganalyze_collector.LogLineInformation_SERVER_RELOAD,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "worker process: parallel worker for PID 30491 (PID 31458) exited with exit code 1",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Failed process was running: SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_PROCESS_EXITED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			UUID:           uuid.UUID{1},
			Details: map[string]interface{}{
				"process_type": "parallel worker",
				"process_pid":  31458,
				"parent_pid":   30491,
				"exit_code":    1,
			},
			RelatedPids: []int32{31458, 30491},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "worker process: logical replication launcher (PID 17443) exited with exit code 1",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_PROCESS_EXITED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"process_type": "logical replication launcher",
				"process_pid":  17443,
				"exit_code":    1,
			},
			RelatedPids: []int32{17443},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "worker process: logical replication launcher (PID 17443) was terminated by signal 9",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_SERVER_PROCESS_EXITED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Details: map[string]interface{}{
				"process_type": "logical replication launcher",
				"process_pid":  17443,
				"signal":       9,
			},
			RelatedPids: []int32{17443},
		}},
		nil,
	},
	// Standby
	{
		[]state.LogLine{{
			Content:  "restored log file \"00000006000004E80000009C\" from archive",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STANDBY_RESTORED_WAL_FROM_ARCHIVE,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "started streaming WAL from primary at 4E8/9E000000 on timeline 6",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STANDBY_STARTED_STREAMING,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "restarted WAL streaming at 3E/62000000 on timeline 3",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STANDBY_STARTED_STREAMING,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "could not receive data from WAL stream: SSL error: sslv3 alert unexpected message",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STANDBY_STREAMING_INTERRUPTED,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "terminating walreceiver process due to administrator command",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STANDBY_STOPPED_STREAMING,
			LogLevel:       pganalyze_collector.LogLineInformation_FATAL,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "consistent recovery state reached at 4E8/9E0979A8",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STANDBY_CONSISTENT_RECOVERY_STATE,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "canceling statement due to conflict with recovery",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "User query might have needed to see row versions that must be removed.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_STANDBY_STATEMENT_CANCELED,
			Query:          "SELECT 1",
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "according to history file, WAL location 2D5/22000000 belongs to timeline 3, but previous recovered WAL file came from timeline 4",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_STANDBY_INVALID_TIMELINE,
		}},
		nil,
	},
	// Constraint violations
	{
		[]state.LogLine{{
			Content:  "duplicate key value violates unique constraint \"test_constraint\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Key (b, c)=(12345, mysecretdata) already exists.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "INSERT INTO a (b, c) VALUES ($1,$2) RETURNING id",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_UNIQUE_CONSTRAINT_VIOLATION,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO a (b, c) VALUES ($1,$2) RETURNING id",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "insert or update on table \"weather\" violates foreign key constraint \"weather_city_fkey\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Key (city)=(Berkeley) is not present in table \"cities\".",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "INSERT INTO weather VALUES ('Berkeley', 45, 53, 0.0, '1994-11-28');",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO weather VALUES ('Berkeley', 45, 53, 0.0, '1994-11-28');",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "update or delete on table \"test\" violates foreign key constraint \"test_fkey\" on table \"othertest\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Key (id)=(123) is still referenced from table \"othertest\".",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "DELETE FROM test WHERE id = 123",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION,
			UUID:           uuid.UUID{1},
			Query:          "DELETE FROM test WHERE id = 123",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "null value in column \"mycolumn\" violates not-null constraint",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Failing row contains (null).",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "INSERT INTO \"test\" (\"mycolumn\") VALUES ($1) RETURNING \"id\"",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_NOT_NULL_CONSTRAINT_VIOLATION,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO \"test\" (\"mycolumn\") VALUES ($1) RETURNING \"id\"",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "new row for relation \"test\" violates check constraint \"positive_value_check\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Failing row contains (-123).",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "check constraint \"valid_tag\" is violated by some row",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}, {
			Content:  "column \"mycolumn\" of table \"test\" contains values that violate the new constraint",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}, {
			Content:  "value for domain mydomain violates check constraint \"mydomain_check\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
			UUID:           uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "conflicting key value violates exclusion constraint \"reservation_during_excl\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Key (during)=([\"2010-01-01 14:45:00\",\"2010-01-01 15:45:00\")) conflicts with existing key (during)=([\"2010-01-01 11:30:00\",\"2010-01-01 15:00:00\")).",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "INSERT INTO reservation VALUES ('[2010-01-01 14:45, 2010-01-01 15:45)');",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_EXCLUSION_CONSTRAINT_VIOLATION,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO reservation VALUES ('[2010-01-01 14:45, 2010-01-01 15:45)');",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "syntax error at or near \"WHERE\" at character 26",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM abc LIMIT 2 WHERE id=1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_SYNTAX_ERROR,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM abc LIMIT 2 WHERE id=1",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "syntax error at end of input",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM (SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_SYNTAX_ERROR,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM (SELECT 1",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid input syntax for integer: \"\" at character 40",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM table WHERE int_column = ''",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_INVALID_INPUT_SYNTAX,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM table WHERE int_column = ''",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "value too long for type character varying(3)",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "INSERT INTO x(y) VALUES ('zzzzz')",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_VALUE_TOO_LONG_FOR_TYPE,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO x(y) VALUES ('zzzzz')",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid value \"string\" for \"YYYY\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Value must be an integer.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "SELECT to_timestamp($1, 'YYYY-mm-DD')",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_INVALID_VALUE,
			UUID:           uuid.UUID{1},
			Query:          "SELECT to_timestamp($1, 'YYYY-mm-DD')",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "malformed array literal: \"a, b\" at character 33",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Array value must start with \"{\" or dimension information.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "SELECT * FROM x WHERE id = ANY ('a, b')",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_MALFORMED_ARRAY_LITERAL,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM x WHERE id = ANY ('a, b')",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "subquery in FROM must have an alias at character 15",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "For example, FROM (SELECT ...) [AS] foo.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "SELECT * FROM (SELECT 1)",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_SUBQUERY_MISSING_ALIAS,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM (SELECT 1)",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "INSERT has more expressions than target columns at character 341",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "INSERT INTO x(y) VALUES (1, 2)",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_INSERT_TARGET_COLUMN_MISMATCH,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO x(y) VALUES (1, 2)",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "op ANY/ALL (array) requires array on right side at character 33",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM x WHERE id = ANY ($1)",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_ANY_ALL_REQUIRES_ARRAY,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM x WHERE id = ANY ($1)",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "column \"abc.def\" must appear in the GROUP BY clause or be used in an aggregate function at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT def, MAX(def) FROM abc",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_COLUMN_MISSING_FROM_GROUP_BY,
			UUID:           uuid.UUID{1},
			Query:          "SELECT def, MAX(def) FROM abc",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "relation \"x\" does not exist at character 15",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM x",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_RELATION_DOES_NOT_EXIST,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM x",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "column \"y\" does not exist at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT y FROM x",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_COLUMN_DOES_NOT_EXIST,
			UUID:           uuid.UUID{1},
			Query:          "SELECT y FROM x",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "column \"y\" of relation \"x\" does not exist",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "ALTER TABLE x DROP COLUMN y;",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_COLUMN_DOES_NOT_EXIST,
			UUID:           uuid.UUID{1},
			Query:          "ALTER TABLE x DROP COLUMN y;",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "column reference \"z\" is ambiguous at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT z FROM x, y",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_COLUMN_REFERENCE_AMBIGUOUS,
			UUID:           uuid.UUID{1},
			Query:          "SELECT z FROM x, y",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "operator does not exist: boolean || boolean at character 13",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "No operator matches the given name and argument type(s). You might need to add explicit type casts.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "SELECT true || true",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_OPERATOR_DOES_NOT_EXIST,
			UUID:           uuid.UUID{1},
			Query:          "SELECT true || true",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "function x(integer) does not exist at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "No function matches the given name and argument types. You might need to add explicit type casts.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "SELECT x(1);",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_FUNCTION_DOES_NOT_EXIST,
			UUID:           uuid.UUID{1},
			Query:          "SELECT x(1);",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "permission denied for schema my_schema at character 25",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM my_schema.table",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_PERMISSION_DENIED,
			UUID:           uuid.UUID{1},
			Query:          "SELECT * FROM my_schema.table",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "current transaction is aborted, commands ignored until end of transaction block",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_TRANSACTION_IS_ABORTED,
			UUID:           uuid.UUID{1},
			Query:          "SELECT 1",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "there is no unique or exclusion constraint matching the ON CONFLICT specification",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "INSERT INTO x (y, z) VALUES ('a', 1) ON CONFLICT (y) DO UPDATE SET z = EXCLUDED.z",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_ON_CONFLICT_NO_CONSTRAINT_MATCH,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO x (y, z) VALUES ('a', 1) ON CONFLICT (y) DO UPDATE SET z = EXCLUDED.z",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "ON CONFLICT DO UPDATE command cannot affect row a second time",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "INSERT INTO x (y, z) VALUES ('a', 1), ('a', 2) ON CONFLICT (y) DO UPDATE SET z = EXCLUDED.z",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}, {
			Content:  "Ensure that no rows proposed for insertion within the same command have duplicate constrained values.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_ON_CONFLICT_ROW_AFFECTED_TWICE,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO x (y, z) VALUES ('a', 1), ('a', 2) ON CONFLICT (y) DO UPDATE SET z = EXCLUDED.z",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "column \"abc\" cannot be cast to type \"date\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT abc::date FROM x",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_COLUMN_CANNOT_BE_CAST,
			UUID:           uuid.UUID{1},
			Query:          "SELECT abc::date FROM x",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "division by zero",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT 1/0",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_DIVISION_BY_ZERO,
			UUID:           uuid.UUID{1},
			Query:          "SELECT 1/0",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "cannot drop table x because other objects depend on it",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "view a depends on table x",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "Use DROP ... CASCADE to drop the dependent objects too.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "DROP TABLE x",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_CANNOT_DROP,
			UUID:           uuid.UUID{1},
			Query:          "DROP TABLE x",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_HINT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "integer out of range",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "INSERT INTO x(y) VALUES (10000000000000)",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_INTEGER_OUT_OF_RANGE,
			UUID:           uuid.UUID{1},
			Query:          "INSERT INTO x(y) VALUES (10000000000000)",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid regular expression: quantifier operand invalid",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT regexp_replace('test', '<(?i:test)', '');",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_INVALID_REGEXP,
			UUID:           uuid.UUID{1},
			Query:          "SELECT regexp_replace('test', '<(?i:test)', '');",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "there is no parameter $1 at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT $1;",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_PARAM_MISSING,
			UUID:           uuid.UUID{1},
			Query:          "SELECT $1;",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "no such savepoint",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "ROLLBACK TO x",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_NO_SUCH_SAVEPOINT,
			UUID:           uuid.UUID{1},
			Query:          "ROLLBACK TO x",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unterminated quoted string at or near \"some string",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT 1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_UNTERMINATED_QUOTED_STRING,
			UUID:           uuid.UUID{1},
			Query:          "SELECT 1",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unterminated quoted identifier at or near \"\"1\" at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT \"1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_UNTERMINATED_QUOTED_IDENTIFIER,
			UUID:           uuid.UUID{1},
			Query:          "SELECT \"1",
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid byte sequence for encoding \"UTF8\": 0xd0 0x2e",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_INVALID_BYTE_SEQUENCE,
		}},
		nil,
	},
	// auto_explain
	{
		[]state.LogLine{{
			Content: "duration: 2334.085 ms  plan:\n" +
				"	{\n" +
				"	  \"Query Text\": \"SELECT abalance FROM pgbench_accounts WHERE aid = 2262632;\",\n" +
				"	  \"Plan\": {\n" +
				"	    \"Node Type\": \"Index Scan\",\n" +
				"	    \"Parallel Aware\": false,\n" +
				"	    \"Scan Direction\": \"Forward\",\n" +
				"	    \"Index Name\": \"pgbench_accounts_pkey\",\n" +
				"	    \"Relation Name\": \"pgbench_accounts\",\n" +
				"	    \"Schema\": \"public\",\n" +
				"	    \"Alias\": \"pgbench_accounts\",\n" +
				"	    \"Startup Cost\": 0.43,\n" +
				"	    \"Total Cost\": 8.45,\n" +
				"	    \"Plan Rows\": 1,\n" +
				"	    \"Plan Width\": 4,\n" +
				"	    \"Actual Rows\": 1,\n" +
				"	    \"Actual Loops\": 1,\n" +
				"	    \"Output\": [\"abalance\"],\n" +
				"	    \"Index Cond\": \"(pgbench_accounts.aid = 2262632)\",\n" +
				"	    \"Rows Removed by Index Recheck\": 0,\n" +
				"	    \"Shared Hit Blocks\": 4,\n" +
				"	    \"Shared Read Blocks\": 0,\n" +
				"	    \"Shared Dirtied Blocks\": 0,\n" +
				"	    \"Shared Written Blocks\": 0,\n" +
				"	    \"Local Hit Blocks\": 0,\n" +
				"	    \"Local Read Blocks\": 0,\n" +
				"	    \"Local Dirtied Blocks\": 0,\n" +
				"	    \"Local Written Blocks\": 0,\n" +
				"	    \"Temp Read Blocks\": 0,\n" +
				"	    \"Temp Written Blocks\": 0,\n" +
				"	    \"I/O Read Time\": 0.000,\n" +
				"	    \"I/O Write Time\": 0.000\n" +
				"	  },\n" +
				"	  \"Triggers\": [\n" +
				"	  ]\n" +
				"	}\n",
		}},
		[]state.LogLine{{
			Query:          "SELECT abalance FROM pgbench_accounts WHERE aid = 2262632;",
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			Details:        map[string]interface{}{"duration_ms": 2334.085},
		}},
		[]state.PostgresQuerySample{{
			Query:         "SELECT abalance FROM pgbench_accounts WHERE aid = 2262632;",
			RuntimeMs:     2334.085,
			HasExplain:    true,
			ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
			ExplainFormat: pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT,
			ExplainOutput: "[{\"Plan\":{\"Actual Loops\":1,\"Actual Rows\":1,\"Alias\":\"pgbench_accounts\",\"I/O Read Time\":0,\"I/O Write Time\":0,\"Index Cond\":\"(pgbench_accounts.aid = 2262632)\",\"Index Name\":\"pgbench_accounts_pkey\",\"Local Dirtied Blocks\":0,\"Local Hit Blocks\":0,\"Local Read Blocks\":0,\"Local Written Blocks\":0,\"Node Type\":\"Index Scan\",\"Output\":[\"abalance\"],\"Parallel Aware\":false,\"Plan Rows\":1,\"Plan Width\":4,\"Relation Name\":\"pgbench_accounts\",\"Rows Removed by Index Recheck\":0,\"Scan Direction\":\"Forward\",\"Schema\":\"public\",\"Shared Dirtied Blocks\":0,\"Shared Hit Blocks\":4,\"Shared Read Blocks\":0,\"Shared Written Blocks\":0,\"Startup Cost\":0.43,\"Temp Read Blocks\":0,\"Temp Written Blocks\":0,\"Total Cost\":8.45}}]",
		}},
	},
	{
		[]state.LogLine{{
			Content: "duration: 2334.085 ms  plan:\n" +
				"	{\n" +
				"	  \"Query Text\": \"SELECT abalance FROM pgbench_accounts WHERE aid = [Your log message was truncated]",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			Details:        map[string]interface{}{"duration_ms": 2334.085, "truncated": true},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "duration: 1681.452 ms  plan:\n" +
				"  Query Text: UPDATE pgbench_branches SET bbalance = bbalance + 2656 WHERE bid = 59;\n" +
				"  Update on public.pgbench_branches  (cost=0.27..8.29 rows=1 width=370) (actual rows=0 loops=1)\n" +
				"    Buffers: shared hit=7\n" +
				"    ->  Index Scan using pgbench_branches_pkey on public.pgbench_branches  (cost=0.27..8.29 rows=1 width=370) (actual rows=1 loops=1)\n" +
				"          Output: bid, (bbalance + 2656), filler, ctid\n" +
				"          Index Cond: (pgbench_branches.bid = 59)",
		}},
		[]state.LogLine{{
			Query:          "UPDATE pgbench_branches SET bbalance = bbalance + 2656 WHERE bid = 59;",
			Classification: pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			Details:        map[string]interface{}{"duration_ms": 1681.452},
		}},
		[]state.PostgresQuerySample{{
			Query:         "UPDATE pgbench_branches SET bbalance = bbalance + 2656 WHERE bid = 59;",
			RuntimeMs:     1681.452,
			HasExplain:    true,
			ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
			ExplainFormat: pganalyze_collector.QuerySample_TEXT_EXPLAIN_FORMAT,
			ExplainOutput: "Update on public.pgbench_branches  (cost=0.27..8.29 rows=1 width=370) (actual rows=0 loops=1)\n" +
				"    Buffers: shared hit=7\n" +
				"    ->  Index Scan using pgbench_branches_pkey on public.pgbench_branches  (cost=0.27..8.29 rows=1 width=370) (actual rows=1 loops=1)\n" +
				"          Output: bid, (bbalance + 2656), filler, ctid\n" +
				"          Index Cond: (pgbench_branches.bid = 59)",
		}},
	},
	// pganalyze-collector-identify
	{
		[]state.LogLine{{
			Content:  "pganalyze-collector-identify: server1",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "PL/pgSQL function inline_code_block line 2 at RAISE",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "/* pganalyze-collector */ DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: server1';\nEND$$;",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Classification: pganalyze_collector.LogLineInformation_PGA_COLLECTOR_IDENTIFY,
			UUID:           uuid.UUID{1},
			Query:          "/* pganalyze-collector */ DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: server1';\nEND$$;",
			Details:        map[string]interface{}{"config_section": "server1"},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID: uuid.UUID{1},
		}, {
			LogLevel:   pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID: uuid.UUID{1},
		}},
		nil,
	},
}

func TestAnalyzeLogLines(t *testing.T) {
	for _, pair := range tests {
		l, s := logs.AnalyzeLogLines(pair.logLinesIn)

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		if diff := cfg.Compare(pair.logLinesOut, l); diff != "" {
			t.Errorf("For %v: log lines diff: (-got +want)\n%s", pair.logLinesIn, diff)
		}
		if diff := cfg.Compare(pair.samplesOut, s); diff != "" {
			t.Errorf("For %v: query samples diff: (-got +want)\n%s", pair.samplesOut, diff)
		}
	}
}
