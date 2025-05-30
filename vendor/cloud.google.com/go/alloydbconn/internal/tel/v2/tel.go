// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tel provides telemetry into the connector's internal operations.
package tel

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/alloydbconn/debug"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/api/option"

	cmexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	meterName         = "alloydb.googleapis.com/client/connector"
	monitoredResource = "alloydb.googleapis.com/InstanceClient"
	dialCount         = "dial_count"
	dialLatency       = "dial_latencies"
	openConnections   = "open_connections"
	bytesSent         = "bytes_sent_count"
	bytesReceived     = "bytes_received_count"
	refreshCount      = "refresh_count"
	// ProjectID specifies the instance's parent project.
	ProjectID = "project_id"
	// Location specifies the instances region (aka location).
	Location = "location"
	// Cluster specifies the cluster name.
	Cluster = "cluster_id"
	// Instance specifies the instance name.
	Instance = "instance_id"
	// ClientID is a unique ID specifying the instance of the
	// alloydbconn.Dialer.
	ClientID = "client_uid"
	// connectorType is one of go or auth-proxy
	connectorType = "connector_type"
	// authType is one of iam or built-in
	authType = "auth_type"
	// isCacheHit reports whether connection info was available in the cache
	isCacheHit = "is_cache_hit"
	// status indicates whether the dial attempt succeeded or not.
	status = "status"
	// refreshType indicates whether the cache is a refresh ahead cache or a
	// lazy cache.
	refreshType = "refresh_type"
	// DialSuccess indicates the dial attempt succeeded.
	DialSuccess = "success"
	// DialUserError indicates the dial attempt failed due to a user mistake.
	DialUserError = "user_error"
	// DialCacheError indicates the dialer failed to retrieved the cached
	// connection info.
	DialCacheError = "cache_error"
	// DialTCPError indicates a TCP-level error.
	DialTCPError = "tcp_error"
	// DialTLSError indicates an error with the TLS connection.
	DialTLSError = "tls_error"
	// DialMDXError indicates an error with the metadata exchange.
	DialMDXError = "mdx_error"
	// RefreshSuccess indicates the refresh operation to retrieve new
	// connection info succeeded.
	RefreshSuccess = "success"
	// RefreshFailure indicates the refresh operation failed.
	RefreshFailure = "failure"
	// RefreshAheadType indicates the dialer is using a refresh ahead cache.
	RefreshAheadType = "refresh_ahead"
	// RefreshLazyType indicates the dialer is using a lazy cache.
	RefreshLazyType = "lazy"
)

// Config holds all the necessary information to configure a MetricRecorder.
type Config struct {
	// Enabled specifies whether the metrics should be enabled.
	Enabled bool
	// Version is the version of the alloydbconn.Dialer.
	Version string
	// ClientID uniquely identifies the instance of the alloydbconn.Dialer.
	ClientID string
	// ProjectID is the project ID of the AlloyDB instance.
	ProjectID string
	// LocationAlloyDBs the location of the AlloyDB instance.
	Location string
	// Cluster is the name of the AlloyDB cluster.
	Cluster string
	// Instance is the name of the AlloyDB instance.
	Instance string
}

// MetricRecorder defines the interface for recording metrics related to the
// internal operations of alloydbconn.Dialer.
type MetricRecorder interface {
	Shutdown(context.Context) error
	RecordBytesRxCount(context.Context, int64, Attributes)
	RecordBytesTxCount(context.Context, int64, Attributes)
	RecordDialCount(context.Context, Attributes)
	RecordDialLatency(context.Context, int64, Attributes)
	RecordOpenConnection(context.Context, Attributes)
	RecordClosedConnection(context.Context, Attributes)
	RecordRefreshCount(context.Context, Attributes)
}

// DefaultExportInterval is the interval that the metric exporter runs. It
// should always be 60s. This value is exposed as a var to faciliate testing.
var DefaultExportInterval = 60 * time.Second

// NewMetricRecorder creates a MetricRecorder. When the configuration is not
// enabled, a null recorder is returned instead.
func NewMetricRecorder(ctx context.Context, l debug.ContextLogger, cfg Config, opts ...option.ClientOption) MetricRecorder {
	if !cfg.Enabled {
		l.Debugf(ctx, "disabling built-in metrics")
		return NullMetricRecorder{}
	}
	eopts := []cmexporter.Option{
		cmexporter.WithCreateServiceTimeSeries(),
		cmexporter.WithProjectID(cfg.ProjectID),
		cmexporter.WithMonitoringClientOptions(opts...),
		cmexporter.WithMetricDescriptorTypeFormatter(func(m metricdata.Metrics) string {
			return "alloydb.googleapis.com/client/connector/" + m.Name
		}),
		cmexporter.WithMonitoredResourceDescription(monitoredResource, []string{
			ProjectID, Location, Cluster, Instance, ClientID,
		}),
	}
	exp, err := cmexporter.New(eopts...)
	if err != nil {
		l.Debugf(ctx, "built-in metrics exporter failed to initialize: %v", err)
		return NullMetricRecorder{}
	}

	res := resource.NewWithAttributes(monitoredResource,
		// The gcp.resource_type is a special attribute that the exporter
		// transforms into the MonitoredResource field.
		attribute.String("gcp.resource_type", monitoredResource),
		attribute.String(ProjectID, cfg.ProjectID),
		attribute.String(Location, cfg.Location),
		attribute.String(Cluster, cfg.Cluster),
		attribute.String(Instance, cfg.Instance),
		attribute.String(ClientID, cfg.ClientID),
	)
	p := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exp,
			// The periodic reader runs every 60 seconds by default, but set
			// the value anyway to be defensive.
			sdkmetric.WithInterval(DefaultExportInterval),
		)),
		sdkmetric.WithResource(res),
	)
	m := p.Meter(meterName, metric.WithInstrumentationVersion(cfg.Version))

	mDialCount, err := m.Int64Counter(dialCount)
	if err != nil {
		_ = exp.Shutdown(ctx)
		l.Debugf(ctx, "built-in metrics exporter failed to initialize dial count metric: %v", err)
		return NullMetricRecorder{}
	}
	mDialLatency, err := m.Float64Histogram(dialLatency)
	if err != nil {
		_ = exp.Shutdown(ctx)
		l.Debugf(ctx, "built-in metrics exporter failed to initialize dial latency metric: %v", err)
		return NullMetricRecorder{}
	}
	mOpenConns, err := m.Int64UpDownCounter(openConnections)
	if err != nil {
		_ = exp.Shutdown(ctx)
		l.Debugf(ctx, "built-in metrics exporter failed to initialize open connections metric: %v", err)
		return NullMetricRecorder{}
	}
	mBytesTx, err := m.Int64Counter(bytesSent)
	if err != nil {
		_ = exp.Shutdown(ctx)
		l.Debugf(ctx, "built-in metrics exporter failed to initialize bytes sent metric: %v", err)
		return NullMetricRecorder{}
	}
	mBytesRx, err := m.Int64Counter(bytesReceived)
	if err != nil {
		_ = exp.Shutdown(ctx)
		l.Debugf(ctx, "built-in metrics exporter failed to initialize bytes received metric: %v", err)
		return NullMetricRecorder{}
	}
	mRefreshCount, err := m.Int64Counter(refreshCount)
	if err != nil {
		_ = exp.Shutdown(ctx)
		l.Debugf(ctx, "built-in metrics exporter failed to initialize refresh count metric: %v", err)
		return NullMetricRecorder{}
	}
	return &metricRecorder{
		exporter:      exp,
		provider:      p,
		dialerID:      cfg.ClientID,
		mDialCount:    mDialCount,
		mDialLatency:  mDialLatency,
		mOpenConns:    mOpenConns,
		mBytesTx:      mBytesTx,
		mBytesRx:      mBytesRx,
		mRefreshCount: mRefreshCount,
	}
}

// metricRecorder holds the various counters that track internal operations.
type metricRecorder struct {
	exporter      sdkmetric.Exporter
	provider      *sdkmetric.MeterProvider
	dialerID      string
	mDialCount    metric.Int64Counter
	mDialLatency  metric.Float64Histogram
	mOpenConns    metric.Int64UpDownCounter
	mBytesTx      metric.Int64Counter
	mBytesRx      metric.Int64Counter
	mRefreshCount metric.Int64Counter
}

// Shutdown should be called when the MetricRecorder is no longer needed.
func (m *metricRecorder) Shutdown(ctx context.Context) error {
	// Shutdown only the provider. The provider will shutdown the exporter as
	// part of its own shutdown, i.e., provider shuts down the reader, the
	// reader shuts down the exporter. So one shutdown call here is enough.
	return m.provider.Shutdown(ctx)
}

func connectorTypeValue(userAgent string) string {
	if strings.Contains(userAgent, "auth-proxy") {
		return "auth_proxy"
	}
	return "go"
}

func authTypeValue(iamAuthn bool) string {
	if iamAuthn {
		return "iam"
	}
	return "built_in"
}

// Attributes holds all the various pieces of metadata to attach to a metric.
type Attributes struct {
	// IAMAuthN specifies whether IAM authentication is enabled.
	IAMAuthN bool
	// UserAgent is the full user-agent of the alloydbconn.Dialer.
	UserAgent string
	// CacheHit specifies whether connection info was present in the cache.
	CacheHit bool
	// DialStatus specifies the result of the dial attempt.
	DialStatus string
	// RefreshStatus specifies the result of the refresh operation.
	RefreshStatus string
	// RefreshType specifies the type of cache in use (e.g., refresh ahead or
	// lazy).
	RefreshType string
}

// RecordBytesRxCount records the number of bytes received for a particular
// instance.
func (m *metricRecorder) RecordBytesRxCount(ctx context.Context, bytes int64, a Attributes) {
	m.mBytesRx.Add(ctx, bytes,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
		)),
	)
}

// RecordBytesTxCount records the number of bytes send for a paritcular
// instance.
func (m *metricRecorder) RecordBytesTxCount(ctx context.Context, bytes int64, a Attributes) {
	m.mBytesTx.Add(ctx, bytes,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
		)),
	)
}

// RecordDialCount records increments the number of dial attempts.
func (m *metricRecorder) RecordDialCount(ctx context.Context, a Attributes) {
	m.mDialCount.Add(ctx, 1,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
			attribute.String(authType, authTypeValue(a.IAMAuthN)),
			attribute.Bool(isCacheHit, a.CacheHit),
			attribute.String(status, a.DialStatus)),
		))
}

// RecordDialLatency records a latency measurement for a particular dial
// attempt.
func (m *metricRecorder) RecordDialLatency(ctx context.Context, latencyMS int64, a Attributes) {
	m.mDialLatency.Record(ctx, float64(latencyMS),
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
		)),
	)
}

// RecordOpenConnection increments the number of open connections.
func (m *metricRecorder) RecordOpenConnection(ctx context.Context, a Attributes) {
	m.mOpenConns.Add(ctx, 1,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
			attribute.String(authType, authTypeValue(a.IAMAuthN)),
		)),
	)
}

// RecordClosedConnection decrements the number of open connections.
func (m *metricRecorder) RecordClosedConnection(ctx context.Context, a Attributes) {
	m.mOpenConns.Add(ctx, -1,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
			attribute.String(authType, authTypeValue(a.IAMAuthN)),
		)),
	)
}

// RecordRefreshCount records the result of a refresh operation.
func (m *metricRecorder) RecordRefreshCount(ctx context.Context, a Attributes) {
	m.mRefreshCount.Add(ctx, 1,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String(connectorType, connectorTypeValue(a.UserAgent)),
			attribute.String(status, a.RefreshStatus),
			attribute.String(refreshType, a.RefreshType),
		)),
	)
}

// NullMetricRecorder implements the MetricRecorder interface with no-ops. It
// is useful for disabling the built-in metrics.
type NullMetricRecorder struct{}

// Shutdown is a no-op.
func (NullMetricRecorder) Shutdown(context.Context) error { return nil }

// RecordBytesRxCount is a no-op.
func (NullMetricRecorder) RecordBytesRxCount(context.Context, int64, Attributes) {}

// RecordBytesTxCount is a no-op.
func (NullMetricRecorder) RecordBytesTxCount(context.Context, int64, Attributes) {}

// RecordDialCount is a no-op.
func (NullMetricRecorder) RecordDialCount(context.Context, Attributes) {}

// RecordDialLatency is a no-op.
func (NullMetricRecorder) RecordDialLatency(context.Context, int64, Attributes) {}

// RecordOpenConnection is a no-op.
func (NullMetricRecorder) RecordOpenConnection(context.Context, Attributes) {}

// RecordClosedConnection is a no-op.
func (NullMetricRecorder) RecordClosedConnection(context.Context, Attributes) {}

// RecordRefreshCount is a no-op.
func (NullMetricRecorder) RecordRefreshCount(context.Context, Attributes) {}
