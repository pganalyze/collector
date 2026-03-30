package selfhosted

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	otlpLogsService "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	otlpLogs "go.opentelemetry.io/proto/otlp/logs/v1"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestOtelHandlerProtobuf(t *testing.T) {
	tests := []struct {
		name                  string
		logsData              *otlpLogs.LogsData
		expectRawItems        int
		expectParsedItems     int
		expectRejectedRecords int64
		checkParsed           func(t *testing.T, items []state.ParsedLogStreamItem)
	}{
		{
			name:              "plain log message",
			logsData:          makePlainLogsData(),
			expectRawItems:    1,
			expectParsedItems: 0,
		},
		{
			name:              "jsonlog message",
			logsData:          makeJsonlogLogsData(),
			expectRawItems:    0,
			expectParsedItems: 1,
			checkParsed: func(t *testing.T, items []state.ParsedLogStreamItem) {
				if items[0].LogLine.Content != "database system is ready to accept connections" {
					t.Errorf("unexpected content: %s", items[0].LogLine.Content)
				}
				if items[0].LogLine.Username != "postgres" {
					t.Errorf("unexpected username: %s", items[0].LogLine.Username)
				}
				if items[0].LogLine.Database != "mydb" {
					t.Errorf("unexpected database: %s", items[0].LogLine.Database)
				}
				if items[0].LogLine.BackendPid != 123 {
					t.Errorf("unexpected backend pid: %d", items[0].LogLine.BackendPid)
				}
				if items[0].LogLine.LogLevel != pganalyze_collector.LogLineInformation_LOG {
					t.Errorf("unexpected log level: %v", items[0].LogLine.LogLevel)
				}
			},
		},
		{
			name:              "k8s jsonlog with detail line",
			logsData:          makeK8sJsonlogLogsData(),
			expectRawItems:    0,
			expectParsedItems: 2,
			checkParsed: func(t *testing.T, items []state.ParsedLogStreamItem) {
				if items[0].LogLine.Content != "relation \"missing\" does not exist" {
					t.Errorf("unexpected content: %s", items[0].LogLine.Content)
				}
				if items[0].LogLine.LogLevel != pganalyze_collector.LogLineInformation_ERROR {
					t.Errorf("unexpected log level: %v", items[0].LogLine.LogLevel)
				}
				if items[1].LogLine.Content != "some detail" {
					t.Errorf("unexpected detail content: %s", items[1].LogLine.Content)
				}
				if items[1].LogLine.LogLevel != pganalyze_collector.LogLineInformation_DETAIL {
					t.Errorf("unexpected detail log level: %v", items[1].LogLine.LogLevel)
				}
			},
		},
		{
			name: "kvlist with unknown logger is rejected",
			logsData: otelLogsData(testTimestamp, otelKVList(
				otelKV("logger", "nginx"),
				otelKVListEntry("record",
					otelKV("message", "some message"),
				),
			)),
			expectRawItems:        0,
			expectParsedItems:     0,
			expectRejectedRecords: 1,
		},
		{
			name: "kvlist without error_severity or logger is rejected",
			logsData: otelLogsData(testTimestamp, otelKVList(
				otelKV("message", "some unrecognized format"),
				otelKV("level", "info"),
			)),
			expectRawItems:        0,
			expectParsedItems:     0,
			expectRejectedRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" protobuf", func(t *testing.T) {
			server, logger := makeOtelTestServer()
			rawLogStream := make(chan SelfHostedLogStreamItem, 10)
			parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

			body, err := proto.Marshal(tt.logsData)
			if err != nil {
				t.Fatalf("failed to marshal protobuf: %v", err)
			}

			warnedAboutMultipleServers := false
			resp, err := handleOtlpLogsRequestProtobuf(body, []*state.Server{server}, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var response otlpLogsService.ExportLogsServiceResponse
			if err := proto.Unmarshal(resp, &response); err != nil {
				t.Fatalf("failed to unmarshal protobuf response: %v", err)
			}

			var gotRejected int64
			if response.PartialSuccess != nil {
				gotRejected = response.PartialSuccess.RejectedLogRecords
			}
			if gotRejected != tt.expectRejectedRecords {
				t.Errorf("expected %d rejected records, got %d", tt.expectRejectedRecords, gotRejected)
			}

			assertStreamItems(t, rawLogStream, parsedLogStream, tt.expectRawItems, tt.expectParsedItems, tt.checkParsed)
		})

		t.Run(tt.name+" json", func(t *testing.T) {
			server, logger := makeOtelTestServer()
			rawLogStream := make(chan SelfHostedLogStreamItem, 10)
			parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

			body, err := protojson.Marshal(tt.logsData)
			if err != nil {
				t.Fatalf("failed to marshal JSON: %v", err)
			}

			warnedAboutMultipleServers := false
			resp, err := handleOtlpLogsRequestJson(body, []*state.Server{server}, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var response otlpLogsService.ExportLogsServiceResponse
			if err := protojson.Unmarshal(resp, &response); err != nil {
				t.Fatalf("failed to unmarshal JSON response: %v", err)
			}

			var gotRejected int64
			if response.PartialSuccess != nil {
				gotRejected = response.PartialSuccess.RejectedLogRecords
			}
			if gotRejected != tt.expectRejectedRecords {
				t.Errorf("expected %d rejected records, got %d", tt.expectRejectedRecords, gotRejected)
			}

			assertStreamItems(t, rawLogStream, parsedLogStream, tt.expectRawItems, tt.expectParsedItems, tt.checkParsed)
		})
	}
}

func TestOtelHandlerHTTPEndpoint(t *testing.T) {
	tests := []struct {
		name               string
		contentType        string
		body               []byte
		expectStatus       int
		expectErrorMessage string
	}{
		{
			name:         "valid protobuf request",
			contentType:  "application/x-protobuf",
			body:         mustMarshalProto(t, makeJsonlogLogsData()),
			expectStatus: http.StatusOK,
		},
		{
			name:         "valid json request",
			contentType:  "application/json",
			body:         mustMarshalProtoJSON(t, makeJsonlogLogsData()),
			expectStatus: http.StatusOK,
		},
		{
			name:         "unsupported content type",
			contentType:  "text/plain",
			body:         []byte("hello"),
			expectStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:               "invalid protobuf returns 400 with status error",
			contentType:        "application/x-protobuf",
			body:               []byte("not valid protobuf"),
			expectStatus:       http.StatusBadRequest,
			expectErrorMessage: "Could not unmarshal Protobuf request body",
		},
		{
			name:               "invalid json returns 400 with status error",
			contentType:        "application/json",
			body:               []byte("not valid json"),
			expectStatus:       http.StatusBadRequest,
			expectErrorMessage: "Could not unmarshal JSON request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, logger := makeOtelTestServer()
			rawLogStream := make(chan SelfHostedLogStreamItem, 10)
			parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

			warnedAboutMultipleServers := false
			handler := makeOtelLogsHandler([]*state.Server{server}, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
			req := httptest.NewRequest(http.MethodPost, "/v1/logs", bytes.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, rec.Code)
			}

			if tt.expectErrorMessage != "" {
				respBody := rec.Body.Bytes()
				var status spb.Status
				switch tt.contentType {
				case "application/x-protobuf":
					if err := proto.Unmarshal(respBody, &status); err != nil {
						t.Fatalf("failed to unmarshal protobuf error response: %v", err)
					}
				case "application/json":
					if err := protojson.Unmarshal(respBody, &status); err != nil {
						t.Fatalf("failed to unmarshal JSON error response: %v", err)
					}
				}
				if status.Message != tt.expectErrorMessage {
					t.Errorf("expected error message %q, got %q", tt.expectErrorMessage, status.Message)
				}
				if status.Code != int32(http.StatusBadRequest) {
					t.Errorf("expected status code %d in error body, got %d", http.StatusBadRequest, status.Code)
				}
			}
		})
	}
}

func otelStringVal(s string) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: s}}
}

func otelKV(key, value string) *common.KeyValue {
	return &common.KeyValue{Key: key, Value: otelStringVal(value)}
}

func otelKVList(kvs ...*common.KeyValue) *common.AnyValue {
	return &common.AnyValue{
		Value: &common.AnyValue_KvlistValue{
			KvlistValue: &common.KeyValueList{Values: kvs},
		},
	}
}

func otelKVListEntry(key string, kvs ...*common.KeyValue) *common.KeyValue {
	return &common.KeyValue{Key: key, Value: otelKVList(kvs...)}
}

func otelLogsData(ts time.Time, body *common.AnyValue) *otlpLogs.LogsData {
	return &otlpLogs.LogsData{
		ResourceLogs: []*otlpLogs.ResourceLogs{
			{
				ScopeLogs: []*otlpLogs.ScopeLogs{
					{
						LogRecords: []*otlpLogs.LogRecord{
							{
								TimeUnixNano: uint64(ts.UnixNano()),
								Body:         body,
							},
						},
					},
				},
			},
		},
	}
}

var testTimestamp = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func makePlainLogsData() *otlpLogs.LogsData {
	return otelLogsData(testTimestamp,
		otelStringVal("2026-01-01 12:00:00 UTC [123] LOG: database system is ready to accept connections"),
	)
}

func makeJsonlogLogsData() *otlpLogs.LogsData {
	return otelLogsData(testTimestamp, otelKVList(
		otelKV("error_severity", "LOG"),
		otelKV("message", "database system is ready to accept connections"),
		otelKV("log_time", "2026-01-01 12:00:00.000 UTC"),
		otelKV("process_id", "123"),
		otelKV("user_name", "postgres"),
		otelKV("database_name", "mydb"),
	))
}

func makeK8sJsonlogLogsData() *otlpLogs.LogsData {
	return otelLogsData(testTimestamp, otelKVList(
		otelKV("logger", "postgres"),
		otelKVListEntry("record",
			otelKV("error_severity", "ERROR"),
			otelKV("message", "relation \"missing\" does not exist"),
			otelKV("detail", "some detail"),
			otelKV("log_time", "2026-01-01 12:00:00.000 UTC"),
			otelKV("process_id", "456"),
		),
		otelKVListEntry("kubernetes",
			otelKV("pod_name", "pg-pod-0"),
			otelKV("namespace_name", "default"),
		),
	))
}

func makeOtelTestServer() (*state.Server, *util.Logger) {
	server := state.MakeServer(config.ServerConfig{}, false)
	server.LogParser = logs.NewLogParser("%m [%p] ", nil)
	logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
	return server, logger
}

func drainChannel[T any](ch chan T) []T {
	var items []T
	for {
		select {
		case item := <-ch:
			items = append(items, item)
		default:
			return items
		}
	}
}

func assertStreamItems(t *testing.T, rawLogStream chan SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, expectRaw int, expectParsed int, checkParsed func(t *testing.T, items []state.ParsedLogStreamItem)) {
	t.Helper()

	rawItems := drainChannel(rawLogStream)
	parsedItems := drainChannel(parsedLogStream)

	if len(rawItems) != expectRaw {
		t.Errorf("expected %d raw items, got %d", expectRaw, len(rawItems))
	}
	if len(parsedItems) != expectParsed {
		t.Errorf("expected %d parsed items, got %d", expectParsed, len(parsedItems))
	}

	if checkParsed != nil && len(parsedItems) == expectParsed {
		checkParsed(t, parsedItems)
	}
}

func makeOtelTestServerWithConfig(cfg config.ServerConfig) (*state.Server, *util.Logger) {
	server := state.MakeServer(cfg, false)
	server.LogParser = logs.NewLogParser("%m [%p] ", nil)
	logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
	return server, logger
}

func TestOtelMultiplexing(t *testing.T) {
	t.Run("plain log sent to all servers via rawLogStream", func(t *testing.T) {
		server1, _ := makeOtelTestServerWithConfig(config.ServerConfig{SectionName: "s1"})
		server2, _ := makeOtelTestServerWithConfig(config.ServerConfig{SectionName: "s2"})
		logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
		servers := []*state.Server{server1, server2}

		rawLogStream := make(chan SelfHostedLogStreamItem, 10)
		parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

		warnedAboutMultipleServers := false
		body, _ := proto.Marshal(makePlainLogsData())
		_, err := handleOtlpLogsRequestProtobuf(body, servers, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Plain logs go to the multiplexed rawLogStream (one item, forwarded to all servers by the multiplexer)
		assertStreamItems(t, rawLogStream, parsedLogStream, 1, 0, nil)
		if !warnedAboutMultipleServers {
			t.Error("expected warning about multiple servers")
		}
	})

	t.Run("simple jsonlog sent to all servers", func(t *testing.T) {
		server1, _ := makeOtelTestServerWithConfig(config.ServerConfig{SectionName: "s1"})
		server2, _ := makeOtelTestServerWithConfig(config.ServerConfig{SectionName: "s2"})
		logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
		servers := []*state.Server{server1, server2}

		rawLogStream := make(chan SelfHostedLogStreamItem, 10)
		parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

		warnedAboutMultipleServers := false
		body, _ := proto.Marshal(makeJsonlogLogsData())
		_, err := handleOtlpLogsRequestProtobuf(body, servers, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Simple jsonlog is sent to both servers as parsed items
		assertStreamItems(t, rawLogStream, parsedLogStream, 0, 2, func(t *testing.T, items []state.ParsedLogStreamItem) {
			if items[0].Identifier != server1.Config.Identifier {
				t.Errorf("expected first item for server1, got %v", items[0].Identifier)
			}
			if items[1].Identifier != server2.Config.Identifier {
				t.Errorf("expected second item for server2, got %v", items[1].Identifier)
			}
		})
		if !warnedAboutMultipleServers {
			t.Error("expected warning about multiple servers")
		}
	})

	t.Run("k8s jsonlog filtered per server", func(t *testing.T) {
		server1, _ := makeOtelTestServerWithConfig(config.ServerConfig{
			SectionName:       "s1",
			LogOtelK8SPodName: "pg-pod-0",
			LogOtelK8SPod:     "pg-pod-0",
		})
		server2, _ := makeOtelTestServerWithConfig(config.ServerConfig{
			SectionName:       "s2",
			LogOtelK8SPodName: "pg-pod-1",
			LogOtelK8SPod:     "pg-pod-1",
		})
		logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
		servers := []*state.Server{server1, server2}

		rawLogStream := make(chan SelfHostedLogStreamItem, 10)
		parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

		warnedAboutMultipleServers := false
		// makeK8sJsonlogLogsData has pod_name "pg-pod-0" which matches server1 only
		body, _ := proto.Marshal(makeK8sJsonlogLogsData())
		_, err := handleOtlpLogsRequestProtobuf(body, servers, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// K8s log should only go to server1 (pg-pod-0), not server2 (pg-pod-1)
		// The K8s log has a detail line, so 2 parsed items for the one matching server
		assertStreamItems(t, rawLogStream, parsedLogStream, 0, 2, func(t *testing.T, items []state.ParsedLogStreamItem) {
			for _, item := range items {
				if item.Identifier != server1.Config.Identifier {
					t.Errorf("expected item for server1, got %v", item.Identifier)
				}
			}
		})
		// No warning expected since K8s filtering is in use
		if warnedAboutMultipleServers {
			t.Error("did not expect warning about multiple servers when K8s filtering is active")
		}
	})

	t.Run("k8s jsonlog sent to both servers when both match", func(t *testing.T) {
		server1, _ := makeOtelTestServerWithConfig(config.ServerConfig{
			SectionName:       "s1",
			Identifier:        config.ServerIdentifier{SystemID: "s1"},
			LogOtelK8SPodName: "pg-pod-0",
			LogOtelK8SPod:     "pg-pod-0",
		})
		server2, _ := makeOtelTestServerWithConfig(config.ServerConfig{
			SectionName: "s2",
			Identifier:  config.ServerIdentifier{SystemID: "s2"},
			// No K8s filter, so all K8s logs match
		})
		logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
		servers := []*state.Server{server1, server2}

		rawLogStream := make(chan SelfHostedLogStreamItem, 10)
		parsedLogStream := make(chan state.ParsedLogStreamItem, 10)

		warnedAboutMultipleServers := false
		body, _ := proto.Marshal(makeK8sJsonlogLogsData())
		_, err := handleOtlpLogsRequestProtobuf(body, servers, rawLogStream, parsedLogStream, logger, false, &warnedAboutMultipleServers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Both servers match: server1 by pod name, server2 has no filter
		// The K8s log has a detail line, so 2 parsed items per server = 4 total
		assertStreamItems(t, rawLogStream, parsedLogStream, 0, 4, func(t *testing.T, items []state.ParsedLogStreamItem) {
			s1Count := 0
			s2Count := 0
			for _, item := range items {
				if item.Identifier == server1.Config.Identifier {
					s1Count++
				} else if item.Identifier == server2.Config.Identifier {
					s2Count++
				}
			}
			if s1Count != 2 {
				t.Errorf("expected 2 items for server1, got %d", s1Count)
			}
			if s2Count != 2 {
				t.Errorf("expected 2 items for server2, got %d", s2Count)
			}
		})
	})
}

func mustMarshalProto(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal protobuf: %v", err)
	}
	return b
}

func mustMarshalProtoJSON(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := protojson.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal protojson: %v", err)
	}
	return b
}
