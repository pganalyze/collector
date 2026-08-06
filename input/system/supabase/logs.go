package supabase

import (
	common "go.opentelemetry.io/proto/otlp/common/v1"
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
