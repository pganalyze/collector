package stream_test

import (
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

type testpair struct {
	logLines          []state.LogLine
	TransientLogState state.TransientLogState
	logFile           state.LogFile
	logFileContent    string
	tooFreshLogLines  []state.LogLine
	err               error
}

var now = time.Now()

var tests = []testpair{
	// Simple case
	{
		[]state.LogLine{{
			CollectedAt: now.Add(-5 * time.Second),
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			Content:     "duration: 10003.847 ms  statement: SELECT pg_sleep(10);\n",
		}},
		state.TransientLogState{
			QuerySamples: []state.PostgresQuerySample{{
				Query:     "SELECT pg_sleep(10);",
				RuntimeMs: 10003.847,
			}},
		},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt:    now.Add(-5 * time.Second),
				LogLevel:       pganalyze_collector.LogLineInformation_LOG,
				ByteEnd:        56,
				Query:          "SELECT pg_sleep(10);",
				Classification: 80,
				Details: map[string]interface{}{
					"duration_ms": 10003.847,
				},
				ReviewedForSecrets: true,
				SecretMarkers: []state.LogSecretMarker{{
					ByteStart: 35,
					ByteEnd:   55,
					Kind:      3,
				}},
			}},
		},
		"duration: 10003.847 ms  statement: SELECT pg_sleep(10);\n",
		[]state.LogLine{},
		nil,
	},
	// Too fresh
	{
		[]state.LogLine{{
			CollectedAt: now,
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			Content:     "duration: 10003.847 ms  statement: SELECT pg_sleep(10);\n",
		}},
		state.TransientLogState{},
		state.LogFile{},
		"",
		[]state.LogLine{{
			CollectedAt: now,
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			Content:     "duration: 10003.847 ms  statement: SELECT pg_sleep(10);\n",
		}},
		nil,
	},
	// Multiple lines (all of same timestamp)
	{
		[]state.LogLine{{
			CollectedAt: now.Add(-5 * time.Second),
			LogLevel:    pganalyze_collector.LogLineInformation_ERROR,
			Content:     "permission denied for function pg_reload_conf\n",
		},
			{
				CollectedAt: now.Add(-5 * time.Second),
				LogLevel:    pganalyze_collector.LogLineInformation_STATEMENT,
				Content:     "SELECT pg_reload_conf();\n",
			}},
		state.TransientLogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt:        now.Add(-5 * time.Second),
				LogLevel:           pganalyze_collector.LogLineInformation_ERROR,
				ByteEnd:            46,
				Query:              "SELECT pg_reload_conf();\n",
				Classification:     123,
				ReviewedForSecrets: true,
			},
				{
					CollectedAt:      now.Add(-5 * time.Second),
					LogLevel:         pganalyze_collector.LogLineInformation_STATEMENT,
					ByteStart:        46,
					ByteContentStart: 46,
					ByteEnd:          71,
				}},
		},
		"permission denied for function pg_reload_conf\nSELECT pg_reload_conf();\n",
		[]state.LogLine{},
		nil,
	},
	// Multiple lines (different timestamps, skips freshness check due to missing level *and PID*)
	{
		[]state.LogLine{{
			CollectedAt: now.Add(-5 * time.Second),
			LogLevel:    pganalyze_collector.LogLineInformation_LOG,
			Content:     "LOG:  duration: 10010.397 ms  statement: SELECT pg_sleep(10\n",
		},
			{
				CollectedAt: now,
				Content:     " );\n",
			}},
		state.TransientLogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt: now.Add(-5 * time.Second),
				LogLevel:    pganalyze_collector.LogLineInformation_LOG,
				ByteEnd:     64,
			}},
		},
		"LOG:  duration: 10010.397 ms  statement: SELECT pg_sleep(10\n );\n",
		[]state.LogLine{},
		nil,
	},
	// Multiple lines (different timestamps, skips freshness check due to missing level *and PID* only for unknown lines)
	{
		[]state.LogLine{{
			CollectedAt:   now.Add(-5 * time.Second),
			LogLevel:      pganalyze_collector.LogLineInformation_LOG,
			LogLineNumber: 2,
			BackendPid:    42,
			Content:       "LOG:  duration: 10010.397 ms  statement: SELECT pg_sleep(10\n",
		},
			{
				CollectedAt: now,
				Content:     " );\n",
			}},
		state.TransientLogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt:   now.Add(-5 * time.Second),
				LogLevel:      pganalyze_collector.LogLineInformation_LOG,
				ByteEnd:       64,
				LogLineNumber: 2,
				BackendPid:    42,
			}},
		},
		"LOG:  duration: 10010.397 ms  statement: SELECT pg_sleep(10\n );\n",
		[]state.LogLine{},
		nil,
	},
	// Multiple lines (different timestamps, requiring skip of freshness check due to log line number)
	//
	// Note that this refers to the Heroku case, where we have log line numbers on unidentified lines
	// (because logplex adds them, not Postgres itself, like in other cases)
	{
		[]state.LogLine{{
			CollectedAt:   now,
			LogLineNumber: 2,
			BackendPid:    42,
			Content:       "second\n",
		},
			{
				CollectedAt:   now.Add(-5 * time.Second),
				LogLevel:      pganalyze_collector.LogLineInformation_LOG,
				LogLineNumber: 1,
				BackendPid:    42,
				Content:       "first\n",
			}},
		state.TransientLogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt:   now.Add(-5 * time.Second),
				LogLevel:      pganalyze_collector.LogLineInformation_LOG,
				LogLineNumber: 1,
				ByteEnd:       13,
				BackendPid:    42,
			}},
		},
		"first\nsecond\n",
		[]state.LogLine{},
		nil,
	},
	// Multiple lines not concatenated yet (use case for self-managed systems)
	{
		[]state.LogLine{
			{
				CollectedAt: now.Add(-10 * time.Second),
				OccurredAt:  now.Add(-10 * time.Second),
				LogLevel:    pganalyze_collector.LogLineInformation_LOG,
				BackendPid:  80,
				Content:     "zero\n",
			},
			{
				CollectedAt: now.Add(-5 * time.Second),
				OccurredAt:  now.Add(-5 * time.Second),
				LogLevel:    pganalyze_collector.LogLineInformation_LOG,
				BackendPid:  42,
				Content:     "first\n",
			},
			{
				CollectedAt: now.Add(-5 * time.Second),
				Content:     "second\n",
			},
			{
				CollectedAt: now.Add(-3 * time.Second),
				OccurredAt:  now.Add(-3 * time.Second),
				LogLevel:    pganalyze_collector.LogLineInformation_ERROR,
				BackendPid:  77,
				Content:     "third\n",
			},
		},
		state.TransientLogState{},
		state.LogFile{
			LogLines: []state.LogLine{
				{
					CollectedAt: now.Add(-10 * time.Second),
					OccurredAt:  now.Add(-10 * time.Second),
					LogLevel:    pganalyze_collector.LogLineInformation_LOG,
					ByteEnd:     5,
					BackendPid:  80,
				},
				{
					CollectedAt:      now.Add(-5 * time.Second),
					OccurredAt:       now.Add(-5 * time.Second),
					LogLevel:         pganalyze_collector.LogLineInformation_LOG,
					ByteStart:        5,
					ByteContentStart: 5,
					ByteEnd:          18,
					BackendPid:       42,
				},
				{
					CollectedAt:      now.Add(-3 * time.Second),
					OccurredAt:       now.Add(-3 * time.Second),
					LogLevel:         pganalyze_collector.LogLineInformation_ERROR,
					ByteStart:        18,
					ByteContentStart: 18,
					ByteEnd:          24,
					BackendPid:       77,
				},
			},
		},
		"zero\nfirst\nsecond\nthird\n",
		[]state.LogLine{},
		nil,
	},
	//{
	// There should be a test for this method
	// - Pass in two logLines, one at X, one at X + 2, and assume the time is x + 3
	// - These lines should be concatenated based on the log line number, ignoring the fact the the second log line would be considered too fresh
	//	[]state.LogLine{{
	//
	//	}},
	//},
}

func TestAnalyzeStreamInGroups(t *testing.T) {
	for _, pair := range tests {
		TransientLogState, logFile, tooFreshLogLines, err := stream.AnalyzeStreamInGroups(pair.logLines)
		logFileContent := ""
		if logFile.TmpFile != nil {
			dat, err := ioutil.ReadFile(logFile.TmpFile.Name())
			if err != nil {
				t.Errorf("Error reading temporary log file: %s", err)
			}
			logFileContent = string(dat)
		}

		TransientLogState.CollectedAt = time.Time{} // Avoid comparing against time.Now()
		logFile.TmpFile = nil                       // Avoid comparing against tempfile
		logFile.UUID = uuid.UUID{}                  // Avoid comparing against a generated UUID

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		if diff := cfg.Compare(pair.TransientLogState, TransientLogState); diff != "" {
			t.Errorf("For %v: log state diff: (-want +got)\n%s", pair.TransientLogState, diff)
		}
		sort.SliceStable(pair.logFile.LogLines, func(i, j int) bool {
			return pair.logFile.LogLines[i].ByteStart < pair.logFile.LogLines[j].ByteStart
		})
		sort.SliceStable(logFile.LogLines, func(i, j int) bool {
			return logFile.LogLines[i].ByteStart < logFile.LogLines[j].ByteStart
		})
		if diff := cfg.Compare(pair.logFile, logFile); diff != "" {
			t.Errorf("For %v: log file diff: (-want +got)\n%s", pair.logFile, diff)
		}
		if diff := cfg.Compare(pair.logFileContent, logFileContent); diff != "" {
			t.Errorf("For %v: log state diff: (-want +got)\n%s", pair.logFileContent, diff)
		}
		if diff := cfg.Compare(pair.tooFreshLogLines, tooFreshLogLines); diff != "" {
			t.Errorf("For %v: too fresh log lines diff: (-want +got)\n%s", pair.tooFreshLogLines, diff)
		}
		if diff := cfg.Compare(pair.err, err); diff != "" {
			t.Errorf("For %v: err diff: (-want +got)\n%s", pair.err, diff)
		}
	}
}
