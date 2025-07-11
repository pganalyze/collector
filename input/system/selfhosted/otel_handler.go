package selfhosted

import (
	"context"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	otlpLogs "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/proto"
)

// There currently are three kinds of log formats we aim to support here:
//
// 1. Plain log messages (unstructured message, body of log record is a string)
// 2. jsonlog encoded as OTel key/value map
// 3. jsonlog wrapped in K8s context (via fluentbit) as OTel key/value map
//
// Other variants (e.g. csvlog, or plain messages in a K8s context) are currently
// not supported and will be ignored.

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

func skipDueToK8sFilter(kubernetes *common.KeyValueList, server *state.Server, prefixedLogger *util.Logger) bool {
	var k8sPodName string
	var k8sNamespaceName string

	if kubernetes == nil {
		return false
	}

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

	if server.Config.LogOtelK8SPod != "" {
		if server.Config.LogOtelK8SPodNamespace != "" && server.Config.LogOtelK8SPodNamespace != k8sNamespaceName {
			return true
		}
		if server.Config.LogOtelK8SPodName != k8sPodName {
			return true
		}
	}

	if server.Config.LogOtelK8SLabels != "" {
		return util.CheckLabelSelectorMismatch(k8sLabels, server.Config.LogOtelK8SLabelSelectors)
	}
	return false
}

func setupOtelHandler(ctx context.Context, server *state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, prefixedLogger *util.Logger) error {
	otelLogServer := server.Config.LogOtelServer
	logParser := server.GetLogParser()
	go func() {
		http.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				prefixedLogger.PrintError("Could not read otel body")
			}
			logsData := &otlpLogs.LogsData{}
			contentType := r.Header.Get("Content-Type")
			switch contentType {
			case "application/json":
				if err := protojson.Unmarshal(b, logsData); err != nil {
					prefixedLogger.PrintError("Could not unmarshal otel body, json expected")
				}
			default:
				if err := proto.Unmarshal(b, logsData); err != nil {
					prefixedLogger.PrintError("Could not unmarshal otel body, binary payload is expected")
				}
			}

			for _, r := range logsData.ResourceLogs {
				for _, s := range r.ScopeLogs {
					for _, l := range s.LogRecords {
						var logger string
						var record *common.KeyValueList
						var kubernetes *common.KeyValueList
						hasErrorSeverity := false
						if l.Body.GetKvlistValue() != nil {
							for _, v := range l.Body.GetKvlistValue().Values {
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
								// jsonlog wrapped in K8s context (via fluentbit)
								logLine, detailLine := logLineFromJsonlog(record, logParser)
								if skipDueToK8sFilter(kubernetes, server, prefixedLogger) {
									continue
								}

								parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
								if detailLine != nil {
									parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *detailLine}
								}
							} else if logger == "" && hasErrorSeverity {
								// simple jsonlog (Postgres jsonlog has error_severity key)
								logLine, detailLine := logLineFromJsonlog(l.Body.GetKvlistValue(), logParser)
								parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
								if detailLine != nil {
									parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *detailLine}
								}
							}
						} else if l.Body.GetStringValue() != "" {
							// plain log message
							item := SelfHostedLogStreamItem{}
							item.Line = l.Body.GetStringValue()
							item.OccurredAt = time.Unix(0, int64(l.TimeUnixNano))
							rawLogStream <- item
						}
					}
				}
			}
		})
		http.ListenAndServe(otelLogServer, nil)
	}()
	return nil
}
