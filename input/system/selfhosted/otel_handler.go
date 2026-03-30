package selfhosted

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pganalyze/collector/config"
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

func setupOtelHandler(ctx context.Context, servers []*state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, prefixedLogger *util.Logger, opts state.CollectionOpts) {
	otelLogServer := servers[0].Config.LogOtelServer

	// Track whether we've warned about multiple servers sharing this OTel endpoint
	warnedAboutMultipleServers := false

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/v1/logs", makeOtelLogsHandler(servers, rawLogStream, parsedLogStream, prefixedLogger, opts.VeryVerbose, &warnedAboutMultipleServers))

	util.GoServeHTTP(ctx, prefixedLogger, otelLogServer, serveMux)
}

func makeOtelLogsHandler(servers []*state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, prefixedLogger *util.Logger, veryVerbose bool, warnedAboutMultipleServers *bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			prefixedLogger.PrintError("OTel log server could not read request body: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Could not read request body"))
			return
		}

		var resp []byte
		contentType := r.Header.Get("Content-Type")
		switch contentType {
		case "application/x-protobuf":
			resp, err = handleOtlpLogsRequestProtobuf(b, servers, rawLogStream, parsedLogStream, prefixedLogger, veryVerbose, warnedAboutMultipleServers)
		case "application/json":
			resp, err = handleOtlpLogsRequestJson(b, servers, rawLogStream, parsedLogStream, prefixedLogger, veryVerbose, warnedAboutMultipleServers)
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Unsupported Content-Type, must be application/x-protobuf or application/json"))
			return
		}

		if err != nil {
			if len(resp) == 0 {
				contentType = "text/plain"
				resp = []byte(err.Error())
			}
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Header().Set("Content-Type", contentType)
		w.Write(resp)
	}
}

func handleOtlpLogsRequestProtobuf(b []byte, servers []*state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, prefixedLogger *util.Logger, veryVerbose bool, warnedAboutMultipleServers *bool) (resp []byte, err error) {
	logsData := &otlpLogs.LogsData{}
	if err = proto.Unmarshal(b, logsData); err != nil {
		prefixedLogger.PrintError("OTel log server could not unmarshal request body, expected binary OTLP Protobuf format: %s", err)
		err = fmt.Errorf("Could not unmarshal Protobuf request body")
		resp, _ = proto.Marshal(&spb.Status{
			Code:    int32(http.StatusBadRequest),
			Message: err.Error(),
		})
		return
	}

	if veryVerbose {
		jsonData, err := json.MarshalIndent(logsData, "", "  ")
		if err != nil {
			prefixedLogger.PrintVerbose("OTel log server failed to convert protobuf to JSON: %v", err)
		} else {
			prefixedLogger.PrintVerbose("OTel log server received Protobuf log data in the following format:\n")
			prefixedLogger.PrintVerbose(string(jsonData))
		}
	}

	response := handleOtlpLogsRequest(logsData, servers, rawLogStream, parsedLogStream, warnedAboutMultipleServers, prefixedLogger)

	return proto.Marshal(response)
}

func handleOtlpLogsRequestJson(b []byte, servers []*state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, prefixedLogger *util.Logger, veryVerbose bool, warnedAboutMultipleServers *bool) (resp []byte, err error) {
	logsData := &otlpLogs.LogsData{}
	if err = protojson.Unmarshal(b, logsData); err != nil {
		prefixedLogger.PrintError("OTel log server could not unmarshal request body, JSON does not match expected format: %s\n  received body: %s", err, string(b))
		err = fmt.Errorf("Could not unmarshal JSON request body")
		resp, _ = protojson.Marshal(&spb.Status{
			Code:    int32(http.StatusBadRequest),
			Message: err.Error(),
		})
		return
	}

	if veryVerbose {
		jsonData, err := json.MarshalIndent(logsData, "", "  ")
		if err != nil {
			prefixedLogger.PrintVerbose("OTel log server failed to convert decoded JSON back to JSON: %v\noriginal:%s", err, string(b))
		} else {
			prefixedLogger.PrintVerbose("OTel log server received JSON log data in the following format:\n")
			prefixedLogger.PrintVerbose(string(jsonData))
		}
	}

	response := handleOtlpLogsRequest(logsData, servers, rawLogStream, parsedLogStream, warnedAboutMultipleServers, prefixedLogger)

	return protojson.Marshal(response)
}

// handleOtlpLogsRequest - Takes one or more OTLP log records and processes them
//
// There are currently three kinds of log formats we aim to support here:
//
// 1. Plain log messages (unstructured message, body of log record is a string)
// 2. jsonlog encoded as OTel key/value map
// 3. jsonlog wrapped in K8s context (via fluentbit/Vector) as OTel key/value map
//
// Other variants (e.g. csvlog, or plain messages in a K8s context) are currently
// not supported and will be ignored.
func handleOtlpLogsRequest(logsData *otlpLogs.LogsData, servers []*state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, warnedAboutMultipleServers *bool, prefixedLogger *util.Logger) *otlpLogsService.ExportLogsServiceResponse {
	var rejectedLogRecords int64
	for _, r := range logsData.ResourceLogs {
		for _, s := range r.ScopeLogs {
			for _, l := range s.LogRecords {
				if l.Body.GetKvlistValue() != nil {
					// jsonlog log message
					record, kubernetes := transformJsonLogRecord(l.Body.GetKvlistValue())
					if record != nil {
						if kubernetes != nil {
							// K8s-wrapped jsonlog: send to all servers that pass the K8s filter
							for _, server := range servers {
								if skipDueToK8sFilter(kubernetes, server.Config) {
									continue
								}
								logParser := server.GetLogParser()
								logLine, detailLine := logLineFromJsonlog(record, logParser)
								parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
								if detailLine != nil {
									parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *detailLine}
								}
							}
						} else {
							// Simple jsonlog: send to all servers
							warnAboutMultipleServers(servers, warnedAboutMultipleServers, prefixedLogger)
							for _, server := range servers {
								logParser := server.GetLogParser()
								logLine, detailLine := logLineFromJsonlog(record, logParser)
								parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
								if detailLine != nil {
									parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *detailLine}
								}
							}
						}
					} else {
						rejectedLogRecords++
					}
				} else if l.Body.GetStringValue() != "" {
					// Plain log message (goes through log transformer which handles per-server routing)
					warnAboutMultipleServers(servers, warnedAboutMultipleServers, prefixedLogger)
					item := SelfHostedLogStreamItem{}
					item.Line = l.Body.GetStringValue()
					item.OccurredAt = time.Unix(0, int64(l.TimeUnixNano))
					rawLogStream <- item
				} else {
					rejectedLogRecords++
				}
			}
		}
	}
	response := &otlpLogsService.ExportLogsServiceResponse{}
	if rejectedLogRecords > 0 {
		response.PartialSuccess = &otlpLogsService.ExportLogsPartialSuccess{RejectedLogRecords: rejectedLogRecords}
	}
	return response
}

// transformJsonLogRecord - Extract the log record and optional Kubernetes
// metadata from an OTel key/value list.
//
// Returns (record, kubernetes) for Kubernetes-wrapped logs, e.g. from Vector.
// Returns (record, nil) for simple jsonlog inputs without Kubernetes metadata.
// Returns (nil, nil) if the record is not recognized.
func transformJsonLogRecord(recordContainer *common.KeyValueList) (record *common.KeyValueList, kubernetes *common.KeyValueList) {
	var logger string
	hasErrorSeverity := false
	for _, v := range recordContainer.Values {
		if v.Key == "logger" {
			logger = v.Value.GetStringValue()
		}
		if v.Key == "record" {
			record = v.Value.GetKvlistValue()
		}
		if v.Key == "kubernetes" {
			kubernetes = v.Value.GetKvlistValue()
		}
		if v.Key == "error_severity" {
			hasErrorSeverity = true
		}
	}
	// TODO: Support other logger names (this is only tested with CNPG)
	if logger == "postgres" {
		// jsonlog wrapped in K8s context (via fluentbit / Vector)
		return record, kubernetes
	} else if logger == "" && hasErrorSeverity {
		// simple jsonlog (Postgres jsonlog has error_severity key)
		return recordContainer, nil
	}

	return nil, nil
}

func warnAboutMultipleServers(servers []*state.Server, warnedAboutMultipleServers *bool, prefixedLogger *util.Logger) {
	if *warnedAboutMultipleServers || len(servers) <= 1 {
		return
	}
	var otherSectionNames []string
	for _, s := range servers[1:] {
		otherSectionNames = append(otherSectionNames, s.Config.SectionName)
	}
	prefixedLogger.PrintWarning("Logs will also be forwarded to other servers (%s) that share the same db_log_otel_server (use K8s pod/label filtering to separate)", strings.Join(otherSectionNames, ", "))
	*warnedAboutMultipleServers = true
}

func logLineFromJsonlog(record *common.KeyValueList, logParser state.LogParser) (state.LogLine, *state.LogLine) {
	var logLine state.LogLine

	// If a DETAIL line is set, we need to create an additional log line
	detailLineContent := ""

	for _, rv := range record.Values {
		if rv.Key == "log_time" {
			logLine.OccurredAt = logParser.GetOccurredAt(rv.Value.GetStringValue())
		}
		if rv.Key == "user_name" {
			logLine.Username = rv.Value.GetStringValue()
		}
		if rv.Key == "database_name" {
			logLine.Database = rv.Value.GetStringValue()
		}
		if rv.Key == "process_id" {
			backendPid, _ := strconv.ParseInt(rv.Value.GetStringValue(), 10, 32)
			logLine.BackendPid = int32(backendPid)
		}
		if rv.Key == "application_name" {
			logLine.Application = rv.Value.GetStringValue()
		}
		if rv.Key == "session_line_num" {
			logLineNumber, _ := strconv.ParseInt(rv.Value.GetStringValue(), 10, 32)
			logLine.LogLineNumber = int32(logLineNumber)
		}
		if rv.Key == "message" {
			logLine.Content = rv.Value.GetStringValue()
		}
		if rv.Key == "detail" {
			detailLineContent = rv.Value.GetStringValue()
		}
		if rv.Key == "error_severity" {
			logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[rv.Value.GetStringValue()])
		}
	}
	if detailLineContent != "" {
		detailLine := logLine
		detailLine.Content = detailLineContent
		detailLine.LogLevel = pganalyze_collector.LogLineInformation_DETAIL
		return logLine, &detailLine
	}
	return logLine, nil
}

func skipDueToK8sFilter(kubernetes *common.KeyValueList, config config.ServerConfig) bool {
	var k8sPodName string
	var k8sNamespaceName string

	k8sLabels := make(map[string]string)
	for _, rv := range kubernetes.Values {
		if rv.Key == "pod_name" {
			k8sPodName = rv.Value.GetStringValue()
		}
		if rv.Key == "namespace_name" {
			k8sNamespaceName = rv.Value.GetStringValue()
		}
		if rv.Key == "labels" {
			for _, ll := range rv.Value.GetKvlistValue().Values {
				k8sLabels[ll.Key] = ll.Value.GetStringValue()
			}
		}
	}

	if config.LogOtelK8SPod != "" {
		if config.LogOtelK8SPodNamespace != "" && config.LogOtelK8SPodNamespace != k8sNamespaceName {
			return true
		}
		if config.LogOtelK8SPodName != k8sPodName {
			return true
		}
	}

	if config.LogOtelK8SLabels != "" {
		return util.CheckLabelSelectorMismatch(k8sLabels, config.LogOtelK8SLabelSelectors)
	}
	return false
}
