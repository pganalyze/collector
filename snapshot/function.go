//go:generate msgp

package snapshot

type Function struct {
	SchemaName      string         `msg:"schema_name"`
	FunctionName    string         `msg:"function_name"`
	Language        string         `msg:"language"`
	Source          string         `msg:"source"`
	SourceBin       NullableString `msg:"source_bin"`
	Config          NullableString `msg:"config"`
	Arguments       NullableString `msg:"arguments"`
	Result          NullableString `msg:"result"`
	Aggregate       bool           `msg:"aggregate"`
	Window          bool           `msg:"window"`
	SecurityDefiner bool           `msg:"security_definer"`
	Leakproof       bool           `msg:"leakproof"`
	Strict          bool           `msg:"strict"`
	ReturnsSet      bool           `msg:"returns_set"`
	Volatile        NullableString `msg:"volatile"`
	Calls           NullableInt    `msg:"calls"`
	TotalTime       NullableFloat  `msg:"total_time"`
	SelfTime        NullableFloat  `msg:"self_time"`
}
