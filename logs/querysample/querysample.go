package querysample

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/logs/util"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	cUtil "github.com/pganalyze/collector/util"
)

func TransformAutoExplainToQuerySample(logLine state.LogLine, explainText string, queryRuntime string) (state.PostgresQuerySample, error) {
	queryRuntimeMs, _ := strconv.ParseFloat(queryRuntime, 64)
	if strings.HasPrefix(explainText, "{") { // json format
		if util.WasTruncated(explainText) {
			return state.PostgresQuerySample{}, fmt.Errorf("auto_explain output was truncated and can't be parsed as JSON")
		} else {
			return transformExplainJSONToQuerySample(logLine, explainText, queryRuntimeMs)
		}
	} else if strings.HasPrefix(explainText, "Query Text:") { // text format
		return transformExplainTextToQuerySample(logLine, explainText, queryRuntimeMs)
	} else {
		return state.PostgresQuerySample{}, fmt.Errorf("unsupported auto_explain format")
	}
}

func transformExplainJSONToQuerySample(logLine state.LogLine, explainText string, queryRuntimeMs float64) (state.PostgresQuerySample, error) {
	var explainJSONOutput state.ExplainPlanContainer

	if err := json.Unmarshal([]byte(explainText), &explainJSONOutput); err != nil {
		return state.PostgresQuerySample{}, err
	}

	// Remove query text from EXPLAIN itself, to avoid duplication and match EXPLAIN (FORMAT JSON)
	sampleQueryText := strings.TrimSpace(explainJSONOutput.QueryText)
	explainJSONOutput.QueryText = ""

	return state.PostgresQuerySample{
		Query:             sampleQueryText,
		RuntimeMs:         queryRuntimeMs,
		OccurredAt:        logLine.OccurredAt,
		Username:          logLine.Username,
		Database:          logLine.Database,
		LogLineUUID:       logLine.UUID,
		HasExplain:        true,
		ExplainSource:     pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
		ExplainFormat:     pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT,
		ExplainOutputJSON: &explainJSONOutput,
		Parameters:        findQueryParameters(explainJSONOutput.QueryParameters),
	}, nil
}

var autoExplainTextPlanDetailsRegexp = regexp.MustCompile(`^Query Text: (.+)\s+([\s\S]+)`)
var herokuAutoExplainWithTabRegexp = regexp.MustCompile(`^Query Text: ([^\t]+)\t([\s\S]+)`)
var autoExplainTextWithQueryParametersRegexp = regexp.MustCompile(`^Query Text: ([\s\S]+)\r?\n\s*Query Parameters: (.+)\r?\n\s*([\s\S]+)`)
var autoExplainTextWithCostsRegexp = regexp.MustCompile(`^Query Text: ([\s\S]+?)\r?\n\s*([\S ]+  \(cost=\d+\.\d{2}\.\.\d+\.\d{2} rows=\d+ width=\d+\)[\s\S]+)`)

func transformExplainTextToQuerySample(logLine state.LogLine, explainText string, queryRuntimeMs float64) (state.PostgresQuerySample, error) {
	querySample := state.PostgresQuerySample{
		RuntimeMs:     queryRuntimeMs,
		OccurredAt:    logLine.OccurredAt,
		Username:      logLine.Username,
		Database:      logLine.Database,
		LogLineUUID:   logLine.UUID,
		HasExplain:    true,
		ExplainSource: pganalyze_collector.QuerySample_AUTO_EXPLAIN_EXPLAIN_SOURCE,
		ExplainFormat: pganalyze_collector.QuerySample_TEXT_EXPLAIN_FORMAT,
	}
	withParametersParts := autoExplainTextWithQueryParametersRegexp.FindStringSubmatch(explainText)
	if len(withParametersParts) == 4 {
		querySample.Parameters = findQueryParameters(withParametersParts[2])
		querySample.Query = withParametersParts[1]
		querySample.ExplainOutputText = withParametersParts[3]
	} else {
		explainParts := autoExplainTextWithCostsRegexp.FindStringSubmatch(explainText)
		if len(explainParts) != 3 {
			// Fallback to the old way (not supporting new lines in Query Text, but does support EXPLAIN without costs)
			explainParts = autoExplainTextPlanDetailsRegexp.FindStringSubmatch(explainText)
			if len(explainParts) != 3 {
				return state.PostgresQuerySample{}, fmt.Errorf("auto_explain output doesn't match expected format")
			}
		}

		// If EXPLAIN output's first char is not a capital letter (e.g. not something like "Update on" or "Index Scan"),
		// likely it's hitting the Heroku's newline break in "Query Text:" chunk
		// Handle the separation of the query and the explain output text with the tab for these cases
		// (this can be retired with the autoExplainTextWithCostsRegexp, but leaving here for the EXPLAIN without costs case)
		explainOutputFirstChar := explainParts[2][0]
		if cUtil.IsHeroku() && !(explainOutputFirstChar >= 'A' && explainOutputFirstChar <= 'Z') {
			if parts := herokuAutoExplainWithTabRegexp.FindStringSubmatch(explainText); len(parts) == 3 {
				explainParts = parts
			}
		}
		querySample.Query = explainParts[1]
		querySample.ExplainOutputText = explainParts[2]
	}
	return querySample, nil
}

func TransformLogMinDurationStatementToQuerySample(logLine state.LogLine, queryText string, queryRuntime string, queryProtocolStep string, parameterParts [][]string) (s state.PostgresQuerySample, ok bool) {
	// Ignore bind/parse steps of extended query protocol, since they are not the actual execution
	// See https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-EXT-QUERY
	if queryProtocolStep == "bind" || queryProtocolStep == "parse" {
		return state.PostgresQuerySample{}, false
	}

	queryText = strings.TrimSpace(queryText)
	if queryText == "" {
		return state.PostgresQuerySample{}, false
	}

	sample := state.PostgresQuerySample{
		Query:       queryText,
		OccurredAt:  logLine.OccurredAt,
		Username:    logLine.Username,
		Database:    logLine.Database,
		LogLineUUID: logLine.UUID,
	}
	sample.RuntimeMs, _ = strconv.ParseFloat(queryRuntime, 64)
	for _, part := range parameterParts {
		if len(part) == 3 {
			if part[1] == "NULL" {
				sample.Parameters = append(sample.Parameters, null.NewString("", false))
			} else {
				sample.Parameters = append(sample.Parameters, null.StringFrom(part[2]))
			}
		}
	}
	return sample, true
}

func findQueryParameters(paramText string) []null.String {
	// Handle Query Parameters (available from Postgres 16+)
	var parameters []null.String
	// Regular expression to find all values in single quotes or NULL
	// Query Parameters example: $1 = 'foo', $2 = '123', $3 = NULL, $4 = 'bo''o'
	re := regexp.MustCompile(`'((?:[^']|'')*)'|NULL`)
	for _, part := range re.FindAllString(paramText, -1) {
		if part == "NULL" {
			parameters = append(parameters, null.NewString("", false))
		} else {
			parameters = append(parameters, null.StringFrom(strings.Trim(part, "'")))
		}
	}
	return parameters
}
