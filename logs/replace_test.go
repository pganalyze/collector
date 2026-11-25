package logs_test

import (
	"bufio"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type replaceTestpair struct {
	filterLogSecret string
	input           string
	output          string
}

var replaceTests = []replaceTestpair{
	{
		filterLogSecret: "all",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: SELECT 1\n",
		output:          "duration: 1242.570 ms  statement: [redacted]\n",
	},
	{
		filterLogSecret: "statement_text",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: SELECT 1\n",
		output:          "duration: 1242.570 ms  statement: [redacted]\n",
	},
	{
		filterLogSecret: "statement_text",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 2007.111 ms  plan:\n{\"Query Text\": \"SELECT pg_sleep($1)\", \"Query Parameters\": \"$1 = '2'\", \"Plan\": { } }\n",
		output:          "duration: 2007.111 ms  plan:\n[redacted]\n",
	},
	{
		filterLogSecret: "statement_text",
		input:           "2025-12-03 07:31:44 UTC:1.1.1.1(123):myuser@mydb:[123]:ERROR:  canceling statement due to statement timeout\n2025-12-03 07:31:44 UTC:1.1.1.1(123):myuser@mydb:[123]:STATEMENT:  SELECT 1\n",
		output:          "canceling statement due to statement timeout\n[redacted]",
	},
	{
		filterLogSecret: "statement_parameter",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 4079.697 ms  execute <unnamed>: \nSELECT * FROM x WHERE y = $1 LIMIT $2\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:DETAIL:  parameters: $1 = 'long string', $2 = '1', $3 = 'long string'\n",
		output:          "duration: 4079.697 ms  execute <unnamed>: \nSELECT * FROM x WHERE y = $1 LIMIT $2\nparameters: $1 = '[redacted]', $2 = '[redacted]', $3 = '[redacted]'\n",
	},
	{
		filterLogSecret: "statement_parameter",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 2007.111 ms  plan:\n{\"Query Text\": \"SELECT * FROM x WHERE y = $1 LIMIT $2\", \"Plan\": { } }\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:CONTEXT: unnamed portal with parameters: $1 = 'long string', $2 = '1', $3 = 'long string'\n",
		output:          "duration: 2007.111 ms  plan:\n{\"Query Text\": \"SELECT * FROM x WHERE y = $1 LIMIT $2\", \"Plan\": { } }\nunnamed portal with parameters: $1 = '[redacted]', $2 = '[redacted]', $3 = '[redacted]'\n",
	},
	{
		filterLogSecret: "none",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  division by zero\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  Unknown Data\n",
		output:          "division by zero\nUnknown Data\n",
	},
	{
		filterLogSecret: "unidentified",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  division by zero\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  Unknown Data\n",
		output:          "division by zero\n[redacted]\n",
	},
	{
		filterLogSecret: "statement_parameter, unidentified",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 2007.111 ms  plan:\n{\"Query Text\": \"SELECT * FROM x WHERE y = $1 LIMIT $2\", \"Plan\": { } }\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:CONTEXT: some new context we do not know with a secret\n",
		output:          "duration: 2007.111 ms  plan:\n{\"Query Text\": \"SELECT * FROM x WHERE y = $1 LIMIT $2\", \"Plan\": { } }\n[redacted]\n",
	},
	{
		filterLogSecret: "statement_text, statement_parameter, unidentified",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 4079.697 ms  execute <unnamed>: \nSELECT * FROM x WHERE y = $1 LIMIT $2\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:DETAIL:  parameters: $1 = 'long string', $2 = '1'\n",
		output:          "duration: 4079.697 ms  execute <unnamed>: \n[redacted]\nparameters: $1 = '[redacted]', $2 = '[redacted]'\n",
	},
	// Multi-line content where a secret marker ends exactly where the next one
	// (gap or remainder) starts - all of it must be redacted, with adjacent
	// markers merged into a single replacement
	{
		filterLogSecret: "all",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  statement: INSERT INTO accounts(name, plan)\n\tVALUES ('alice', 'pro');\n",
		output:          "statement: [redacted]\n",
	},
	{
		filterLogSecret: "all",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  statement: EXECUTE p_ins('bob');\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:DETAIL:  prepare: PREPARE p_ins(text) AS INSERT INTO accounts(name, plan)\n\tVALUES ($1, 'free');\n",
		output:          "statement: [redacted]\nprepare: [redacted]\n",
	},
	// The continuation lines of a multi-line statement are statement text too, so
	// filtering only statement_text must redact them, not just "all"
	{
		filterLogSecret: "statement_text",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  statement: INSERT INTO accounts(name, plan)\n\tVALUES ('alice', 'pro');\n",
		output:          "statement: [redacted]\n",
	},
	{
		filterLogSecret: "statement_text",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  statement: EXECUTE p_ins('bob');\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:DETAIL:  prepare: PREPARE p_ins(text) AS INSERT INTO accounts(name, plan)\n\tVALUES ($1, 'free');\n",
		output:          "statement: [redacted]\nprepare: [redacted]\n",
	},
	{
		filterLogSecret: "statement_text",
		input:           "2018-03-11 20:00:02 UTC::@:[1]:LOG:  server process (PID 123) was terminated by signal 11: Segmentation fault\n2018-03-11 20:00:02 UTC::@:[1]:DETAIL:  Failed process was running: SELECT * FROM accounts\n\tWHERE name = 'alice';\n",
		output:          "server process (PID 123) was terminated by signal 11: Segmentation fault\nFailed process was running: [redacted]\n",
	},
}

func TestReplaceSecrets(t *testing.T) {
	for _, pair := range replaceTests {
		reader := bufio.NewReader(strings.NewReader(pair.input))
		server := state.MakeServer(config.ServerConfig{}, false)
		server.LogParser = logs.NewLogParser(logs.LogPrefixAmazonRds, nil, false)
		logLines, _ := logs.ParseAndAnalyzeBuffer(reader, time.Time{}, server, state.CollectionOpts{}, &util.Logger{})
		logs.ReplaceSecrets(logLines, state.ParseFilterLogSecret(pair.filterLogSecret))

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		output := ""
		for _, logLine := range logLines {
			output += logLine.Content
		}
		if diff := cfg.Compare(pair.output, output); diff != "" {
			t.Errorf("For filter \"%s\", text:\n%vdiff: (-want +got)\n%s", pair.filterLogSecret, pair.input, diff)
		}
	}
}

// Structured log sources (OTel jsonlog, Supabase, Tembo) deliver the message and
// detail fields as-is, with bare newlines (no tab prefix) and no trailing newline
func TestReplaceSecretsStructured(t *testing.T) {
	makeLine := func(level pganalyze_collector.LogLineInformation_LogLevel, content string) state.LogLine {
		return state.LogLine{UUID: uuid.New(), LogLevel: level, BackendPid: 1, Content: content}
	}
	tests := []struct {
		in   []state.LogLine
		want []string
	}{
		{
			in:   []state.LogLine{makeLine(pganalyze_collector.LogLineInformation_LOG, "statement: INSERT INTO accounts(name, plan)\nVALUES ('alice', 'pro');")},
			want: []string{"statement: [redacted]"},
		},
		{
			in: []state.LogLine{
				makeLine(pganalyze_collector.LogLineInformation_ERROR, "deadlock detected"),
				makeLine(pganalyze_collector.LogLineInformation_DETAIL, "Process 1 waits for ShareLock on transaction 2; blocked by process 2.\nProcess 2 waits for ShareLock on transaction 1; blocked by process 1.\nProcess 1: UPDATE accounts\nSET name = 'alice' WHERE id = 1;\nProcess 2: UPDATE accounts\nSET name = 'bob' WHERE id = 2;"),
			},
			// Redacting a statement's continuation lines must keep the newline that
			// separates it from the next "Process N:" entry
			want: []string{
				"deadlock detected",
				"Process 1 waits for ShareLock on transaction 2; blocked by process 2.\nProcess 2 waits for ShareLock on transaction 1; blocked by process 1.\nProcess 1: [redacted]\nProcess 2: [redacted]",
			},
		},
	}
	// Every secret in these inputs is statement text, so filtering only
	// statement_text must redact the same as "all"
	for _, filter := range []string{"all", "statement_text"} {
		for _, tt := range tests {
			analyzed, _ := logs.AnalyzeLogLines(tt.in)
			logs.ReplaceSecrets(analyzed, state.ParseFilterLogSecret(filter))
			got := make([]string, len(analyzed))
			for i, l := range analyzed {
				got[i] = l.Content
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("For filter %q:\ngot  %q\nwant %q", filter, got, tt.want)
			}
		}
	}
}
