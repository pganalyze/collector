package logs_test

import (
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

type parseTestpair struct {
	prefixIn  string
	lineIn    string
	lineOut   state.LogLine
	lineOutOk bool
}

var parseTests = []parseTestpair{
	// rsyslog format
	{
		"",
		"Feb  1 21:48:31 ip-172-31-14-41 postgres[9076]: [3-1] LOG:  database system is ready to accept connections",
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
	// RDS format
	{
		"",
		"2018-08-22 16:00:04 UTC:ec2-1-1-1-1.compute-1.amazonaws.com(48808):myuser@mydb:[18762]:LOG:  duration: 3668.685 ms  execute <unnamed>: SELECT 1",
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
		"",
		"2018-08-22 16:00:03 UTC:127.0.0.1(36404):myuser@mydb:[21495]:LOG:  duration: 1630.946 ms  execute 3: SELECT 1",
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
	// Simple format
	{
		"",
		"2018-05-04 03:06:18.360 UTC [3184] LOG:  pganalyze-collector-identify: server1",
		state.LogLine{
			OccurredAt: time.Date(2018, time.May, 4, 3, 6, 18, 360*1000*1000, time.UTC),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 3184,
			Content:    "pganalyze-collector-identify: server1",
		},
		true,
	},
	{
		"",
		"2018-05-04 03:06:18.360 +0100 [3184] LOG:  pganalyze-collector-identify: server1",
		state.LogLine{
			OccurredAt: time.Date(2018, time.May, 4, 3, 6, 18, 360*1000*1000, time.FixedZone("+0100", 3600)),
			LogLevel:   pganalyze_collector.LogLineInformation_LOG,
			BackendPid: 3184,
			Content:    "pganalyze-collector-identify: server1",
		},
		true,
	},
}

func TestParseLogLineWithPrefix(t *testing.T) {
	for _, pair := range parseTests {
		l, lOk := logs.ParseLogLineWithPrefix(pair.prefixIn, pair.lineIn)

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		if pair.lineOutOk != lOk {
			t.Errorf("For \"%v\": expected parsing ok? to be %v, but was %v\n", pair.lineIn, pair.lineOutOk, lOk)
		}

		if diff := cfg.Compare(pair.lineOut, l); diff != "" {
			t.Errorf("For \"%v\": log line diff: (-got +want)\n%s", pair.lineIn, diff)
		}
	}
}
