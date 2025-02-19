package logs_test

import (
	"bufio"
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
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
}

func TestReplaceSecrets(t *testing.T) {
	for _, pair := range replaceTests {
		reader := bufio.NewReader(strings.NewReader(pair.input))
		server := state.MakeServer(config.ServerConfig{}, false)
		server.LogParser = logs.NewLogParser(logs.LogPrefixAmazonRds, nil)
		logLines, _ := logs.ParseAndAnalyzeBuffer(reader, time.Time{}, server)
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
