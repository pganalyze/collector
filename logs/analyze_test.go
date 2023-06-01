package logs_test

import (
	"testing"

	"github.com/guregu/null"
	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/logs"
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
			Content:  "duration: 3205.800 ms  execute a2: SELECT \"servers\".* FROM \"servers\" WHERE \"servers\".\"id\" = 1 LIMIT 2\n",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Query:              "SELECT \"servers\".* FROM \"servers\" WHERE \"servers\".\"id\" = 1 LIMIT 2",
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 35,
				ByteEnd:   101,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		[]state.PostgresQuerySample{{
			Query:     "SELECT \"servers\".* FROM \"servers\" WHERE \"servers\".\"id\" = 1 LIMIT 2",
			RuntimeMs: 3205.8,
		}},
	},
	{
		[]state.LogLine{{
			Content:  "duration: 4079.697 ms  execute <unnamed>: \nSELECT * FROM x WHERE y = $1 LIMIT $2",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "parameters: $1 = 'long string', $2 = '1'",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			Query:              "SELECT * FROM x WHERE y = $1 LIMIT $2",
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 43,
				ByteEnd:   80,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 18,
				ByteEnd:   29,
				Kind:      state.StatementParameterLogSecret,
			}, {
				ByteStart: 38,
				ByteEnd:   39,
				Kind:      state.StatementParameterLogSecret,
			}},
		}},
		[]state.PostgresQuerySample{{
			Query:     "SELECT * FROM x WHERE y = $1 LIMIT $2",
			RuntimeMs: 4079.697,
			Parameters: []null.String{
				null.StringFrom("long string"),
				null.StringFrom("1"),
			},
		}},
	},
	{
		[]state.LogLine{{
			Content:  "duration: 4079.697 ms  execute <unnamed>: \nSELECT * FROM x WHERE y = $1 AND z = $2 LIMIT $3",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "parameters: $1 = 'long string', $2 = NULL, $3 = '10'",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			Query:              "SELECT * FROM x WHERE y = $1 AND z = $2 LIMIT $3",
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 43,
				ByteEnd:   91,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 18,
				ByteEnd:   29,
				Kind:      state.StatementParameterLogSecret,
			}, {
				ByteStart: 37,
				ByteEnd:   41,
				Kind:      state.StatementParameterLogSecret,
			}, {
				ByteStart: 49,
				ByteEnd:   51,
				Kind:      state.StatementParameterLogSecret,
			}},
		}},
		[]state.PostgresQuerySample{{
			Query:     "SELECT * FROM x WHERE y = $1 AND z = $2 LIMIT $3",
			RuntimeMs: 4079.697,
			Parameters: []null.String{
				null.StringFrom("long string"),
				null.NewString("", false),
				null.StringFrom("10"),
			},
		}},
	},
	{
		[]state.LogLine{{
			Content:  "duration: 3205.800 ms  execute a2: SELECT ...[Your log message was truncated]",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			Details:            map[string]interface{}{"truncated": true},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 35,
				ByteEnd:   77,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "duration: 123.500 ms",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_DURATION,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_LOG,
			UUID:               uuid.UUID{1},
			Query:              "SELECT $1, $2",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 19,
				ByteEnd:   32,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 18,
				ByteEnd:   19,
				Kind:      state.StatementParameterLogSecret,
			}, {
				ByteStart: 28,
				ByteEnd:   29,
				Kind:      state.StatementParameterLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_LOG,
			UUID:               uuid.UUID{1},
			Query:              "EXECUTE x(1);",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 11,
				ByteEnd:   24,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 9,
				ByteEnd:   32,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	// Connects/Disconnects
	{
		[]state.LogLine{{
			Content: "connection received: host=172.30.0.165 port=56902",
		}, {
			Content: "connection received: host=ec2-102-13-140-150.compute-1.amazonaws.com port=12345",
		}, {
			Content: "connection received: host=[local]",
		}, {
			Content: "connection authorized: user=myuser database=mydb SSL enabled (protocol=TLSv1.2, cipher=ECDHE-RSA-AES256-GCM-SHA384, compression=off)",
		}, {
			Content: "connection authorized: user=myuser database=myuser application_name=puma: cluster worker 2: 44125 [myapp]",
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
			Content:  "password authentication failed for user \"special\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
			UUID:     uuid.UUID{2},
		}, {
			Content:  "Role \"special\" does not exist. Connection matched pg_hba.conf line 67: \"hostnossl all all all md5\"",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "database \"template0\" is not currently accepting connections",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content:  "role \"abc\" is not permitted to log in",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content:  "could not connect to Ident server at address \"127.0.0.1\", port 113: Connection refused",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}, {
			Content:  "Ident authentication failed for user \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}, {
			Content: "disconnection: session time: 1:53:01.198 user=myuser database=mydb host=172.30.0.165 port=56902",
		}, {
			Content: "disconnection: session time: 0:00:00.199 user=user database=db host=ec2-102-13-140-150.compute-1.amazonaws.com port=12345",
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_RECEIVED,
			Details: map[string]interface{}{
				"host": "172.30.0.165",
			},
			ReviewedForSecrets: true,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_RECEIVED,
			Details: map[string]interface{}{
				"host": "ec2-102-13-140-150.compute-1.amazonaws.com",
			},
			ReviewedForSecrets: true,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_RECEIVED,
			Details: map[string]interface{}{
				"host": "[local]",
			},
			ReviewedForSecrets: true,
		}, {
			Classification: pganalyze_collector.LogLineInformation_CONNECTION_AUTHORIZED,
			Details: map[string]interface{}{
				"ssl_protocol": "TLSv1.2",
			},
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_AUTHORIZED,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 40,
				ByteEnd:   107,
				Kind:      state.OpsLogSecret,
			}},
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			UUID:               uuid.UUID{2},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{2},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 72,
				ByteEnd:   97,
				Kind:      state.OpsLogSecret,
			}},
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_REJECTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_DISCONNECTED,
			Details:            map[string]interface{}{"session_time_secs": 6781.198},
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_DISCONNECTED,
			Details:            map[string]interface{}{"session_time_secs": 0.199},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "incomplete startup packet",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_CLIENT_FAILED_TO_CONNECT,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unexpected EOF on client connection with an open transaction",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_LOST_OPEN_TX,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "could not receive data from client: Connection reset by peer",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_LOST,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "could not send data to client: Broken pipe",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_LOST,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "terminating connection because protocol synchronization was lost",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_LOST,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "connection to client lost",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_LOST,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "unexpected EOF on client connection",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_LOST,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "terminating connection due to administrator command",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_CONNECTION_TERMINATED,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "remaining connection slots are reserved for non-replication superuser connections",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_OUT_OF_CONNECTIONS,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "sorry, too many clients already",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_OUT_OF_CONNECTIONS,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "too many connections for role \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_ROLE,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "too many connections for database \"postgres\"",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_TOO_MANY_CONNECTIONS_DATABASE,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_COULD_NOT_ACCEPT_SSL_CONNECTION,
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_COULD_NOT_ACCEPT_SSL_CONNECTION,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unsupported frontend protocol 65363.12345: server supports 1.0 to 3.0",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			Classification:     pganalyze_collector.LogLineInformation_PROTOCOL_ERROR_UNSUPPORTED_VERSION,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_PROTOCOL_ERROR_INCOMPLETE_MESSAGE,
			Query:              "COPY \"abc\" (\"x\") FROM STDIN BINARY",
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 34,
				Kind:    state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	// Checkpoints
	{
		[]state.LogLine{{
			Content: "checkpoint starting: xlog",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_CHECKPOINT_STARTING,
			Details:            map[string]interface{}{"reason": "xlog"},
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
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
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "restartpoint starting: shutdown immediate",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_RESTARTPOINT_STARTING,
			Details:            map[string]interface{}{"reason": "shutdown immediate"},
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_RESTARTPOINT_AT,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	// WAL/Archiving
	{
		[]state.LogLine{{
			Content: "invalid record length at 4E8/9E0979A8",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_WAL_INVALID_RECORD_LENGTH,
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_WAL_REDO,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_WAL_REDO,
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_WAL_REDO,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_WAL_INVALID_RECORD_LENGTH,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_WAL_REDO,
			ReviewedForSecrets: true,
		}, {
			Classification: pganalyze_collector.LogLineInformation_WAL_REDO,
			Details: map[string]interface{}{
				"last_transaction": "2018-03-12 02:09:12.585354+00",
			},
			ReviewedForSecrets: true,
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
				"exit_code": 1,
			},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 32,
				ByteEnd:   105,
				Kind:      state.OpsLogSecret,
			}},
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
				"signal": 6,
			},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 32,
				ByteEnd:   143,
				Kind:      state.OpsLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_WAL_ARCHIVE_COMMAND_FAILED,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "pg_stop_backup complete, all required WAL segments have been archived",
			LogLevel: pganalyze_collector.LogLineInformation_NOTICE,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_WAL_BASE_BACKUP_COMPLETE,
			LogLevel:           pganalyze_collector.LogLineInformation_NOTICE,
			ReviewedForSecrets: true,
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
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 32,
				Kind:    state.StatementTextLogSecret,
			}},
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
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
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
			RelatedPids:        []int32{583, 2078, 456},
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_QUERY,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 28,
				Kind:    state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 24,
				Kind:    state.StatementTextLogSecret,
			}},
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
			Classification:     pganalyze_collector.LogLineInformation_LOCK_TIMEOUT,
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Query:              "UPDATE mytable SET y = 2 WHERE x = 1",
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 36,
				Kind:    state.StatementTextLogSecret,
			}},
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
			RelatedPids:        []int32{583, 123, 2078},
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_LOCK_DEADLOCK_DETECTED,
			Query:              "INSERT INTO x (id, name, email) VALUES (1, 'ABC', 'abc@example.com') ON CONFLICT(email) DO UPDATE SET name = excluded.name RETURNING id",
			UUID:               uuid.UUID{1},
			RelatedPids:        []int32{9788, 91, 98, 91},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 165,
				ByteEnd:   304,
				Kind:      state.StatementTextLogSecret,
			}, {
				ByteStart: 317,
				ByteEnd:   456,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 135,
				Kind:    state.StatementTextLogSecret,
			}},
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
			ReviewedForSecrets: true,
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
			RelatedPids:        []int32{660, 663},
			Query:              "SELECT pg_advisory_lock(1, 2);",
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 30,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_AUTOVACUUM_CANCEL,
			UUID:               uuid.UUID{1},
			Database:           "dbname",
			SchemaName:         "schemaname",
			RelationName:       "tablename",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
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
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
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
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "autovacuum launcher started",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_STARTED,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "autovacuum launcher shutting down",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_SHUTTING_DOWN,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "terminating autovacuum process due to administrator command",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_AUTOVACUUM_LAUNCHER_SHUTTING_DOWN,
			ReviewedForSecrets: true,
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
			Database:       "mydb",
			SchemaName:     "public",
			RelationName:   "vac_test",
			Details: map[string]interface{}{
				"aggressive":          false,
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
			ReviewedForSecrets: true,
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
			Database:       "demo_pgbench",
			SchemaName:     "public",
			RelationName:   "pgbench_tellers",
			Details: map[string]interface{}{
				"aggressive":          false,
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
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"mydb.myschema.mytable\": index scans: 0\n" +
				"	pages: 0 removed, 14761 remain, 0 skipped due to pins, 12461 skipped frozen\n" +
				"	tuples: 0 removed, 122038 remain, 14433 are dead but not yet removable, oldest xmin: 538040633\n" +
				"	index scan bypassed: 255 pages from table (1.73% of total) have 661 dead item identifiers\n" +
				"	I/O timings: read: 0.000 ms, write: 0.000 ms\n" +
				"	avg read rate: 0.000 MB/s, avg write rate: 0.000 MB/s\n" +
				"	buffer usage: 4420 hits, 0 misses, 0 dirtied\n" +
				"	WAL usage: 1 records, 0 full page images, 245 bytes\n" +
				"	system usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.01 s",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Database:       "mydb",
			SchemaName:     "myschema",
			RelationName:   "mytable",
			Details: map[string]interface{}{
				"aggressive":               false,
				"anti_wraparound":          false,
				"num_index_scans":          0,
				"pages_removed":            0,
				"rel_pages":                14761,
				"pinskipped_pages":         0,
				"frozenskipped_pages":      12461,
				"tuples_deleted":           0,
				"new_rel_tuples":           122038,
				"new_dead_tuples":          14433,
				"oldest_xmin":              538040633,
				"lpdead_index_scan":        "bypassed",
				"lpdead_item_pages":        255,
				"lpdead_item_page_percent": 1.73,
				"lpdead_items":             661,
				"blk_read_time":            0,
				"blk_write_time":           0,
				"read_rate_mb":             0,
				"write_rate_mb":            0,
				"vacuum_page_hit":          4420,
				"vacuum_page_miss":         0,
				"vacuum_page_dirty":        0,
				"wal_records":              1,
				"wal_fpi":                  0,
				"wal_bytes":                245,
				"rusage_kernel":            0.00,
				"rusage_user":              0.00,
				"elapsed_secs":             0.01,
			},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic aggressive vacuum to prevent wraparound of table \"mydb.myschema.mytable\": index scans: 0\n" +
				"	pages: 0 removed, 241245 remain, 0 skipped due to pins, 241244 skipped frozen\n" +
				"	tuples: 0 removed, 17418745 remain, 0 are dead but not yet removable, oldest xmin: 538040633\n" +
				"	index scan not needed: 3 pages from table (0.01% of total) had 0 dead item identifiers removed\n" +
				"	I/O timings: read: 10.540 ms, write: 0.000 ms\n" +
				"	avg read rate: 38.748 MB/s, avg write rate: 0.538 MB/s\n" +
				"	buffer usage: 50 hits, 72 misses, 1 dirtied\n" +
				"	WAL usage: 1 records, 1 full page images, 2147 bytes\n" +
				"	system usage: CPU: user: 1.23 s, system: 4.56 s, elapsed: 0.01 s",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Database:       "mydb",
			SchemaName:     "myschema",
			RelationName:   "mytable",
			Details: map[string]interface{}{
				"aggressive":               true,
				"anti_wraparound":          true,
				"num_index_scans":          0,
				"pages_removed":            0,
				"rel_pages":                241245,
				"pinskipped_pages":         0,
				"frozenskipped_pages":      241244,
				"tuples_deleted":           0,
				"new_rel_tuples":           17418745,
				"new_dead_tuples":          0,
				"oldest_xmin":              538040633,
				"lpdead_index_scan":        "not needed",
				"lpdead_item_pages":        3,
				"lpdead_item_page_percent": 0.01,
				"lpdead_items":             0,
				"blk_read_time":            10.54,
				"blk_write_time":           0,
				"read_rate_mb":             38.748,
				"write_rate_mb":            0.538,
				"vacuum_page_hit":          50,
				"vacuum_page_miss":         72,
				"vacuum_page_dirty":        1,
				"wal_records":              1,
				"wal_fpi":                  1,
				"wal_bytes":                2147,
				"rusage_user":              1.23,
				"rusage_kernel":            4.56,
				"elapsed_secs":             0.01,
			},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic vacuum of table \"alloydbadmin.public.heartbeat\": index scans: 0, elapsed time: 0 s, index vacuum time: 0 ms," +
				" pages: 0 removed, 1 remain, 0 skipped due to pins, 0 skipped frozen 0 skipped using mintxid," +
				" tuples: 60 removed, 1 remain, 0 are dead but not yet removable, oldest xmin: 1782," +
				" index scan not needed: 0 pages from table (0.00% of total) had 0 dead item identifiers removed," +
				" I/O timings: read: 0.000 ms, write: 0.000 ms," +
				" avg read rate: 0.000 MB/s, avg write rate: 0.000 MB/s," +
				" buffer usage: 42 hits, 0 misses, 0 dirtied," +
				" WAL usage: 3 records, 0 full page images, 286 bytes," +
				" system usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.01 s",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOVACUUM_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Database:       "alloydbadmin",
			SchemaName:     "public",
			RelationName:   "heartbeat",
			Details: map[string]interface{}{
				"aggressive":               false,
				"anti_wraparound":          false,
				"num_index_scans":          0,
				"pages_removed":            0,
				"rel_pages":                1,
				"pinskipped_pages":         0,
				"frozenskipped_pages":      0,
				"tuples_deleted":           60,
				"new_rel_tuples":           1,
				"new_dead_tuples":          0,
				"oldest_xmin":              1782,
				"lpdead_index_scan":        "not needed",
				"lpdead_item_pages":        0,
				"lpdead_item_page_percent": 0,
				"lpdead_items":             0,
				"blk_read_time":            0,
				"blk_write_time":           0,
				"read_rate_mb":             0,
				"write_rate_mb":            0,
				"vacuum_page_hit":          42,
				"vacuum_page_miss":         0,
				"vacuum_page_dirty":        0,
				"wal_records":              3,
				"wal_fpi":                  0,
				"wal_bytes":                286,
				"rusage_user":              0.00,
				"rusage_kernel":            0.00,
				"elapsed_secs":             0.01,
			},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic aggressive vacuum of table \"demo_pgbench.public.pgbench_tellers\": index scans: 0" +
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
			Database:       "demo_pgbench",
			SchemaName:     "public",
			RelationName:   "pgbench_tellers",
			Details: map[string]interface{}{
				"aggressive":          true,
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
			ReviewedForSecrets: true,
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
			Database:       "postgres",
			SchemaName:     "public",
			RelationName:   "pgbench_branches",
			Details: map[string]interface{}{
				"rusage_kernel": 1.02,
				"rusage_user":   2.08,
				"elapsed_secs":  108.25,
			},
			ReviewedForSecrets: true,
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
			Database:       "demo_pgbench",
			SchemaName:     "public",
			RelationName:   "pgbench_history",
			Details: map[string]interface{}{
				"rusage_kernel": 0.01,
				"rusage_user":   0.23,
				"elapsed_secs":  0.89,
			},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "automatic analyze of table \"mydb.myschema.mytable\"\n" +
				"	I/O timings: read: 1.027 ms, write: 0.000 ms\n" +
				"	avg read rate: 1.339 MB/s, avg write rate: 8.705 MB/s\n" +
				"	buffer usage: 1369 hits, 6 misses, 39 dirtied\n" +
				"	system usage: CPU: user: 0.02 s, system: 0.00 s, elapsed: 0.03 s",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification: pganalyze_collector.LogLineInformation_AUTOANALYZE_COMPLETED,
			LogLevel:       pganalyze_collector.LogLineInformation_LOG,
			Database:       "mydb",
			SchemaName:     "myschema",
			RelationName:   "mytable",
			Details: map[string]interface{}{
				"blk_read_time":      1.027,
				"blk_write_time":     0.000,
				"read_rate_mb":       1.339,
				"write_rate_mb":      8.705,
				"analyze_page_hit":   1369,
				"analyze_page_miss":  6,
				"analyze_page_dirty": 39,
				"rusage_user":        0.02,
				"rusage_kernel":      0.00,
				"elapsed_secs":       0.03,
			},
			ReviewedForSecrets: true,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "skipping vacuum of \"mytable\" --- lock not available",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_SKIPPING_VACUUM_LOCK_NOT_AVAILABLE,
			UUID:               uuid.UUID{1},
			RelationName:       "mytable",
			ReviewedForSecrets: true,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "skipping analyze of \"pgbench_tellers\" --- lock not available",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			UUID:     uuid.UUID{1},
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_SKIPPING_ANALYZE_LOCK_NOT_AVAILABLE,
			UUID:               uuid.UUID{1},
			RelationName:       "pgbench_tellers",
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_USER,
			Query:              "SELECT 1",
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 8,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_CANCELED_TIMEOUT,
			Query:              "SELECT 1",
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 8,
				Kind:    state.StatementTextLogSecret,
			}},
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
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 28,
				ByteEnd:   58,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:           pganalyze_collector.LogLineInformation_WARNING,
			UUID:               uuid.UUID{2},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{2},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{2},
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_CRASHED,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_START_RECOVERING,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			UUID:               uuid.UUID{2},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{2},
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_SHUTDOWN,
			ReviewedForSecrets: true,
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
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 50,
				Kind:    state.StatementTextLogSecret,
			}},
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
			Classification:     pganalyze_collector.LogLineInformation_SERVER_MISC,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 29,
				ByteEnd:   66,
				Kind:      state.OpsLogSecret,
			}, {
				ByteStart: 69,
				ByteEnd:   94,
				Kind:      state.OpsLogSecret,
			}},
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_MISC,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 21,
				ByteEnd:   43,
				Kind:      state.OpsLogSecret,
			}, {
				ByteStart: 49,
				ByteEnd:   81,
				Kind:      state.OpsLogSecret,
			}, {
				ByteStart: 84,
				ByteEnd:   95,
				Kind:      state.OpsLogSecret,
			}},
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_MISC,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_SERVER_OUT_OF_MEMORY,
			UUID:               uuid.UUID{1},
			Query:              "SELECT 123",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 10,
				Kind:    state.StatementTextLogSecret,
			}},
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
			RelatedPids:        []int32{123},
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_WARNING,
			Classification:     pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM,
			ReviewedForSecrets: true,
		}, {
			LogLevel:       pganalyze_collector.LogLineInformation_ERROR,
			Classification: pganalyze_collector.LogLineInformation_SERVER_INVALID_CHECKSUM,
			UUID:           uuid.UUID{1},
			Query:          "SELECT 1",
			Details: map[string]interface{}{
				"block": 335458,
				"file":  "base/16385/99454",
			},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 8,
				Kind:    state.StatementTextLogSecret,
			}},
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
			Classification:     pganalyze_collector.LogLineInformation_SERVER_RELOAD,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_RELOAD,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_RELOAD,
			ReviewedForSecrets: true,
		}, {
			Classification:     pganalyze_collector.LogLineInformation_SERVER_RELOAD,
			ReviewedForSecrets: true,
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
			RelatedPids:        []int32{31458, 30491},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 28,
				ByteEnd:   36,
				Kind:      state.StatementTextLogSecret,
			}},
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
			RelatedPids:        []int32{17443},
			ReviewedForSecrets: true,
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
			RelatedPids:        []int32{17443},
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "using stale statistics instead of current ones because stats collector is not responding",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_SERVER_STATS_COLLECTOR_TIMEOUT,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "pgstat wait timeout",
			LogLevel: pganalyze_collector.LogLineInformation_WARNING,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_SERVER_STATS_COLLECTOR_TIMEOUT,
			LogLevel:           pganalyze_collector.LogLineInformation_WARNING,
			ReviewedForSecrets: true,
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
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_RESTORED_WAL_FROM_ARCHIVE,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "started streaming WAL from primary at 4E8/9E000000 on timeline 6",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_STARTED_STREAMING,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "restarted WAL streaming at 3E/62000000 on timeline 3",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_STARTED_STREAMING,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "could not receive data from WAL stream: SSL error: sslv3 alert unexpected message",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_STREAMING_INTERRUPTED,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 40,
				ByteEnd:   81,
				Kind:      state.OpsLogSecret,
			}},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "terminating walreceiver process due to administrator command",
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_STOPPED_STREAMING,
			LogLevel:           pganalyze_collector.LogLineInformation_FATAL,
			ReviewedForSecrets: true,
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "consistent recovery state reached at 4E8/9E0979A8",
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_CONSISTENT_RECOVERY_STATE,
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_STATEMENT_CANCELED,
			Query:              "SELECT 1",
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 8,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "according to history file, WAL location 2D5/22000000 belongs to timeline 3, but previous recovered WAL file came from timeline 4",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_STANDBY_INVALID_TIMELINE,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_UNIQUE_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO a (b, c) VALUES ($1,$2) RETURNING id",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 12,
				ByteEnd:   31,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 0,
				ByteEnd:   48,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		nil,
	}, {
		[]state.LogLine{{
			Content:  "duplicate key value violates unique constraint \"query_stats_pkey\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "COPY query_stats(\"database_id\",\"fingerprint\",\"collected_at\",\"collected_interval_secs\") FROM STDIN BINARY",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}, {
			Content:  "COPY query_stats, line 1",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "Key (database_id, fingerprint, postgres_role_id, collected_at)=(123, \\x025352e69ba951615a192c04aa3b217cad54b390f9, 2019-01-13 19:36:00) already exists.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_UNIQUE_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			Query:              "COPY query_stats(\"database_id\",\"fingerprint\",\"collected_at\",\"collected_interval_secs\") FROM STDIN BINARY",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 0,
				ByteEnd:   104,
				Kind:      state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 64,
				ByteEnd:   134,
				Kind:      state.TableDataLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO weather VALUES ('Berkeley', 45, 53, 0.0, '1994-11-28');",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 12,
				ByteEnd:   20,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 0,
				ByteEnd:   67,
				Kind:      state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_FOREIGN_KEY_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			Query:              "DELETE FROM test WHERE id = 123",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 10,
				ByteEnd:   13,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 0,
				ByteEnd:   31,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "null value in column \"mycolumn\" violates not-null constraint",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Failing row contains (null, secret).",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "INSERT INTO \"test\" (\"mycolumn\", \"mysecret\") VALUES ($1) RETURNING \"id\"",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_NOT_NULL_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO \"test\" (\"mycolumn\", \"mysecret\") VALUES ($1) RETURNING \"id\"",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 22,
				ByteEnd:   34,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 0,
				ByteEnd:   70,
				Kind:      state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 22,
				ByteEnd:   26,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_CHECK_CONSTRAINT_VIOLATION,
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_EXCLUSION_CONSTRAINT_VIOLATION,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO reservation VALUES ('[2010-01-01 14:45, 2010-01-01 15:45)');",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 14,
				ByteEnd:   59,
				Kind:      state.TableDataLogSecret,
			}, {
				ByteStart: 99,
				ByteEnd:   144,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 72,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_SYNTAX_ERROR,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM abc LIMIT 2 WHERE id=1",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 25,
				ByteEnd:   30,
				Kind:      state.ParsingErrorLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 36,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_SYNTAX_ERROR,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM (SELECT 1",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 23,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid input syntax for integer: \"A\" at character 40",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT * FROM table WHERE int_column = 'A'",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INVALID_INPUT_SYNTAX,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM table WHERE int_column = 'A'",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 35,
				ByteEnd:   36,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 42,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid input syntax for type json",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Escape sequence \"\\1\" is invalid.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "JSON data, line 1: ...',n ''foobar: \\1...",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "SELECT $1::json",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INVALID_INPUT_SYNTAX,
			UUID:               uuid.UUID{1},
			Query:              "SELECT $1::json",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 17,
				ByteEnd:   19,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 19,
				ByteEnd:   41,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 15,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_VALUE_TOO_LONG_FOR_TYPE,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO x(y) VALUES ('zzzzz')",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 33,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INVALID_VALUE,
			UUID:               uuid.UUID{1},
			Query:              "SELECT to_timestamp($1, 'YYYY-mm-DD')",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 15,
				ByteEnd:   21,
				Kind:      state.TableDataLogSecret,
			}, {
				ByteStart: 28,
				ByteEnd:   32,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 37,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_MALFORMED_ARRAY_LITERAL,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM x WHERE id = ANY ('a, b')",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 26,
				ByteEnd:   30,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 39,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "malformed array literal: \"{\"{\\\"bad\\\":\\\"data\\\"}\"}\"",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Unexpected array element.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "SELECT $1::text[]",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_MALFORMED_ARRAY_LITERAL,
			UUID:               uuid.UUID{1},
			Query:              "SELECT $1::text[]",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 26,
				ByteEnd:   48,
				Kind:      state.TableDataLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 17,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_SUBQUERY_MISSING_ALIAS,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM (SELECT 1)",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 24,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INSERT_TARGET_COLUMN_MISMATCH,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO x(y) VALUES (1, 2)",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 30,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_ANY_ALL_REQUIRES_ARRAY,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM x WHERE id = ANY ($1)",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 35,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COLUMN_MISSING_FROM_GROUP_BY,
			UUID:               uuid.UUID{1},
			Query:              "SELECT def, MAX(def) FROM abc",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 29,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_RELATION_DOES_NOT_EXIST,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM x",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 15,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COLUMN_DOES_NOT_EXIST,
			UUID:               uuid.UUID{1},
			Query:              "SELECT y FROM x",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 15,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COLUMN_DOES_NOT_EXIST,
			UUID:               uuid.UUID{1},
			Query:              "ALTER TABLE x DROP COLUMN y;",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 28,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COLUMN_REFERENCE_AMBIGUOUS,
			UUID:               uuid.UUID{1},
			Query:              "SELECT z FROM x, y",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 18,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_OPERATOR_DOES_NOT_EXIST,
			UUID:               uuid.UUID{1},
			Query:              "SELECT true || true",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 19,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_FUNCTION_DOES_NOT_EXIST,
			UUID:               uuid.UUID{1},
			Query:              "SELECT x(1);",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 12,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_PERMISSION_DENIED,
			UUID:               uuid.UUID{1},
			Query:              "SELECT * FROM my_schema.table",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 29,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_TRANSACTION_IS_ABORTED,
			UUID:               uuid.UUID{1},
			Query:              "SELECT 1",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 8,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_ON_CONFLICT_NO_CONSTRAINT_MATCH,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO x (y, z) VALUES ('a', 1) ON CONFLICT (y) DO UPDATE SET z = EXCLUDED.z",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 81,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_ON_CONFLICT_ROW_AFFECTED_TWICE,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO x (y, z) VALUES ('a', 1), ('a', 2) ON CONFLICT (y) DO UPDATE SET z = EXCLUDED.z",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 91,
				Kind:    state.StatementTextLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COLUMN_CANNOT_BE_CAST,
			UUID:               uuid.UUID{1},
			Query:              "SELECT abc::date FROM x",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 23,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_DIVISION_BY_ZERO,
			UUID:               uuid.UUID{1},
			Query:              "SELECT 1/0",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 10,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_CANNOT_DROP,
			UUID:               uuid.UUID{1},
			Query:              "DROP TABLE x",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 12,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INTEGER_OUT_OF_RANGE,
			UUID:               uuid.UUID{1},
			Query:              "INSERT INTO x(y) VALUES (10000000000000)",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 40,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INVALID_REGEXP,
			UUID:               uuid.UUID{1},
			Query:              "SELECT regexp_replace('test', '<(?i:test)', '');",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 48,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_PARAM_MISSING,
			UUID:               uuid.UUID{1},
			Query:              "SELECT $1;",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 10,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_NO_SUCH_SAVEPOINT,
			UUID:               uuid.UUID{1},
			Query:              "ROLLBACK TO x",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 13,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "unterminated quoted string at or near \"'1\" at character 8",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT '1",
			LogLevel: pganalyze_collector.LogLineInformation_QUERY,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_UNTERMINATED_QUOTED_STRING,
			UUID:               uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 39,
				ByteEnd:   41,
				Kind:      state.ParsingErrorLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_QUERY,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 9,
				Kind:    state.StatementTextLogSecret,
			}},
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
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_UNTERMINATED_QUOTED_IDENTIFIER,
			UUID:               uuid.UUID{1},
			Query:              "SELECT \"1",
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 43,
				ByteEnd:   45,
				Kind:      state.ParsingErrorLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 9,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "invalid byte sequence for encoding \"UTF8\": 0xd0 0x2e",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INVALID_BYTE_SEQUENCE,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 43,
				ByteEnd:   52,
				Kind:      state.TableDataLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "could not serialize access due to concurrent update",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "SELECT \"1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COULD_NOT_SERIALIZE_REPEATABLE_READ,
			UUID:               uuid.UUID{1},
			Query:              "SELECT \"1",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 9,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "could not serialize access due to read/write dependencies among transactions",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "Reason code: Canceled on identification as a pivot, during write.",
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
		}, {
			Content:  "The transaction might succeed if retried.",
			LogLevel: pganalyze_collector.LogLineInformation_HINT,
		}, {
			Content:  "SELECT \"1",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_COULD_NOT_SERIALIZE_SERIALIZABLE,
			UUID:               uuid.UUID{1},
			Query:              "SELECT \"1",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_DETAIL,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_HINT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 9,
				Kind:    state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content:  "range lower bound must be less than or equal to range upper bound",
			LogLevel: pganalyze_collector.LogLineInformation_ERROR,
			UUID:     uuid.UUID{1},
		}, {
			Content:  "COPY mytable, line 169, column rangecolumn",
			LogLevel: pganalyze_collector.LogLineInformation_CONTEXT,
		}, {
			Content:  "COPY public.mytable(\"rangecolumn\") FROM STDIN BINARY",
			LogLevel: pganalyze_collector.LogLineInformation_STATEMENT,
		}},
		[]state.LogLine{{
			LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
			Classification:     pganalyze_collector.LogLineInformation_INCONSISTENT_RANGE_BOUNDS,
			UUID:               uuid.UUID{1},
			Query:              "COPY public.mytable(\"rangecolumn\") FROM STDIN BINARY",
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 52,
				Kind:    state.StatementTextLogSecret,
			}},
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
			Query:              "SELECT abalance FROM pgbench_accounts WHERE aid = 2262632;",
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 30,
				ByteEnd:   1025,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		[]state.PostgresQuerySample{{
			Query:         "SELECT abalance FROM pgbench_accounts WHERE aid = 2262632;",
			RuntimeMs:     2334.085,
			HasExplain:    true,
			ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
			ExplainFormat: pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT,
			ExplainOutputJSON: &state.ExplainPlanContainer{
				Plan: []byte("{\n" +
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
					"	  }"),
				Triggers: &([]state.ExplainPlanTrigger{}),
			},
		}},
	},
	{
		[]state.LogLine{{
			Content: "duration: 2334.085 ms  plan:\n" +
				"	{\n" +
				"	  \"Query Text\": \"SELECT abalance FROM pgbench_accounts WHERE aid = [Your log message was truncated]",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			Details:            map[string]interface{}{"query_sample_error": "auto_explain output was truncated and can't be parsed as JSON"},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 30,
				ByteEnd:   132,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		nil,
	},
	{
		[]state.LogLine{{
			Content: "duration: 2334.085 ms  plan:\n" +
				"	{\n" +
				"	  \"Query Text\": \"SELECT abalance FROM pgbench_accounts WHERE aid = \n [Your log message was truncated]\n some other log content",
		}},
		[]state.LogLine{{
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			Details:            map[string]interface{}{"query_sample_error": "auto_explain output was truncated and can't be parsed as JSON"},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 30,
				ByteEnd:   158,
				Kind:      state.StatementTextLogSecret,
			}},
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
			Query:              "UPDATE pgbench_branches SET bbalance = bbalance + 2656 WHERE bid = 59;",
			Classification:     pganalyze_collector.LogLineInformation_STATEMENT_AUTO_EXPLAIN,
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 31,
				ByteEnd:   474,
				Kind:      state.StatementTextLogSecret,
			}},
		}},
		[]state.PostgresQuerySample{{
			Query:         "UPDATE pgbench_branches SET bbalance = bbalance + 2656 WHERE bid = 59;",
			RuntimeMs:     1681.452,
			HasExplain:    true,
			ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
			ExplainFormat: pganalyze_collector.QuerySample_TEXT_EXPLAIN_FORMAT,
			ExplainOutputText: "Update on public.pgbench_branches  (cost=0.27..8.29 rows=1 width=370) (actual rows=0 loops=1)\n" +
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
			LogLevel:           pganalyze_collector.LogLineInformation_LOG,
			Classification:     pganalyze_collector.LogLineInformation_PGA_COLLECTOR_IDENTIFY,
			UUID:               uuid.UUID{1},
			Query:              "/* pganalyze-collector */ DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: server1';\nEND$$;",
			Details:            map[string]interface{}{"config_section": "server1"},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteStart: 30,
				ByteEnd:   37,
				Kind:      state.UnidentifiedLogSecret,
			}},
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_CONTEXT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
		}, {
			LogLevel:           pganalyze_collector.LogLineInformation_STATEMENT,
			ParentUUID:         uuid.UUID{1},
			ReviewedForSecrets: true,
			SecretMarkers: []state.LogSecretMarker{{
				ByteEnd: 94,
				Kind:    state.StatementTextLogSecret,
			}},
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
			t.Errorf("For %v: log lines diff: (-want +got)\n%s", pair.logLinesIn, diff)
		}
		if diff := cfg.Compare(pair.samplesOut, s); diff != "" {
			t.Errorf("For %v: query samples diff: (-want +got)\n%s", pair.samplesOut, diff)
		}

		for idx, line := range pair.logLinesOut {
			if !line.ReviewedForSecrets && line.LogLevel != pganalyze_collector.LogLineInformation_STATEMENT && line.LogLevel != pganalyze_collector.LogLineInformation_QUERY {
				t.Errorf("Missing secret review for:\n%s %s\n", pair.logLinesIn[idx].LogLevel, pair.logLinesIn[idx].Content)
			}
		}
	}
}
