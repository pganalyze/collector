package querysample

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const otelServiceName = "Postgres (pganalyze)"
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

func initProvider(ctx context.Context, endpoint string) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceName(otelServiceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
	}
	scheme := strings.ToLower(url.Scheme)

	var traceExporter *otlptrace.Exporter
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	switch scheme {
	case "http", "https":
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(url.Host)}
		if scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		traceExporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create HTTP trace exporter: %w", err)
		}
	case "grpc":
		// For now we always require TLS for gRPC connections
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(url.Host)}
		traceExporter, err = otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create HTTP trace exporter: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported protocol: %s", url.Scheme)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	return tracerProvider, tracerProvider.Shutdown, nil
}

func ReportQuerySamplesAsTraceSpans(ctx context.Context, server *state.Server, logger *util.Logger, grant state.GrantLogs, samples []state.PostgresQuerySample) {
	endpoint := server.Config.OtelExporterOtlpEndpoint
	if endpoint == "" {
		return
	}

	// TODO: Initialize the provider once instead of each time we need to send.
	// When we fix this we likely need to explicitly flush all spans at the end
	// of this function (currently done by shutdown).
	tracerProvider, shutdown, err := initProvider(ctx, server.Config.OtelExporterOtlpEndpoint)
	if err != nil {
		logger.PrintError("Failed to initialize OpenTelemetry tracing provider: %s", err)
		return
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			logger.PrintError("Failed to shutdown OpenTelemetry tracing provider: %s", err)
		}
	}()

	for _, sample := range samples {
		if !sample.HasExplain {
			// Skip samples without an EXPLAIN plan for now
			continue
		}
		queryTags := parseTags(sample.Query)
		if _, ok := queryTags["traceparent"]; ok {
			prop := propagation.TraceContext{}
			ctx := prop.Extract(context.Background(), propagation.MapCarrier(queryTags))

			tracer := tracerProvider.Tracer(
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
		}
	}
}
