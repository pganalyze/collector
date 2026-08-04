package config

import "strings"

// Detection of managed Postgres platforms from connection details rather
// than relying on the presence of a specific environment variable.

func isNeonHost(host string) bool {
	return strings.HasSuffix(host, ".neon.tech")
}

func isSupabaseHost(host string) bool {
	return strings.HasSuffix(host, ".supabase.co") || strings.HasSuffix(host, ".pooler.supabase.com")
}

// Supabase connects either directly (db.<project-ref>.supabase.co) or through the
// Supavisor pooler (<region>.pooler.supabase.com). Pooler hosts are shared across
// projects, with the project ref carried in the username as <role>.<project-ref>,
// so we key on the project ref to identify the system consistently either way.
func supabaseSystemID(config ServerConfig) string {
	host := config.GetDbHost()
	if strings.HasSuffix(host, ".pooler.supabase.com") {
		if _, ref, found := strings.Cut(config.GetDbUsername(), "."); found && ref != "" {
			return ref
		}
		return host
	}
	ref := strings.TrimPrefix(strings.TrimSuffix(host, ".supabase.co"), "db.")
	if ref != "" && !strings.Contains(ref, ".") {
		return ref
	}
	return host
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
