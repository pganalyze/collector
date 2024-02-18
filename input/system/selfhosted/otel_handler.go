package selfhosted

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
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

var k8sSelectorRegexp = regexp.MustCompile(`\s*([^!=\s]+)\s*([!=]+)\s*([^\s]+)\s*`)

func logLineFromJsonlog(record *common.KeyValueList, tz *time.Location) (state.LogLine, *state.LogLine) {
	var logLine state.LogLine
	logLine.CollectedAt = time.Now()
	logLine.UUID = uuid.NewV4()

	// If a DETAIL line is set, we need to create an additional log line
	detailLineContent := ""

	for _, rv := range record.Values {
		if rv.Key == "log_time" {
			logLine.OccurredAt = logs.GetOccurredAt(rv.Value.GetStringValue(), tz, false)
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
		parts := strings.SplitN(server.Config.LogOtelK8SPod, "/", 2)
		if len(parts) == 2 {
			if k8sNamespaceName != parts[0] || k8sPodName != parts[1] {
				return true
			}
		} else if len(parts) == 1 {
			if k8sPodName != parts[0] {
				return true
			}
		} else {
			prefixedLogger.PrintWarning("Pod specification for OTel server not valid (need zero or one / separator): \"%s\", skipping log record\n", server.Config.LogOtelK8SPod)
			return true
		}
	}

	if server.Config.LogOtelK8SLabels != "" {
		selectors := strings.Split(server.Config.LogOtelK8SLabels, ",")
		for _, selector := range selectors {
			parts := k8sSelectorRegexp.FindStringSubmatch(selector)
			if parts != nil {
				selKey := parts[1]
				selEq := parts[2] == "=" || parts[2] == "=="
				selNotEq := parts[2] == "!="
				selValue := parts[3]
				v, ok := k8sLabels[selKey]
				if ok {
					if (selEq && v != selValue) || (selNotEq && v == selValue) {
						return true
					}
				}
			} else {
				prefixedLogger.PrintWarning("Label selector for OTel server not valid: \"%s\", skipping log record\n", server.Config.LogOtelK8SLabels)
				return true
			}
		}
	}
	return false
}

func setupOtelHandler(ctx context.Context, server *state.Server, rawLogStream chan<- SelfHostedLogStreamItem, parsedLogStream chan state.ParsedLogStreamItem, prefixedLogger *util.Logger) error {
	otelLogServer := server.Config.LogOtelServer
	tz := server.GetLogTimezone()
	go func() {
		http.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				prefixedLogger.PrintError("Could not read otel body")
			}
			logsData := &otlpLogs.LogsData{}
			if err := proto.Unmarshal(b, logsData); err != nil {
				prefixedLogger.PrintError("Could not unmarshal otel body")
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
							// TODO: Does the logger name need to be configurable? (this is tested with CNPG,
							// but would the logger be different with other operators/container names?)
							if logger == "postgres" {
								// jsonlog wrapped in K8s context (via fluentbit)
								logLine, detailLine := logLineFromJsonlog(record, tz)
								if skipDueToK8sFilter(kubernetes, server, prefixedLogger) {
									continue
								}

								parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
								if detailLine != nil {
									parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *detailLine}
								}
							} else if logger == "" && hasErrorSeverity {
								// simple jsonlog
								logLine, detailLine := logLineFromJsonlog(l.Body.GetKvlistValue(), tz)
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
