package state

// Known PostgresVersion values - use these for checks in version-dependent code
const (
	PostgresVersion10 = 100000
	PostgresVersion11 = 110000
	PostgresVersion12 = 120000
	PostgresVersion13 = 130000
	PostgresVersion14 = 140000
	PostgresVersion15 = 150000
	PostgresVersion16 = 160000
	PostgresVersion17 = 170000
	PostgresVersion18 = 180000

	// MinRequiredPostgresVersion - We require PostgreSQL 10 or newer
	MinRequiredPostgresVersion = PostgresVersion10
)

// PostgresVersion - Identifying information about the PostgreSQL server version and build details
type PostgresVersion struct {
	Full    string `json:"full"`    // e.g. "PostgreSQL 9.5.1 on x86_64-pc-linux-gnu, compiled by gcc (Debian 4.9.2-10) 4.9.2, 64-bit"
	Short   string `json:"short"`   // e.g. "9.5.1"
	Numeric int    `json:"numeric"` // e.g. 90501

	// For collector use only, to avoid calling functions that don't work
	IsAwsAurora bool // Amazon Aurora
	IsAlloyDB   bool // Google AlloyDB for PostgreSQL
	IsEPAS      bool // EnterpriseDB Advanced Server
}

// Oldest minor release per major version from the November 2026 security release
var nov2026MinimumMinorVersions = map[int]int{
	14: 25,
	15: 20,
	16: 16,
	17: 12,
	18: 7,
}

const Nov2026MinimumVersionsHuman = "18.7, 17.12, 16.16, 15.20 or 14.25"

// HasNov2026SecurityFixes - Whether this server includes the November 2026 security fixes,
// required for Workbooks EXPLAIN ANALYZE runs unless workbooks_explain_analyze is set to
// "allow_no_version_check"
func (v PostgresVersion) HasNov2026SecurityFixes() bool {
	major := v.Numeric / 10000
	minor := v.Numeric % 10000

	if major > 18 {
		return true // newer major versions include the fixes
	}
	if major < 14 {
		return false // no longer supported, did not receive the fixes
	}
	return minor >= nov2026MinimumMinorVersions[major]
}
