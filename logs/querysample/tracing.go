package querysample

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const otelSpanName = "EXPLAIN Plan"

func urlToSample(server *state.Server, grant state.GrantLogs, sample state.PostgresQuerySample) string {
	fp := util.FingerprintQuery(sample.Query, server.Config.FilterQueryText, -1)
	fpBin := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBin, fp)

	return fmt.Sprintf(
		"%s/databases/%s/queries/%s/samples/%d?role=%s",
		grant.Config.ServerURL,
		sample.Database,
		hex.EncodeToString(fpBin),
		sample.OccurredAt.Unix(),
		sample.Username,
	)
}

func startAndEndTime(traceState trace.TraceState, sample state.PostgresQuerySample) (startTime time.Time, endTime time.Time) {
	if pganalyzeState := traceState.Get("pganalyze"); pganalyzeState != "" {
		// A pganalyze traceState allows the client to pass the query start time (sent time)
		// on the client side, in nano second precision, like pganalyze=t:1697666938.6297212
		// If there are multiple values in a pganalzye traceState, they are separated by semicolon
		// like pganalyze=t:1697666938.6297212;x=123
		for _, part := range strings.Split(strings.TrimSpace(pganalyzeState), ";") {
			if strings.Contains(part, ":") {
				keyAndValue := strings.SplitN(part, ":", 2)
				if strings.TrimSpace(keyAndValue[0]) == "t" {
					if start, err := util.TimeFromStr(keyAndValue[1]); err == nil {
						startTime = start
						// With this, we're adding the query duration to the start time.
						// This could result creating inaccurate spans, as the start time passed
						// from the client side using tracestate is the time of the query is sent
						// from the client to the server.
						// This means, we will ignore the network time between the client and the
						// server, as well as the machine clock different between them.
						endTime = startTime.Add(time.Duration(sample.RuntimeMs) * time.Millisecond)
						return
					}
				}
			}
		}
	}
	// If no start time was found in the tracestate, calculate start and end time based on sample data
	duration := time.Duration(sample.RuntimeMs) * time.Millisecond
	startTime = sample.OccurredAt.Add(-1 * duration)
	endTime = sample.OccurredAt

	return
}

func ExportQuerySamplesAsTraceSpans(ctx context.Context, server *state.Server, logger *util.Logger, grant state.GrantLogs, samples []state.PostgresQuerySample) {
	exportCount := 0
	for _, sample := range samples {
		if !sample.HasExplain {
			// Skip samples without an EXPLAIN plan for now
			continue
		}
		queryTags := parseTags(sample.Query)
		if _, ok := queryTags["traceparent"]; ok {
			prop := propagation.TraceContext{}
			ctx := prop.Extract(context.Background(), propagation.MapCarrier(queryTags))

			tracer := server.Config.OTelTracingProvider.Tracer(
				"",
				trace.WithInstrumentationVersion(util.CollectorVersion),
				trace.WithSchemaURL(semconv.SchemaURL),
			)
			startTime, endTime := startAndEndTime(trace.SpanContextFromContext(ctx).TraceState(), sample)
			_, span := tracer.Start(ctx, otelSpanName, trace.WithTimestamp(startTime))
			// See https://opentelemetry.io/docs/specs/otel/trace/semantic_conventions/database/
			// however note that "db.postgresql.plan" is non-standard.
			span.SetAttributes(attribute.String("db.system", "postgresql"))
			span.SetAttributes(attribute.String("db.postgresql.plan", urlToSample(server, grant, sample)))
			span.End(trace.WithTimestamp(endTime))
			exportCount += 1
		}
	}

	if exportCount > 0 {
		err := server.Config.OTelTracingProvider.ForceFlush(ctx)
		if err != nil {
			logger.PrintError("Failed to export OpenTelemetry data: %s", err)
		}
		logger.PrintVerbose("Exported %d tracing spans to OpenTelemetry endpoint at %s", exportCount, server.Config.OtelExporterOtlpEndpoint)
	}
}
