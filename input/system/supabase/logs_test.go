package supabase

import (
	"testing"

	common "go.opentelemetry.io/proto/otlp/common/v1"
)

func strVal(s string) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: s}}
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
