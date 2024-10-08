package logs_test

import (
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

type parseTestpair struct {
	prefixIn  string
	lineIn    string
	lineInTz  *time.Location
	lineOut   state.LogLine
	lineOutOk bool
}

func mustTimeLocation(tzStr string) *time.Location {
	tz, err := time.LoadLocation(tzStr)
	if err != nil {
		panic(err)
	}
	return tz
}

var BSTTimeLocation = mustTimeLocation("Europe/London")

var parseTests = []parseTestpair{
	// rsyslog format
	{
		"",
		"Feb  1 21:48:31 ip-172-31-14-41 postgres[9076]: [3-1] LOG:  database system is ready to accept connections",
		nil,
		state.LogLine{
			OccurredAt: time.Date(time.Now().Year(), time.February, 1, 21, 48, 31, 0, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 9076,
			Content:    "database system is ready to accept connections",
		},
		true,
	},
	{
		"",
		"Feb  1 21:48:31 ip-172-31-14-41 postgres[9076]: [3-2] #011 something",
		nil,
		state.LogLine{
			OccurredAt: time.Date(time.Now().Year(), time.February, 1, 21, 48, 31, 0, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_UNKNOWN,
			BackendPid: 9076,
			Content:    "\t something",
		},
		false,
	},
	{
		"",
		"Feb  1 21:48:31 ip-172-31-14-41 postgres[123]: [8-1] [user=postgres,db=postgres,app=[unknown]] LOG: connection received: host=[local]",
		nil,
		state.LogLine{
			OccurredAt: time.Date(time.Now().Year(), time.February, 1, 21, 48, 31, 0, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 123,
			Username:   "postgres",
			Database:   "postgres",
			Content:    "connection received: host=[local]",
		},
		true,
	},
	// Amazon RDS format
	{
		logs.LogPrefixAmazonRds,
		"2018-08-22 16:00:04 UTC:ec2-1-1-1-1.compute-1.amazonaws.com(48808):myuser@mydb:[18762]:LOG:  duration: 3668.685 ms  execute <unnamed>: SELECT 1",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.August, 22, 16, 0, 4, 0, time.UTC),
			Username:   "myuser",
			Database:   "mydb",
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 18762,
			Content:    "duration: 3668.685 ms  execute <unnamed>: SELECT 1",
		},
		true,
	},
	{
		logs.LogPrefixAmazonRds,
		"2018-08-22 16:00:03 UTC:127.0.0.1(36404):myuser@mydb:[21495]:LOG:  duration: 1630.946 ms  execute 3: SELECT 1",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.August, 22, 16, 0, 3, 0, time.UTC),
			Username:   "myuser",
			Database:   "mydb",
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 21495,
			Content:    "duration: 1630.946 ms  execute 3: SELECT 1",
		},
		true,
	},
	{
		logs.LogPrefixAmazonRds,
		"2018-08-22 16:00:03 UTC:[local]:myuser@mydb:[21495]:LOG:  duration: 1630.946 ms  execute 3: SELECT 1",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.August, 22, 16, 0, 3, 0, time.UTC),
			Username:   "myuser",
			Database:   "mydb",
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 21495,
			Content:    "duration: 1630.946 ms  execute 3: SELECT 1",
		},
		true,
	},
	// Azure format
	{
		logs.LogPrefixAzure,
		"2020-06-21 22:37:10 UTC-5eefe116.22f4-LOG:  could not receive data from client: An existing connection was forcibly closed by the remote host.",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2020, time.June, 21, 22, 37, 10, 0, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			Content:    "could not receive data from client: An existing connection was forcibly closed by the remote host.",
		},
		true,
	},
	// Custom 1 format
	{
		logs.LogPrefixCustom1,
		"2018-09-27 06:57:01.030 EST [20194][] : [1-1] [app=pganalyze_collector] LOG:  connection received: host=[local]",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.September, 27, 6, 57, 1, 30*1000*1000, time.FixedZone("EST", -5*3600)),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    20194,
			LogLineNumber: 1,
			Application:   "pganalyze_collector",
			Content:       "connection received: host=[local]",
		},
		true,
	},
	// Custom 2 format
	{
		logs.LogPrefixCustom2,
		"2018-09-28 07:37:59 UTC [331-1] postgres@postgres LOG:  connection received: host=[local]",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.September, 28, 7, 37, 59, 0, time.UTC),
			Username:      "postgres",
			Database:      "postgres",
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    331,
			LogLineNumber: 1,
			Application:   "",
			Content:       "connection received: host=[local]",
		},
		true,
	},
	// Custom 3 format
	{
		logs.LogPrefixCustom3,
		"2018-09-27 06:57:01.030 UTC [20194] [user=[unknown],db=[unknown],app=[unknown]] LOG:  connection received: host=[local]",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.September, 27, 6, 57, 1, 30*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 20194,
			Content:    "connection received: host=[local]",
		},
		true,
	},
	{
		logs.LogPrefixCustom3,
		"2018-09-27 06:57:02.779 UTC [20194] [user=postgres,db=postgres,app=psql] ERROR:  canceling statement due to user request",
		nil,
		state.LogLine{
			OccurredAt:  time.Date(2018, time.September, 27, 6, 57, 2, 779*1000*1000, time.UTC),
			Username:    "postgres",
			Database:    "postgres",
			Application: "psql",
			LogLevel:    pganalyze_collector.LogLineInformation_ERROR,
			BackendPid:  20194,
			Content:     "canceling statement due to user request",
		},
		true,
	},
	{
		logs.LogPrefixCustom3,
		"2018-09-27 06:57:02.779 UTC [20194] [user=postgres,db=postgres,app=psql] LOG:  duration: 3000.019 ms  statement: SELECT pg_sleep(3\n);",
		nil,
		state.LogLine{
			OccurredAt:  time.Date(2018, time.September, 27, 6, 57, 2, 779*1000*1000, time.UTC),
			Username:    "postgres",
			Database:    "postgres",
			Application: "psql",
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			BackendPid:  20194,
			Content:     "duration: 3000.019 ms  statement: SELECT pg_sleep(3\n);",
		},
		true,
	},
	{
		logs.LogPrefixCustom3,
		"2018-09-27 06:57:01.030 BST [20194] [user=[unknown],db=[unknown],app=[unknown]] LOG:  connection received: host=[local]",
		BSTTimeLocation,
		state.LogLine{
			OccurredAt: time.Date(2018, time.September, 27, 6, 57, 1, 30*1000*1000, BSTTimeLocation),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 20194,
			Content:    "connection received: host=[local]",
		},
		true,
	},
	{
		logs.LogPrefixCustom3,
		"2018-09-27 06:57:02.779 UTC [20194] [user=postgres,db=postgres,app=sidekiq 1.2.3 queues:something[0 of 50 busy]] LOG:  duration: 3000.019 ms  statement: SELECT pg_sleep(3);",
		nil,
		state.LogLine{
			OccurredAt:  time.Date(2018, time.September, 27, 6, 57, 2, 779*1000*1000, time.UTC),
			Username:    "postgres",
			Database:    "postgres",
			Application: "sidekiq 1.2.3 queues:something[0 of 50 busy]",
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			BackendPid:  20194,
			Content:     "duration: 3000.019 ms  statement: SELECT pg_sleep(3);",
		},
		true,
	},
	// Custom 4 format
	{
		logs.LogPrefixCustom4,
		"2018-09-27 06:57:01.030 UTC [20194] [user=[unknown],db=[unknown],app=[unknown],host=[local]] LOG:  connection received: host=[local]",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.September, 27, 6, 57, 1, 30*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 20194,
			Content:    "connection received: host=[local]",
		},
		true,
	},
	{
		logs.LogPrefixCustom4,
		"2018-09-27 06:57:02.779 UTC [20194] [user=postgres,db=postgres,app=psql,host=127.0.0.1] ERROR:  canceling statement due to user request",
		nil,
		state.LogLine{
			OccurredAt:  time.Date(2018, time.September, 27, 6, 57, 2, 779*1000*1000, time.UTC),
			Username:    "postgres",
			Database:    "postgres",
			Application: "psql",
			LogLevel:    pganalyze_collector.LogLineInformation_ERROR,
			BackendPid:  20194,
			Content:     "canceling statement due to user request",
		},
		true,
	},
	// Custom 5 format
	{
		logs.LogPrefixCustom5,
		"2018-09-28 07:37:59 UTC [331]: [1-1] user=[unknown],db=[unknown] - PG-00000 LOG:  connection received: host=127.0.0.1 port=49738",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.September, 28, 7, 37, 59, 0, time.UTC),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    331,
			LogLineNumber: 1,
			Content:       "connection received: host=127.0.0.1 port=49738",
		},
		true,
	},
	{
		logs.LogPrefixCustom5,
		"2018-09-28 07:39:48 UTC [347]: [3-1] user=postgres,db=postgres - PG-57014 ERROR:  canceling statement due to user request",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.September, 28, 7, 39, 48, 0, time.UTC),
			Username:      "postgres",
			Database:      "postgres",
			LogLevel:      pganalyze_collector.LogLineInformation_ERROR,
			BackendPid:    347,
			LogLineNumber: 3,
			Content:       "canceling statement due to user request",
		},
		true,
	},
	// Custom 6 format
	{
		logs.LogPrefixCustom6,
		"2018-10-16 01:25:58 UTC [93897]: [4-1] user=,db=,app=,client= LOG:  database system is ready to accept connections",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.October, 16, 1, 25, 58, 0, time.UTC),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    93897,
			LogLineNumber: 4,
			Content:       "database system is ready to accept connections",
		},
		true,
	},
	{
		logs.LogPrefixCustom6,
		"2018-10-16 01:26:09 UTC [93907]: [1-1] user=[unknown],db=[unknown],app=[unknown],client=::1 LOG:  connection received: host=::1 port=61349",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.October, 16, 1, 26, 9, 0, time.UTC),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    93907,
			LogLineNumber: 1,
			Content:       "connection received: host=::1 port=61349",
		},
		true,
	},
	{
		// Updating this test to reflect that %a (application name) is now captured.
		// In the original parsing logic (see parse.go:299), we explicitly skip
		// setting %a even though it's captured by the prefix.
		logs.LogPrefixCustom6,
		"2018-10-16 01:26:33 UTC [93911]: [3-1] user=postgres,db=postgres,app=psql,client=::1 ERROR:  canceling statement due to user request",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2018, time.October, 16, 1, 26, 33, 0, time.UTC),
			Username:      "postgres",
			Database:      "postgres",
			Application:   "psql",
			LogLevel:      pganalyze_collector.LogLineInformation_ERROR,
			BackendPid:    93911,
			LogLineNumber: 3,
			Content:       "canceling statement due to user request",
		},
		true,
	},
	// Custom 7 format
	{
		logs.LogPrefixCustom7,
		"2019-01-01 01:59:42 UTC [1]: [4-1] [trx_id=0] user=,db= LOG:  database system is ready to accept connections",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2019, time.January, 1, 1, 59, 42, 0, time.UTC),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    1,
			LogLineNumber: 4,
			Content:       "database system is ready to accept connections",
		},
		true,
	},
	{
		logs.LogPrefixCustom7,
		"2019-01-01 02:00:28 UTC [35]: [1-1] [trx_id=0] user=[unknown],db=[unknown] LOG:  connection received: host=::1 port=38842",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2019, time.January, 1, 2, 0, 28, 0, time.UTC),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    35,
			LogLineNumber: 1,
			Content:       "connection received: host=::1 port=38842",
		},
		true,
	},
	{
		logs.LogPrefixCustom7,
		"2019-01-01 02:00:28 UTC [34]: [3-1] [trx_id=120950] user=postgres,db=postgres ERROR:  canceling statement due to user request",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2019, time.January, 1, 2, 0, 28, 0, time.UTC),
			Username:      "postgres",
			Database:      "postgres",
			LogLevel:      pganalyze_collector.LogLineInformation_ERROR,
			BackendPid:    34,
			LogLineNumber: 3,
			Content:       "canceling statement due to user request",
		},
		true,
	},
	// Custom 8 format
	{
		logs.LogPrefixCustom8,
		"[1127]: [8-1] db=postgres,user=pganalyze LOG:  duration: 2001.842 ms  statement: SELECT pg_sleep(2);",
		nil,
		state.LogLine{
			OccurredAt:    time.Time{},
			Username:      "pganalyze",
			Database:      "postgres",
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    1127,
			LogLineNumber: 8,
			Content:       "duration: 2001.842 ms  statement: SELECT pg_sleep(2);",
		},
		true,
	},
	// Custom 9 format
	{
		logs.LogPrefixCustom9,
		"2020-05-21 17:53:05.307 UTC    [5ec6bfff.1] [1] LOG:  database system is ready to accept connections",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2020, time.May, 21, 17, 53, 05, 307*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 1,
			Content:    "database system is ready to accept connections",
		},
		true,
	},
	{
		logs.LogPrefixCustom9,
		"2020-05-21 17:54:35.256 UTC 172.18.0.1(56402) pgaweb [unknown] [5ec6c05b.22] [34] LOG:  connection authorized: user=pgaweb database=pgaweb application_name=psql",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2020, time.May, 21, 17, 54, 35, 256*1000*1000, time.UTC),
			Username:   "pgaweb",
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 34,
			Content:    "connection authorized: user=pgaweb database=pgaweb application_name=psql",
		},
		true,
	},
	{
		logs.LogPrefixCustom9,
		"2020-05-21 17:54:43.808 UTC 172.18.0.1(56402) pgaweb psql [5ec6c05b.22] [34] LOG:  disconnection: session time: 0:00:08.574 user=pgaweb database=pgaweb host=172.18.0.1 port=56402",
		nil,
		state.LogLine{
			OccurredAt:  time.Date(2020, time.May, 21, 17, 54, 43, 808*1000*1000, time.UTC),
			Username:    "pgaweb",
			Application: "psql",
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			BackendPid:  34,
			Content:     "disconnection: session time: 0:00:08.574 user=pgaweb database=pgaweb host=172.18.0.1 port=56402",
		},
		true,
	},
	// Custom 10 format
	{
		logs.LogPrefixCustom10,
		"2020-09-04 16:03:11.375 UTC [417880]: [1-1] db=mydb,user=myuser LOG:  pganalyze-collector-identify: myserver",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2020, time.September, 4, 16, 3, 11, 375*1000*1000, time.UTC),
			Username:      "myuser",
			Database:      "mydb",
			BackendPid:    417880,
			LogLineNumber: 1,
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			Content:       "pganalyze-collector-identify: myserver",
		},
		true,
	},
	{
		logs.LogPrefixCustom10,
		// Database and user are empty
		"2024-10-04 06:38:28.808 UTC [2569017]: [1-1] db=,user= LOG: automatic vacuum of table \"some_database.some_schema.some_table\": index scans: 0\npages: 0 removed, 14 remain, 0 skipped due to pins, 0 skipped frozen\ntuples: 0 removed, 107 remain, 94 are dead but not yet removable, oldest xmin: 17243830\nindex scan not needed: 0 pages from table (0.00% of total) had 0 dead item identifiers removed\nI/O timings: read: 0.000 ms, write: 0.000 ms\navg read rate: 0.000 MB/s, avg write rate: 0.000 MB/s\nbuffer usage: 52 hits, 0 misses, 0 dirtied\nWAL usage: 0 records, 0 full page images, 0 bytes\nsystem usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.00 s",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2024, time.October, 4, 6, 38, 28, 808*1000*1000, time.UTC),
			Username:      "",
			Database:      "",
			BackendPid:    2569017,
			LogLineNumber: 1,
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			Content:       "automatic vacuum of table \"some_database.some_schema.some_table\": index scans: 0\npages: 0 removed, 14 remain, 0 skipped due to pins, 0 skipped frozen\ntuples: 0 removed, 107 remain, 94 are dead but not yet removable, oldest xmin: 17243830\nindex scan not needed: 0 pages from table (0.00% of total) had 0 dead item identifiers removed\nI/O timings: read: 0.000 ms, write: 0.000 ms\navg read rate: 0.000 MB/s, avg write rate: 0.000 MB/s\nbuffer usage: 52 hits, 0 misses, 0 dirtied\nWAL usage: 0 records, 0 full page images, 0 bytes\nsystem usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.00 s",
		},
		true,
	},
	// Custom 11 format
	{
		logs.LogPrefixCustom11,
		"pid=8284,user=[unknown],db=[unknown],app=[unknown],client=[local] LOG: connection received: host=[local]",
		nil,
		state.LogLine{
			BackendPid: 8284,
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			Content:    "connection received: host=[local]",
		},
		true,
	},
	{
		// Updating this to reflect that we can now capture the unusual application
		// name (previously Application was omitted in the expected output struct)
		logs.LogPrefixCustom11,
		"pid=8284,user=[unknown],db=[unknown],app=why would you[] name your application this,client=[local] LOG: connection received: host=[local]",
		nil,
		state.LogLine{
			Application: "why would you[] name your application this",
			BackendPid:  8284,
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			Content:     "connection received: host=[local]",
		},
		true,
	},
	// Custom 12 format
	{
		logs.LogPrefixCustom12,
		"user=[unknown],db=[unknown],app=[unknown],client=[local] LOG: connection received: host=[local]",
		nil,
		state.LogLine{
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			Content:  "connection received: host=[local]",
		},
		true,
	},
	// Custom 13 format
	{
		logs.LogPrefixCustom13,
		"27-2021-11-17 19:06:14 UTC-619552a6.1b-1----2021-11-17 19:06:14.946 UTC LOG:  database system was shut down at 2021-11-17 19:01:42 UTC",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2021, time.November, 17, 19, 6, 14, 946*1000*1000, time.UTC),
			BackendPid:    27,
			LogLineNumber: 1,
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			Content:       "database system was shut down at 2021-11-17 19:01:42 UTC",
		},
		true,
	},
	{
		logs.LogPrefixCustom13,
		"51-2021-11-17 19:11:13 UTC-619553d1.33-2-172.20.0.1-pgaweb-pgaweb-2021-11-17 19:11:13.562 UTC LOG:  connection authorized: user=pgaweb database=pgaweb application_name=puma: cluster worker 2: 18544 [pganalyze]",
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2021, time.November, 17, 19, 11, 13, 562*1000*1000, time.UTC),
			Username:      "pgaweb",
			Database:      "pgaweb",
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			BackendPid:    51,
			LogLineNumber: 2,
			Content:       "connection authorized: user=pgaweb database=pgaweb application_name=puma: cluster worker 2: 18544 [pganalyze]",
		},
		true,
	},
	// Custom 14 format
	{
		logs.LogPrefixCustom14,
		"2021-11-17 19:06:53.897 UTC [34][autovacuum worker][3/5][22996] LOG:  automatic analyze of table \"mydb.pg_catalog.pg_class\" system usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.01 s",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2021, time.November, 17, 19, 6, 53, 897*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 34,
			Content:    "automatic analyze of table \"mydb.pg_catalog.pg_class\" system usage: CPU: user: 0.00 s, system: 0.00 s, elapsed: 0.01 s",
		},
		true,
	},
	// Custom 15 format
	{
		logs.LogPrefixCustom15,
		"2022-07-22 06:13:13.389 UTC [1] LOG:  database system is ready to accept connections",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2022, time.July, 22, 6, 13, 13, 389*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 1,
			Content:    "database system is ready to accept connections",
		},
		true,
	},
	{
		logs.LogPrefixCustom15,
		"2022-07-22 06:13:45.781 UTC [75] myuser@mydb LOG:  connection authorized: user=myuser database=mydb application_name=psql",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2022, time.July, 22, 6, 13, 45, 781*1000*1000, time.UTC),
			Username:   "myuser",
			Database:   "mydb",
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 75,
			Content:    "connection authorized: user=myuser database=mydb application_name=psql",
		},
		true,
	},
	// Custom 16 format
	{
		logs.LogPrefixCustom16,
		"2022-07-22 06:13:12 UTC [1] LOG:  starting PostgreSQL 14.2 (Debian 14.2-1.pgdg110+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 10.2.1-6) 10.2.1 20210110, 64-bit",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2022, time.July, 22, 6, 13, 12, 0, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 1,
			Content:    "starting PostgreSQL 14.2 (Debian 14.2-1.pgdg110+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 10.2.1-6) 10.2.1 20210110, 64-bit",
		},
		true,
	},
	{
		logs.LogPrefixCustom16,
		"2022-07-22 06:14:23 UTC [76] my-user@my-db 1.2.3.4 LOG:  disconnection: session time: 0:00:01.667 user=my-user database=my-db host=1.2.3.4 port=5678",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2022, time.July, 22, 6, 14, 23, 0, time.UTC),
			Username:   "my-user",
			Database:   "my-db",
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 76,
			Content:    "disconnection: session time: 0:00:01.667 user=my-user database=my-db host=1.2.3.4 port=5678",
		},
		true,
	},
	// Simple format
	{
		logs.LogPrefixSimple,
		"2018-05-04 03:06:18.360 UTC [3184] LOG:  pganalyze-collector-identify: server1",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.May, 4, 3, 6, 18, 360*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 3184,
			Content:    "pganalyze-collector-identify: server1",
		},
		true,
	},
	{
		logs.LogPrefixSimple,
		"2018-05-04 03:06:18.360 +0100 [3184] LOG:  pganalyze-collector-identify: server1",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2018, time.May, 4, 3, 6, 18, 360*1000*1000, time.FixedZone("+0100", 3600)),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 3184,
			Content:    "pganalyze-collector-identify: server1",
		},
		true,
	},
	{
		logs.LogPrefixSimple,
		"2022-12-23 09:53:43.862 -03 [790081] LOG: pganalyze-collector-identify: server1",
		nil,
		state.LogLine{
			OccurredAt: time.Date(2022, time.December, 23, 9, 53, 43, 862*1000*1000, time.FixedZone("-03", -3*3600)),
			LogLevel:   6,
			BackendPid: 790081,
			Content:    "pganalyze-collector-identify: server1",
		},
		true,
	},
	// Heroku format
	{
		logs.LogPrefixHeroku1,
		` sql_error_code = 28000 FATAL:  no pg_hba.conf entry for host "127.0.0.1", user "postgres", database "postgres", SSL off`,
		nil,
		state.LogLine{
			LogLevel: pganalyze_collector.LogLineInformation_FATAL,
			Content:  "no pg_hba.conf entry for host \"127.0.0.1\", user \"postgres\", database \"postgres\", SSL off",
		},
		true,
	},
	{
		logs.LogPrefixHeroku2,
		` sql_error_code = 28000 time_ms = "2022-06-02 22:48:20.807 UTC" pid="11666" proc_start_time="2022-06-02 22:48:20 UTC" session_id="62993e34.2d92" vtid="6/17007" tid="0" log_line="1" database="postgres" connection_source="127.0.0.1(36532)" user="postgres" application_name="[unknown]" FATAL:  no pg_hba.conf entry for host "127.0.0.1", user "postgres", database "postgres", SSL off`,
		nil,
		state.LogLine{
			OccurredAt:    time.Date(2022, time.June, 2, 22, 48, 20, 807*1000*1000, time.UTC),
			Username:      "postgres",
			Database:      "postgres",
			LogLevel:      pganalyze_collector.LogLineInformation_FATAL,
			BackendPid:    11666,
			LogLineNumber: 1,
			Content:       "no pg_hba.conf entry for host \"127.0.0.1\", user \"postgres\", database \"postgres\", SSL off",
		},
		true,
	},
}

func TestLogParser(t *testing.T) {
	for _, pair := range parseTests {
		// Syslog format has a separate, fixed prefix, so the prefix argument is
		// ignored by the parser in that case. We use an empty string to indicate
		// that this is a syslog test case.
		isSyslog := pair.prefixIn == ""
		parser := logs.NewLogParser(pair.prefixIn, pair.lineInTz, isSyslog)
		l, lOk := parser.ParseLine(pair.lineIn)

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		if pair.lineOutOk != lOk {
			t.Errorf("For \"%v\": expected parsing ok? to be %v, but was %v\n", pair.lineIn, pair.lineOutOk, lOk)
		}

		if diff := cfg.Compare(l, pair.lineOut); diff != "" {
			t.Errorf("For \"%v\": log line diff: (-got +want)\n%s", pair.lineIn, diff)
		}
	}
}
