package config

import (
	"cmp"
	"strings"
)

// Detection of managed Postgres platforms from connection details rather
// than relying on the presence of a specific environment variable.

func isNeonHost(host string) bool {
	return strings.HasSuffix(host, ".neon.tech")
}

func isSupabaseHost(host string) bool {
	return strings.HasSuffix(host, ".supabase.co") || strings.HasSuffix(host, ".pooler.supabase.com")
}

// Supabase connects either directly (db.<project-ref>.supabase.co) or through the
// Supavisor pooler (<region>.pooler.supabase.com).
func supabaseSystemID(config ServerConfig) string {
	host := config.GetDbHost()

	var ref string
	if strings.HasSuffix(host, ".pooler.supabase.com") {
		// Shared pooler host: the ref is carried in the username as <role>.<ref>.
		username := config.GetDbUsername()
		if idx := strings.LastIndexByte(username, '.'); idx >= 0 {
			ref = username[idx+1:]
		}
	} else {
		// Direct/pgbouncer host: db.<ref>.supabase.co (or the bare <ref>.supabase.co).
		trimmed := strings.TrimSuffix(host, ".supabase.co")
		if idx := strings.LastIndexByte(trimmed, '.'); idx >= 0 {
			ref = trimmed[idx+1:]
		} else {
			ref = trimmed
		}
	}

	return cmp.Or(ref, host)
}

// extractSupabaseUsername returns the Postgres role from a Supabase username. On
// the Supavisor pooler the username carries the project ref as <role>.<project-ref>
// for tenant routing, but we need just the role name part. Direct connections have no
// suffix and are returned unchanged.
func extractSupabaseUsername(username string) string {
	if idx := strings.LastIndexByte(username, '.'); idx > 0 {
		return username[:idx]
	}
	return username
}
