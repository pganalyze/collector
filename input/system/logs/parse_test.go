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
