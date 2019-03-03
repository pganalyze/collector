package state

// Known PostgresVersion values - use these for checks in version-dependent code
const (
	PostgresVersion92 = 90200
	PostgresVersion93 = 90300
	PostgresVersion94 = 90400
	PostgresVersion95 = 90500
	PostgresVersion96 = 90600
	PostgresVersion10 = 100000
	PostgresVersion11 = 110000
	PostgresVersion12 = 120000

	// MinRequiredPostgresVersion - We require PostgreSQL 9.2 or newer, since pg_stat_statements only started being usable then
	MinRequiredPostgresVersion = PostgresVersion92
)

// PostgresVersion - Identifying information about the PostgreSQL server version and build details
type PostgresVersion struct {
	Full    string `json:"full"`    // e.g. "PostgreSQL 9.5.1 on x86_64-pc-linux-gnu, compiled by gcc (Debian 4.9.2-10) 4.9.2, 64-bit"
	Short   string `json:"short"`   // e.g. "9.5.1"
	Numeric int    `json:"numeric"` // e.g. 90501

	// For collector use only, to avoid calling functions that don't work in AWS Aurora
	IsAwsAurora bool
}
