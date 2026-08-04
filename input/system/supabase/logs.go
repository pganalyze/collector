package supabase

import (
	"strconv"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	otlpLogs "go.opentelemetry.io/proto/otlp/logs/v1"
)

// Parsing for Supabase's log drain, which delivers Postgres logs over OTLP through
// its Logflare-backed pipeline.

// ParsedFields returns the Postgres "parsed" csvlog fields from a Supabase log drain
// record, or nil if this is not a Postgres log record. Supabase's Supavisor pooler
// application logs share the envelope but carry no "parsed" object.
func ParsedFields(body *common.KeyValueList) *common.KeyValueList {
	for _, v := range body.Values {
		if v.Key != "metadata" {
			continue
		}
		for _, m := range v.Value.GetKvlistValue().GetValues() {
			if m.Key == "parsed" {
				return m.Value.GetKvlistValue()
			}
		}
	}
	return nil
}

// LogLineFrom maps a Supabase log drain Postgres record to a LogLine. The message -
// including the collector identify marker - comes from the record's EventName (the
// drain's event_message), not from "parsed", whose message/query fields are usually
// empty. The timestamp comes from the record's TimeUnixNano.
func LogLineFrom(l *otlpLogs.LogRecord, parsed *common.KeyValueList) state.LogLine {
	logLine := state.LogLine{
		Content:    l.EventName,
		OccurredAt: time.Unix(0, int64(l.TimeUnixNano)),
	}
	for _, rv := range parsed.Values {
		switch rv.Key {
		case "user_name":
			logLine.Username = rv.Value.GetStringValue()
		case "database_name":
			logLine.Database = rv.Value.GetStringValue()
		case "application_name":
			logLine.Application = rv.Value.GetStringValue()
		case "process_id":
			logLine.BackendPid = int32(anyValueInt(rv.Value))
		case "session_line_num":
			logLine.LogLineNumber = int32(anyValueInt(rv.Value))
		case "error_severity":
			logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[rv.Value.GetStringValue()])
		}
	}
	return logLine
}

// anyValueInt reads an integer from an OTel value that may be encoded either as an int
// (the drain sends process_id and session_line_num as ints) or as a numeric string.
func anyValueInt(v *common.AnyValue) int64 {
	if s := v.GetStringValue(); s != "" {
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	return v.GetIntValue()
}
