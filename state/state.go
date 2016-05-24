package state

type State struct {
	Backends   []PostgresBackend
	Statements []PostgresStatement
	Relations  []PostgresRelation
	Settings   []PostgresSetting
	Functions  []PostgresFunction
	Version    PostgresVersion
	System     *SystemState
	Logs       []LogLine
	Explains   []PostgresExplain
}
