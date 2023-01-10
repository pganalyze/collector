package logs_test

import (
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
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
		output:          "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: XXXXXXXX\n",
	},
	{
		filterLogSecret: "statement_text",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: SELECT 1\n",
		output:          "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: XXXXXXXX\n",
	},
	{
		filterLogSecret: "statement_parameter",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: SELECT 1\n",
		output:          "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 1242.570 ms  statement: SELECT 1\n",
	},
	{
		filterLogSecret: "none",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  division by zero\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  Unknown Data\n",
		output:          "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  division by zero\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  Unknown Data\n",
	},
	{
		filterLogSecret: "unidentified",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  division by zero\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  Unknown Data\n",
		output:          "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  division by zero\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:ERROR:  XXXXXXXXXXXXX",
	},
	{
		filterLogSecret: "none",
		input:           "Unknown Data\n",
		output:          "Unknown Data\n",
	},
	{
		filterLogSecret: "unidentified",
		input:           "Unknown Data\n",
		output:          "XXXXXXXXXXXXX",
	},
	{
		filterLogSecret: "statement_text, statement_parameter, unidentified",
		input:           "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 4079.697 ms  execute <unnamed>: \nSELECT * FROM x WHERE y = $1 LIMIT $2\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:DETAIL:  parameters: $1 = 'long string', $2 = '1'\n",
		output:          "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  duration: 4079.697 ms  execute <unnamed>: \nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\n2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:DETAIL:  parameters: $1 = 'XXXXXXXXXXX', $2 = 'X'\n",
	},
}

func TestReplaceSecrets(t *testing.T) {
	for _, pair := range replaceTests {
		logLines, _, _ := logs.ParseAndAnalyzeBuffer(strings.NewReader(pair.input), 0, time.Time{}, &state.Server{})
		output := logs.ReplaceSecrets([]byte(pair.input), logLines, state.ParseFilterLogSecret(pair.filterLogSecret))

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true

		if diff := cfg.Compare(pair.output, string(output)); diff != "" {
			t.Errorf("For filter \"%s\", text:\n%vdiff: (-want +got)\n%s", pair.filterLogSecret, pair.input, diff)
		}
	}
}
