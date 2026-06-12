package postgres

import (
	"context"
	"database/sql"
	"runtime"
	"strings"
	"time"

	"github.com/pganalyze/collector/util"
)

// Schema collection debug logging
//
// When very verbose logging is enabled (--very-verbose), the helpers in this file
// emit additional diagnostics for each schema query: the exact SQL that was run,
// how long it took, how many objects each step returned, and a snapshot of Go
// memory usage after each step. This is intended for diagnosing slow or
// memory-intensive schema collection on servers with large or unusual catalogs.
//
// All output is gated on CollectionOpts.VeryVerbose so it never appears under
// plain --verbose, and the helpers are no-ops (with no measurement overhead
// beyond the query itself) when very verbose logging is disabled.

const schemaDebugLogPrefix = "[schema-debug]"

// loggedSchemaQuery runs db.QueryContext and, when very verbose logging is
// enabled, logs the SQL and the time the query took to return.
func (c *Collection) loggedSchemaQuery(ctx context.Context, db *sql.DB, label, query string, args ...interface{}) (*sql.Rows, error) {
	return loggedSchemaQueryWithLogger(ctx, c.Logger, c.GlobalOpts.VeryVerbose, db, label, query, args...)
}

// loggedSchemaQueryWithLogger is the logger-based variant of loggedSchemaQuery,
// for schema query callers that do not have a *Collection available.
func loggedSchemaQueryWithLogger(ctx context.Context, logger *util.Logger, veryVerbose bool, db *sql.DB, label, query string, args ...interface{}) (*sql.Rows, error) {
	if !veryVerbose {
		return db.QueryContext(ctx, query, args...)
	}

	start := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	elapsed := time.Since(start)

	logger.PrintVerbose("%s %s: query ran in %s", schemaDebugLogPrefix, label, elapsed)
	logger.PrintVerbose("%s %s: SQL: %s | args: %v", schemaDebugLogPrefix, label, strings.TrimSpace(query), args)

	return rows, err
}

// traceSchemaStep logs the number of objects returned by a schema collection
// step, the time it took, and a snapshot of Go memory usage afterwards. Only
// counts are logged, never the objects themselves. It is a no-op unless very
// verbose logging is enabled.
func (c *Collection) traceSchemaStep(label string, count int, start time.Time) {
	if !c.GlobalOpts.VeryVerbose {
		return
	}

	elapsed := time.Since(start)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.Logger.PrintVerbose(
		"%s %s: returned %d objects in %s | heapAlloc=%d bytes sys=%d bytes heapObjects=%d",
		schemaDebugLogPrefix, label, count, elapsed, m.HeapAlloc, m.Sys, m.HeapObjects)
}
