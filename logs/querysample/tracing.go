package querysample

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
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
			duration := -1 * time.Duration(sample.RuntimeMs) * time.Millisecond
			startTime := sample.OccurredAt.Add(duration)
			endTime := sample.OccurredAt
			_, span := tracer.Start(ctx, otelSpanName, trace.WithTimestamp(startTime))
			span.SetAttributes(attribute.String("url.full", urlToSample(server, grant, sample)))
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
