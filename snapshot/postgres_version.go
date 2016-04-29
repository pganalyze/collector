//go:generate msgp

package snapshot

type PostgresVersion struct {
	Full    string `msg:"full"`
	Short   string `msg:"short"`
	Numeric int    `msg:"numeric"`
}

const (
	PostgresVersion92 = 90200
	PostgresVersion93 = 90300
	PostgresVersion94 = 90400
	PostgresVersion95 = 90500
	PostgresVersion96 = 90600

	// MinRequiredPostgresVersion - We require PostgreSQL 9.2 or newer, since pg_stat_statements only started being usable then
	MinRequiredPostgresVersion = PostgresVersion92
)
