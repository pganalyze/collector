package selfhosted

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/mcuadros/go-syslog.v2"

	"github.com/pganalyze/collector/util"
)

var logLinePartsRegexp = regexp.MustCompile(`^\[(\d+)-(\d+)\] (.*)`)
var logLineNumberPartsRegexp = regexp.MustCompile(`^\[(\d+)-(\d+)\]$`)

func setupSyslogHandler(ctx context.Context, logSyslogServer string, out chan<- SelfHostedLogStreamItem, prefixedLogger *util.Logger) error {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	err := server.ListenTCP(logSyslogServer)
	if err != nil {
		return err
	}
	server.Boot()

	go func(ctx context.Context, server *syslog.Server, channel syslog.LogPartsChannel) {
		for {
			select {
			case logParts := <-channel:
				item := SelfHostedLogStreamItem{}

				item.OccurredAt, _ = logParts["timestamp"].(time.Time)

				pidStr, _ := logParts["proc_id"].(string)
				if s, err := strconv.ParseInt(pidStr, 10, 32); err == nil {
					item.BackendPid = int32(s)
				}

				logLine, _ := logParts["message"].(string)
				logLineParts := logLinePartsRegexp.FindStringSubmatch(logLine)
				if len(logLineParts) != 0 {
					if s, err := strconv.ParseInt(logLineParts[1], 10, 32); err == nil {
						item.LogLineNumber = int32(s)
					}
					if s, err := strconv.ParseInt(logLineParts[2], 10, 32); err == nil {
						item.LogLineNumberChunk = int32(s)
					}
					item.Line = logLineParts[3]
				} else {
					item.Line = logLine

					logLineNumberStr, _ := logParts["structured_data"].(string)
					logLineNumberParts := logLineNumberPartsRegexp.FindStringSubmatch(logLineNumberStr)
					if len(logLineNumberParts) != 0 {
						if s, err := strconv.ParseInt(logLineNumberParts[1], 10, 32); err == nil {
							item.LogLineNumber = int32(s)
						}
						if s, err := strconv.ParseInt(logLineNumberParts[2], 10, 32); err == nil {
							item.LogLineNumberChunk = int32(s)
						}
					}
				}

				out <- item

				// TODO: Support using the same syslog server for different source Postgres servers,
				// and disambiguate based on logParts["client"]
			case <-ctx.Done():
				server.Kill()
				break
			}
		}
	}(ctx, server, channel)

	return nil
}
