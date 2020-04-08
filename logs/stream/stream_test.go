package stream_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

type testpair struct {
	logLines  []state.LogLine
	logState state.LogState
	logFile state.LogFile
	logFileContent string
	tooFreshLogLines []state.LogLine
	err error
}

var now = time.Now()

var tests = []testpair{
	// Simple case
	{
		[]state.LogLine{{
			CollectedAt: now.Add(- 5 * time.Second),
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			Content: "dummy\n",
		}},
		state.LogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt: now.Add(- 5 * time.Second),
				LogLevel: pganalyze_collector.LogLineInformation_LOG,
				ByteEnd: 6,
			}},
		},
		"dummy\n",
		[]state.LogLine{},
		nil,
	},
	// Too fresh
	{
		[]state.LogLine{{
			CollectedAt: now,
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			Content: "dummy\n",
		}},
		state.LogState{},
		state.LogFile{},
		"",
		[]state.LogLine{{
			CollectedAt: now,
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			Content: "dummy\n",
		}},
		nil,
	},
	// Multiple lines (all of same timestamp)
	{
		[]state.LogLine{{
			CollectedAt: now.Add(- 5 * time.Second),
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			Content: "first\n",
		},
		{
			CollectedAt: now.Add(- 5 * time.Second),
			LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
			Content: "second\n",
		}},
		state.LogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt: now.Add(- 5 * time.Second),
				LogLevel: pganalyze_collector.LogLineInformation_LOG,
				ByteEnd: 6,
			},
			{
				CollectedAt: now.Add(- 5 * time.Second),
				LogLevel: pganalyze_collector.LogLineInformation_DETAIL,
				ByteStart: 6,
				ByteContentStart: 6,
				ByteEnd: 13,
			}},
		},
		"first\nsecond\n",
		[]state.LogLine{},
		nil,
	},
	// Multiple lines (different timestamps, skips freshness check due to missing level *and PID*)
	{
		[]state.LogLine{{
			CollectedAt: now.Add(- 5 * time.Second),
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			Content: "first\n",
		},
		{
			CollectedAt: now,
			Content: "second\n",
		}},
		state.LogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt: now.Add(- 5 * time.Second),
				LogLevel: pganalyze_collector.LogLineInformation_LOG,
				ByteEnd: 13,
			}},
		},
		"first\nsecond\n",
		[]state.LogLine{},
		nil,
	},
	// Multiple lines (different timestamps, requiring skip of freshness check due to log line number)
	{
		[]state.LogLine{{
			CollectedAt: now,
			LogLineNumber: 2,
			BackendPid: 42,
			Content: "second\n",
		},
		{
			CollectedAt: now.Add(- 5 * time.Second),
			LogLevel: pganalyze_collector.LogLineInformation_LOG,
			LogLineNumber: 1,
			BackendPid: 42,
			Content: "first\n",
		}},
		state.LogState{},
		state.LogFile{
			LogLines: []state.LogLine{{
				CollectedAt: now.Add(- 5 * time.Second),
				LogLevel: pganalyze_collector.LogLineInformation_LOG,
				LogLineNumber: 1,
				ByteEnd: 13,
				BackendPid: 42,
			}},
		},
		"first\nsecond\n",
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
		logState, logFile, tooFreshLogLines, err := stream.AnalyzeStreamInGroups(pair.logLines)
		logFileContent := ""
		if logFile.TmpFile != nil {
			dat, err := ioutil.ReadFile(logFile.TmpFile.Name())
			if err != nil {
				t.Errorf("Error reading temporary log file: %s", err)
			}
			logFileContent = string(dat)
		}

		logState.CollectedAt = time.Time{} // Avoid comparing against time.Now()
		logFile.TmpFile = nil // Avoid comparing against tempfile
		logFile.UUID = uuid.UUID{} // Avoid comparing against a generated UUID

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		if diff := cfg.Compare(pair.logState, logState); diff != "" {
			t.Errorf("For %v: log state diff: (-want +got)\n%s", pair.logState, diff)
		}
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
