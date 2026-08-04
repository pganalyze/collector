package supabase

import (
	"testing"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	otlpLogs "go.opentelemetry.io/proto/otlp/logs/v1"
)

func strVal(s string) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: s}}
}

func intVal(n int64) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_IntValue{IntValue: n}}
}

func kv(key string, v *common.AnyValue) *common.KeyValue {
	return &common.KeyValue{Key: key, Value: v}
}

func kvList(kvs ...*common.KeyValue) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_KvlistValue{KvlistValue: &common.KeyValueList{Values: kvs}}}
}

func body(kvs ...*common.KeyValue) *common.KeyValueList {
	return &common.KeyValueList{Values: kvs}
}

func TestParsedFields(t *testing.T) {
	tests := []struct {
		name    string
		body    *common.KeyValueList
		wantNil bool
	}{
		{
			name: "postgres log record has metadata.parsed",
			body: body(
				kv("id", strVal("abc")),
				kv("metadata", kvList(kv("host", strVal("db-x")), kv("parsed", kvList(kv("error_severity", strVal("LOG")))))),
			),
			wantNil: false,
		},
		{
			name:    "supavisor app log has metadata but no parsed",
			body:    body(kv("metadata", kvList(kv("app_name", strVal("x")), kv("db_name", strVal("postgres"))))),
			wantNil: true,
		},
		{
			name:    "record without metadata",
			body:    body(kv("id", strVal("abc"))),
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsedFields(tt.body); (got == nil) != tt.wantNil {
				t.Errorf("ParsedFields() == nil = %v, want %v", got == nil, tt.wantNil)
			}
		})
	}
}

func TestLogLineFrom(t *testing.T) {
	l := &otlpLogs.LogRecord{
		TimeUnixNano: 1785849893220595000,
		EventName:    "duration: 3003.075 ms  plan: {...}",
	}
	parsed := body(
		kv("application_name", strVal("Supavisor")),
		kv("error_severity", strVal("LOG")),
		kv("user_name", strVal("pganalyze")),
		kv("database_name", strVal("postgres")),
		kv("process_id", intVal(60649)),
		kv("session_line_num", intVal(12)),
	)

	got := LogLineFrom(l, parsed)

	if got.Content != l.EventName {
		t.Errorf("Content = %q, want the record's EventName", got.Content)
	}
	if got.Application != "Supavisor" {
		t.Errorf("Application = %q", got.Application)
	}
	if got.Username != "pganalyze" {
		t.Errorf("Username = %q", got.Username)
	}
	if got.Database != "postgres" {
		t.Errorf("Database = %q", got.Database)
	}
	if got.BackendPid != 60649 {
		t.Errorf("BackendPid = %d", got.BackendPid)
	}
	if got.LogLineNumber != 12 {
		t.Errorf("LogLineNumber = %d", got.LogLineNumber)
	}
	if got.LogLevel != pganalyze_collector.LogLineInformation_LOG {
		t.Errorf("LogLevel = %v", got.LogLevel)
	}
	if got.OccurredAt.IsZero() {
		t.Error("OccurredAt should be set from TimeUnixNano")
	}
}

func TestLogLineFromStringEncodedInts(t *testing.T) {
	// Some producers send process_id / session_line_num as numeric strings rather
	// than ints; anyValueInt must handle both.
	parsed := body(
		kv("process_id", strVal("777")),
		kv("session_line_num", strVal("3")),
	)

	got := LogLineFrom(&otlpLogs.LogRecord{}, parsed)

	if got.BackendPid != 777 {
		t.Errorf("BackendPid = %d, want 777", got.BackendPid)
	}
	if got.LogLineNumber != 3 {
		t.Errorf("LogLineNumber = %d, want 3", got.LogLineNumber)
	}
}
