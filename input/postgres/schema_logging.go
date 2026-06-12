package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pganalyze/collector/util"
)

// Schema collection debug logging
//
// When enabled, the helpers in this file emit additional diagnostics for each
// schema query: how long each query took, how many objects each step returned,
// a snapshot of Go memory usage after each step, and (optionally) the exact SQL
// that was run. This is intended for diagnosing slow or memory-intensive schema
// collection on servers with large or unusual catalogs.
//
// This logging is gated on its own environment variable (LOG_SCHEMA_DEBUG)
// rather than on the --verbose/--very-verbose flags. This keeps it independent
// of unrelated verbose logging (e.g. log insights), which would otherwise be
// enabled at the same time and drown out the schema diagnostics. Output is
// emitted at INFO level so it appears whenever the environment variable is set,
// without requiring any verbose flag. The helpers are no-ops (with no
// measurement overhead beyond the query itself) when the variable is unset.

const schemaDebugLogPrefix = "[schema-debug]"

const (
	// logSchemaDebugEnvVar is the master switch for schema collection debug
	// logging (query runtimes, per-step object counts, and Go memory usage).
	logSchemaDebugEnvVar = "LOG_SCHEMA_DEBUG"

	// logSchemaSQLEnvVar additionally logs the SQL text of each schema query.
	// Logging the full SQL on every query in every database is a lot of output,
	// so it is opt-in beyond LOG_SCHEMA_DEBUG and only takes effect when schema
	// debug logging is already enabled.
	logSchemaSQLEnvVar = "LOG_SCHEMA_SQL"
)

// humanizeBytes formats a byte count as a human-readable string (e.g. "43.2 MB")
// so memory snapshots are easier to eyeball than raw byte counts.
func humanizeBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func envFlagEnabled(name string) bool {
	switch strings.ToLower(os.Getenv(name)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// SchemaDebugEnabled reports whether schema collection debug logging has been
// enabled via the LOG_SCHEMA_DEBUG environment variable.
func SchemaDebugEnabled() bool {
	return envFlagEnabled(logSchemaDebugEnvVar)
}

// logSchemaSQLEnabled reports whether the SQL text of each schema query should
// also be logged. Only meaningful when SchemaDebugEnabled() is true.
func logSchemaSQLEnabled() bool {
	return envFlagEnabled(logSchemaSQLEnvVar)
}

// loggedSchemaQuery runs db.QueryContext and, when schema debug logging is
// enabled, logs the time the query took to return (and the SQL itself when
// LOG_SCHEMA_SQL is set).
func (c *Collection) loggedSchemaQuery(ctx context.Context, db *sql.DB, label, query string, args ...interface{}) (*sql.Rows, error) {
	return loggedSchemaQueryWithLogger(ctx, c.Logger, db, label, query, args...)
}

// loggedSchemaQueryWithLogger is the logger-based variant of loggedSchemaQuery,
// for schema query callers that do not have a *Collection available.
func loggedSchemaQueryWithLogger(ctx context.Context, logger *util.Logger, db *sql.DB, label, query string, args ...interface{}) (*sql.Rows, error) {
	if !SchemaDebugEnabled() {
		return db.QueryContext(ctx, query, args...)
	}

	start := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	elapsed := time.Since(start)

	logger.PrintInfo("%s [query] %s ran in %s", schemaDebugLogPrefix, label, elapsed.Round(time.Microsecond))
	if logSchemaSQLEnabled() {
		logger.PrintInfo("%s [query] %s SQL: %s | args: %v", schemaDebugLogPrefix, label, strings.TrimSpace(query), args)
	}

	return rows, err
}

// traceSchemaStep logs the number of objects returned by a schema collection
// step, the time it took, and a snapshot of Go memory usage afterwards. Only
// counts are logged, never the objects themselves. It is a no-op unless schema
// debug logging is enabled.
func (c *Collection) traceSchemaStep(label string, count int, start time.Time) {
	if !SchemaDebugEnabled() {
		return
	}

	elapsed := time.Since(start)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.Logger.PrintInfo(
		"%s [step]  %s: %d objects, %s | heap %s, sys %s, %d heap objects",
		schemaDebugLogPrefix, label, count, elapsed.Round(time.Microsecond),
		humanizeBytes(m.HeapAlloc), humanizeBytes(m.Sys), m.HeapObjects)
}
