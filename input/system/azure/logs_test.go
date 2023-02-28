package azure_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/input/system/azure"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

type parseRecordTestpair struct {
	recordIn      string
	linesOut      []state.LogLine
	serverNameOut string
	errOut        error
}

var parseRecordTests = []parseRecordTestpair{
	// Flexible Server examples
	{
		`{
			"properties": {
				"timestamp": "2023-02-27 22:21:11.823 UTC",
				"processId": 2030,
				"errorLevel": "LOG",
				"sqlerrcode": "00000",
				"message": "2023-02-27 22:21:11.823 UTC [2030] [user=postgres,db=postgres,app=pgbench] LOG:  duration: 0.053 ms  bind <unnamed>: UPDATE pgbench_tellers SET tbalance = tbalance + $1 WHERE tid = $2;",
				"detail": "parameters: $1 = '4234', $2 = '2'"
			},
			"time": "2023-02-27T22:21:11.830Z",
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/FLEXIBLESERVERS/PGANALYZE-TEST",
			"location": "eastus",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent"
		}`,
		[]state.LogLine{
			{
				Content:     "duration: 0.053 ms  bind <unnamed>: UPDATE pgbench_tellers SET tbalance = tbalance + $1 WHERE tid = $2;",
				LogLevel:    pganalyze_collector.LogLineInformation_LOG,
				OccurredAt:  time.Date(2023, time.February, 27, 22, 21, 11, 823*1000*1000, time.UTC),
				Username:    "postgres",
				Database:    "postgres",
				Application: "pgbench",
				BackendPid:  2030,
			},
			{
				Content:     "parameters: $1 = '4234', $2 = '2'",
				LogLevel:    pganalyze_collector.LogLineInformation_DETAIL,
				OccurredAt:  time.Date(2023, time.February, 27, 22, 21, 11, 823*1000*1000, time.UTC),
				Username:    "postgres",
				Database:    "postgres",
				Application: "pgbench",
				BackendPid:  2030,
			},
		},
		"pganalyze-test",
		nil,
	},
	{
		`{
			"time": "2023-02-27T22:21:18.277Z",
			"properties": {
				"timestamp": "2023-02-27 22:21:18.277 UTC",
				"processId": 1984,
				"errorLevel": "LOG",
				"sqlerrcode": "00000",
				"message": "2023-02-27 22:21:18.277 UTC [1984] [user=azuresu,db=postgres,app=[unknown]] LOG:  duration: 0.028 ms  execute <unnamed>: SELECT 1"
			},
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/FLEXIBLESERVERS/PGANALYZE-TEST",
			"location": "eastus",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent"
		}`,
		[]state.LogLine{
			{
				Content:    "duration: 0.028 ms  execute <unnamed>: SELECT 1",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 22, 21, 18, 277*1000*1000, time.UTC),
				Username:   "azuresu",
				Database:   "postgres",
				BackendPid: 1984,
			},
		},
		"pganalyze-test",
		nil,
	},
	{
		`{
			"time": "2023-02-27T22:21:18.690Z",
			"properties": {
				"timestamp": "2023-02-27 22:21:18.690 UTC",
				"processId": 2046,
				"errorLevel": "LOG",
				"sqlerrcode": "00000",
				"message": "2023-02-27 22:18:54.840 UTC [2046] [user=[unknown],db=[unknown],app=[unknown]] LOG:  connection received: host=169.254.128.1 port=53002"
			},
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/FLEXIBLESERVERS/PGANALYZE-TEST",
			"location": "eastus",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent"
		}`,
		[]state.LogLine{
			{
				Content:    "connection received: host=169.254.128.1 port=53002",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 22, 18, 54, 840*1000*1000, time.UTC),
				BackendPid: 2046,
			},
		},
		"pganalyze-test",
		nil,
	},
	// Single Server examples
	{
		`{
			"LogicalServerName": "pganalyze-test-2",
			"SubscriptionId": "00000000-0000-0000-0000-000000000000",
			"ResourceGroup": "test",
			"time": "2023-02-27T23:21:40Z",
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/SERVERS/PGANALYZE-TEST-2",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent",
			"properties": {
				"prefix": "2023-02-27 23:21:40 UTC-63fd3b04.cf0-",
				"message": "connection received: host=10.0.1.4 port=38850 pid=3312",
				"detail": "",
				"errorLevel": "LOG",
				"domain": "postgres-11",
				"schemaName": "",
				"tableName": "",
				"columnName": "",
				"datatypeName": ""
			}
		}`,
		[]state.LogLine{
			{
				Content:    "connection received: host=10.0.1.4 port=38850",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 23, 21, 40, 0, time.UTC),
			},
		},
		"pganalyze-test-2",
		nil,
	},
	{
		`{
			"LogicalServerName": "pganalyze-test-2",
			"SubscriptionId": "00000000-0000-0000-0000-000000000000",
			"ResourceGroup": "test",
			"time": "2023-02-27T23:21:40Z",
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/SERVERS/PGANALYZE-TEST-2",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent",
			"properties": {
				"prefix": "2023-02-27 23:21:40 UTC-63fd3b04.cf0-",
				"message": "connection authorized: user=postgresdatabase=postgres SSL enabled (protocol=TLSv1.2, cipher=ECDHE-RSA-AES256-GCM-SHA384, compression=off)",
				"detail": "",
				"errorLevel": "LOG",
				"domain": "postgres-11",
				"schemaName": "",
				"tableName": "",
				"columnName": "",
				"datatypeName": ""
			}
		}`,
		[]state.LogLine{
			{
				Content:    "connection authorized: user=postgres database=postgres SSL enabled (protocol=TLSv1.2, cipher=ECDHE-RSA-AES256-GCM-SHA384, compression=off)",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 23, 21, 40, 0, time.UTC),
			},
		},
		"pganalyze-test-2",
		nil,
	},
	{
		`{
			"LogicalServerName": "pganalyze-test-2",
			"SubscriptionId": "00000000-0000-0000-0000-000000000000",
			"ResourceGroup": "test",
			"time": "2023-02-27T23:21:40Z",
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/SERVERS/PGANALYZE-TEST-2",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent",
			"properties": {
				"prefix": "2023-02-27 23:21:40 UTC-63fd3b04.cf0-",
				"message": "duration: 0.000 ms  statement: select count(*) from pgbench_branches",
				"detail": "",
				"errorLevel": "LOG",
				"domain": "postgres-11",
				"schemaName": "",
				"tableName": "",
				"columnName": "",
				"datatypeName": ""
			}
		}`,
		[]state.LogLine{
			{
				Content:    "duration: 0.000 ms  statement: select count(*) from pgbench_branches",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 23, 21, 40, 0, time.UTC),
			},
		},
		"pganalyze-test-2",
		nil,
	},
	{
		`{
			"LogicalServerName": "pganalyze-test-2",
			"SubscriptionId": "00000000-0000-0000-0000-000000000000",
			"ResourceGroup": "test",
			"time": "2023-02-27T23:21:40Z",
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/SERVERS/PGANALYZE-TEST-2",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent",
			"properties": {
				"prefix": "2023-02-27 23:21:40 UTC-63fd3b04.cfc-",
				"message": "duration: 0.000 ms  bind <unnamed>: UPDATE pgbench_accounts SET abalance = abalance + $1 WHERE aid = $2;",
				"detail": "parameters: $1 = '353', $2 = '9001050'",
				"errorLevel": "LOG",
				"domain": "postgres-11",
				"schemaName": "",
				"tableName": "",
				"columnName": "",
				"datatypeName": ""
			}
		}`,
		[]state.LogLine{
			{
				Content:    "duration: 0.000 ms  bind <unnamed>: UPDATE pgbench_accounts SET abalance = abalance + $1 WHERE aid = $2;",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 23, 21, 40, 0, time.UTC),
			},
			{
				Content:    "parameters: $1 = '353', $2 = '9001050'",
				LogLevel:   pganalyze_collector.LogLineInformation_DETAIL,
				OccurredAt: time.Date(2023, time.February, 27, 23, 21, 40, 0, time.UTC),
			},
		},
		"pganalyze-test-2",
		nil,
	},
	{
		`{
			"LogicalServerName": "pganalyze-test-2",
			"SubscriptionId": "00000000-0000-0000-0000-000000000000",
			"ResourceGroup": "test",
			"time": "2023-02-27T23:26:18Z",
			"resourceId": "/SUBSCRIPTIONS/00000000-0000-0000-0000-000000000000/RESOURCEGROUPS/TEST/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/SERVERS/PGANALYZE-TEST-2",
			"category": "PostgreSQLLogs",
			"operationName": "LogEvent",
			"properties": {
				"prefix": "2023-02-27 23:26:18 UTC-63fd3286.d4-",
				"message": "checkpoint complete (212): wrote 0 buffers (0.0%); 0 WAL file(s) added, 1 removed, 0 recycled; write=0.016 s, sync=0.000 s, total=0.063 s; sync files=0, longest=0.000 s, average=0.000 s; distance=4512 kB, estimate=1741524 kB",
				"detail": "",
				"errorLevel": "LOG",
				"domain": "postgres-11",
				"schemaName": "",
				"tableName": "",
				"columnName": "",
				"datatypeName": ""
			}
		}`,
		[]state.LogLine{
			{
				Content:    "checkpoint complete: wrote 0 buffers (0.0%); 0 WAL file(s) added, 1 removed, 0 recycled; write=0.016 s, sync=0.000 s, total=0.063 s; sync files=0, longest=0.000 s, average=0.000 s; distance=4512 kB, estimate=1741524 kB",
				LogLevel:   pganalyze_collector.LogLineInformation_LOG,
				OccurredAt: time.Date(2023, time.February, 27, 23, 26, 18, 0, time.UTC),
			},
		},
		"pganalyze-test-2",
		nil,
	},
}

func TestParseRecordToLogLines(t *testing.T) {
	for _, pair := range parseRecordTests {
		var record azure.AzurePostgresLogRecord
		err := json.Unmarshal([]byte(pair.recordIn), &record)
		if err != nil {
			t.Errorf("For \"%v\": expected unmarshaling to succeed, but it failed: %s\n", pair.recordIn, err)
		}
		lines, serverName, err := azure.ParseRecordToLogLines(record)
		if pair.errOut != err {
			t.Errorf("For \"%v\": expected error to be %v, but was %v\n", pair.recordIn, pair.errOut, err)
		}
		if pair.serverNameOut != serverName {
			t.Errorf("For \"%v\": expected server name to be %v, but was %v\n", pair.recordIn, pair.serverNameOut, serverName)
		}

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true
		if diff := cfg.Compare(lines, pair.linesOut); diff != "" {
			t.Errorf("For \"%v\": log line diff: (-got +want)\n%s", pair.recordIn, diff)
		}
	}
}
