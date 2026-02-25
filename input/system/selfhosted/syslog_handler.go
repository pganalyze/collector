package selfhosted

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/mcuadros/go-syslog.v2"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

var logLinePartsRegexp = regexp.MustCompile(`^\s*\[(\d+)-(\d+)\] (.*)`)
var logLineNumberPartsRegexp = regexp.MustCompile(`^\[(\d+)-(\d+)\]$`)

func setupSyslogHandler(ctx context.Context, config config.ServerConfig, out chan<- SelfHostedLogStreamItem, prefixedLogger *util.Logger, opts state.CollectionOpts) error {
	logSyslogServer := config.LogSyslogServer
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	// Peer name verification is already handled by crypto/tls and is not required at the go-syslog level
	// The defaultTlsPeerName in go-syslog can lead to false verification failures, so set nil to bypass
	server.SetTlsPeerNameFunc(nil)

	if config.LogSyslogServerCertFile != "" {
		serverCaPool := x509.NewCertPool()
		clientCaPool := x509.NewCertPool()
		if config.LogSyslogServerCAFile != "" {
			ca, err := os.ReadFile(config.LogSyslogServerCAFile)
			if err != nil {
				return fmt.Errorf("failed to read a Certificate Authority: %s", err)
			}
			if ok := serverCaPool.AppendCertsFromPEM(ca); !ok {
				return fmt.Errorf("failed to append a Certificate Authority")
			}
		}
		if config.LogSyslogServerClientCAFile != "" {
			ca, err := os.ReadFile(config.LogSyslogServerClientCAFile)
			if err != nil {
				return fmt.Errorf("failed to read a client Certificate Authority: %s", err)
			}
			if ok := clientCaPool.AppendCertsFromPEM(ca); !ok {
				return fmt.Errorf("failed to append a client Certificate Authority")
			}
		}

		cert, err := os.ReadFile(config.LogSyslogServerCertFile)
		if err != nil {
			return fmt.Errorf("failed to read a certificate: %s", err)
		}
		key, err := os.ReadFile(config.LogSyslogServerKeyFile)
		if err != nil {
			return fmt.Errorf("failed to read a key: %s", err)
		}
		tlsCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return err
		}

		tlsConfig := tls.Config{
			ClientAuth:   tls.VerifyClientCertIfGiven,
			Certificates: []tls.Certificate{tlsCert},
			RootCAs:      serverCaPool,
			ClientCAs:    clientCaPool,
		}
		err = server.ListenTCPTLS(logSyslogServer, &tlsConfig)
		if err != nil {
			return err
		}
	} else {
		err := server.ListenTCP(logSyslogServer)
		if err != nil {
			return err
		}
		err = server.ListenUDP(logSyslogServer)
		if err != nil {
			return err
		}
	}

	server.Boot()

	go func(ctx context.Context, server *syslog.Server, channel syslog.LogPartsChannel) {
		for {
			select {
			case logParts := <-channel:
				if opts.VeryVerbose {
					jsonData, err := json.MarshalIndent(logParts, "", "  ")
					if err != nil {
						prefixedLogger.PrintVerbose("Failed to convert LogParts struct to JSON: %v", err)
					}
					prefixedLogger.PrintVerbose("Received syslog log data in the following format:\n")
					prefixedLogger.PrintVerbose(string(jsonData))
				}
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
				return
			}
		}
	}(ctx, server, channel)

	return nil
}
