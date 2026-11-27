package state

import (
	"strconv"
	"strings"
)

// Known PostgresVersion values - use these for checks in version-dependent code
const (
	PostgresVersion12 = 120000
	PostgresVersion13 = 130000
	PostgresVersion14 = 140000
	PostgresVersion15 = 150000
	PostgresVersion16 = 160000
	PostgresVersion17 = 170000
	PostgresVersion18 = 180000

	// MinRequiredPostgresVersion - We require PostgreSQL 12 or newer
	MinRequiredPostgresVersion = PostgresVersion12
)

// PostgresVersion - Identifying information about the PostgreSQL server version and build details
type PostgresVersion struct {
	Full    string `json:"full"`    // e.g. "PostgreSQL 9.5.1 on x86_64-pc-linux-gnu, compiled by gcc (Debian 4.9.2-10) 4.9.2, 64-bit"
	Short   string `json:"short"`   // e.g. "9.5.1"
	Numeric int    `json:"numeric"` // e.g. 90501

	// For collector use only, to avoid calling functions that don't work
	IsAwsAurora   bool   // Amazon Aurora
	AuroraVersion string // Amazon Aurora version including patch level, e.g. "18.6.2" (empty if not Aurora)
	IsAlloyDB     bool   // Google AlloyDB for PostgreSQL
	IsEPAS        bool   // EnterpriseDB Advanced Server
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

// Amazon Aurora patch releases that backport the November 2026 security fixes onto an
// older community minor release (server_version still reports the older minor release),
// keyed by Aurora major/minor version, with the oldest patch version that has the fixes.
//
// See https://docs.aws.amazon.com/AmazonRDS/latest/AuroraPostgreSQLReleaseNotes/AuroraPostgreSQL.Updates.html
// (patch releases from November 26, 2026)
var nov2026AuroraMinimumPatchVersions = map[[2]int]int{
	{14, 24}: 2,
	{15, 19}: 2,
	{16, 15}: 2,
	{17, 11}: 2,
	{18, 6}:  2,
}

const Nov2026AuroraMinimumVersionsHuman = "18.6.2, 17.11.2, 16.15.2, 15.19.2 or 14.24.2"

// Nov2026MinimumVersionsHumanFor - Human-readable list of the minimum versions that include
// the November 2026 security fixes, including Aurora patch versions for Aurora servers
func (v PostgresVersion) Nov2026MinimumVersionsHumanFor() string {
	if v.IsAwsAurora {
		return Nov2026MinimumVersionsHuman + " (Aurora: " + Nov2026AuroraMinimumVersionsHuman + ")"
	}
	return Nov2026MinimumVersionsHuman
}

// HasNov2026SecurityFixes - Whether this server includes the November 2026 security fixes,
// required for Workbooks EXPLAIN ANALYZE runs unless workbooks_explain_analyze is set to
// "allow_no_version_check"
func (v PostgresVersion) HasNov2026SecurityFixes() bool {
	major := v.Numeric / 10000
	minor := v.Numeric % 10000

	if v.IsAwsAurora && auroraHasNov2026SecurityFixes(v.AuroraVersion) {
		return true
	}

	if major > 18 {
		return true // newer major versions include the fixes
	}
	if major < 14 {
		return false // no longer supported, did not receive the fixes
	}
	return minor >= nov2026MinimumMinorVersions[major]
}

func auroraHasNov2026SecurityFixes(auroraVersion string) bool {
	parts := strings.Split(strings.TrimSpace(auroraVersion), ".")
	if len(parts) != 3 {
		return false
	}
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return false
		}
		nums[i] = n
	}
	minPatch, ok := nov2026AuroraMinimumPatchVersions[[2]int{nums[0], nums[1]}]
	return ok && nums[2] >= minPatch
}
